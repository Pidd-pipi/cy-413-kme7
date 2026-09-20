package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
)

type MoodCheckInService struct {
	repo   repository.MoodCheckInRepository
	moods  repository.MoodRepository
	logger *slog.Logger
}

func NewMoodCheckInService(r repository.MoodCheckInRepository, m repository.MoodRepository, l *slog.Logger) *MoodCheckInService {
	return &MoodCheckInService{repo: r, moods: m, logger: l}
}

// normalizeSource 记录首次触发来源；未知来源归入情绪记录页，保证老数据与异常调用有稳定取值。
func normalizeSource(source string) string {
	if source == constants.CheckInSourceDashboard {
		return constants.CheckInSourceDashboard
	}
	return constants.CheckInSourceMoods
}

// Reconcile 在任意情绪写操作后，按用户某天是否仍有低落记录对齐回访状态，是低落情绪回访闭环的唯一入口。
//   - 当天存在低落记录且无回访：生成次日回访，记录首次触发来源；
//   - 同日再次低落：合并进已有待回访，来源与首次指数保持不变；
//   - 指数回升（不再有低落记录）：撤销待回访；撤销后再次低落则重新激活；
//   - 已提交结果的回访永久冻结，任何后续修改都不再影响它。
func (s *MoodCheckInService) Reconcile(uid uint, day time.Time, source string, level int) (*model.MoodCheckIn, error) {
	day = day.Truncate(24 * time.Hour)
	count, e := s.moods.CountLow(uid, day, constants.LowMoodLevel)
	if e != nil {
		return nil, fmt.Errorf("MoodCheckIn[trigger_date=%s] reconcile failed: %w", day.Format("2006-01-02"), e)
	}
	current, e := s.repo.ByTriggerDate(uid, day)
	if e != nil && !errors.Is(e, repository.ErrNotFound) {
		return nil, fmt.Errorf("MoodCheckIn[trigger_date=%s] fetch failed: %w", day.Format("2006-01-02"), e)
	}

	if count > 0 {
		return s.schedule(uid, day, normalizeSource(source), level, current)
	}
	return s.revoke(current)
}

func (s *MoodCheckInService) schedule(uid uint, day time.Time, source string, level int, current *model.MoodCheckIn) (*model.MoodCheckIn, error) {
	if current != nil {
		switch current.Status {
		case constants.CheckInStatusPending:
			// 同日再次记录低落：只合并一次，首次触发来源与首次低落指数不变。
			s.logger.Info(constants.LogCheckInMerged, "user_id", uid, "trigger_date", day.Format("2006-01-02"))
			return current, nil
		case constants.CheckInStatusImproved, constants.CheckInStatusStillTroubled:
			// 已提交结果不受后续修改影响。
			return current, nil
		case constants.CheckInStatusRevoked:
			// 撤销后当天再次低落：重新激活，本轮首次触发来源更新为当前入口。
			now := time.Now()
			current.Status = constants.CheckInStatusPending
			current.Source = source
			if level > 0 {
				current.MoodLevel = level
			}
			current.RevokedAt = nil
			current.UpdatedAt = now
			if e := s.repo.Update(current); e != nil {
				return nil, fmt.Errorf("MoodCheckIn[id=%d] reactivate failed: %w", current.ID, e)
			}
			s.logger.Info(constants.LogCheckInReactivated, "checkin_id", current.ID, "source", source)
			return current, nil
		}
	}

	v := &model.MoodCheckIn{
		UserID:      uid,
		TriggerDate: day,
		CheckInDate: day.AddDate(0, 0, 1),
		Status:      constants.CheckInStatusPending,
		Source:      source,
		MoodLevel:   level,
	}
	if e := s.repo.Create(v); e != nil {
		// 并发创建时唯一索引兜底：重读已有记录，按“合并一次”继续对齐，而不是报错。
		raced, re := s.repo.ByTriggerDate(uid, day)
		if re != nil {
			return nil, fmt.Errorf("MoodCheckIn[user_id] create failed: %w", e)
		}
		s.logger.Info(constants.LogCheckInMerged, "user_id", uid, "trigger_date", day.Format("2006-01-02"))
		return raced, nil
	}
	s.logger.Info(constants.LogCheckInScheduled, "checkin_id", v.ID, "user_id", uid, "source", source, "checkin_date", v.CheckInDate.Format("2006-01-02"))
	return v, nil
}

func (s *MoodCheckInService) revoke(current *model.MoodCheckIn) (*model.MoodCheckIn, error) {
	if current == nil || current.Status != constants.CheckInStatusPending {
		return current, nil
	}
	now := time.Now()
	current.Status = constants.CheckInStatusRevoked
	current.RevokedAt = &now
	current.UpdatedAt = now
	if e := s.repo.Update(current); e != nil {
		return nil, fmt.Errorf("MoodCheckIn[id=%d] revoke failed: %w", current.ID, e)
	}
	s.logger.Info(constants.LogCheckInRevoked, "checkin_id", current.ID)
	return current, nil
}

// Respond 仅允许本人在回访处于待回访时确认好转或仍困扰；重复及并发提交只保留第一条。
func (s *MoodCheckInService) Respond(uid, id uint, req dto.CheckInRespondRequest) (*model.MoodCheckIn, bool, error) {
	applied, e := s.repo.SubmitResult(uid, id, req.Result, strings.TrimSpace(req.Note))
	if e != nil {
		return nil, false, fmt.Errorf("MoodCheckIn[id=%d] respond failed: %w", id, e)
	}
	if !applied {
		// 回访不存在、已撤销或已被提交：取回现有结果做幂等/冲突判定。
		existing, fe := s.repo.ByID(id, uid)
		if fe != nil {
			if errors.Is(fe, repository.ErrNotFound) {
				return nil, false, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("MoodCheckIn[id=%d] respond failed: not found", id), fe)
			}
			return nil, false, fmt.Errorf("MoodCheckIn[id=%d] fetch failed: %w", id, fe)
		}
		if existing.Status == req.Result {
			// 完全重复的提交：幂等返回已有结果，不新增记录。
			return existing, true, nil
		}
		s.logger.Warn(constants.LogCheckInConflict, "checkin_id", id, "existing", existing.Status, "incoming", req.Result)
		return existing, false, util.NewAppError(constants.CodeConflict, fmt.Sprintf("MoodCheckIn[id=%d] respond failed: %s", id, constants.MessageCheckInConflict), nil)
	}
	v, e := s.repo.ByID(id, uid)
	if e != nil {
		return nil, false, fmt.Errorf("MoodCheckIn[id=%d] fetch after respond failed: %w", id, e)
	}
	s.logger.Info(constants.LogCheckInResponded, "checkin_id", id, "result", req.Result)
	return v, true, nil
}

func (s *MoodCheckInService) List(uid uint, date, triggerDate, status string) ([]model.MoodCheckIn, error) {
	if status != "" && status != constants.CheckInStatusPending && status != constants.CheckInStatusImproved && status != constants.CheckInStatusStillTroubled && status != constants.CheckInStatusRevoked {
		return nil, util.NewAppError(constants.CodeValidation, "MoodCheckIn[status] list failed: unsupported status", nil)
	}
	var d *time.Time
	if date != "" {
		parsed, e := time.Parse("2006-01-02", date)
		if e != nil {
			return nil, util.NewAppError(constants.CodeValidation, "MoodCheckIn[checkin_date] list failed: invalid date", e)
		}
		d = &parsed
	}
	var td *time.Time
	if triggerDate != "" {
		parsed, e := time.Parse("2006-01-02", triggerDate)
		if e != nil {
			return nil, util.NewAppError(constants.CodeValidation, "MoodCheckIn[trigger_date] list failed: invalid date", e)
		}
		td = &parsed
	}
	vs, e := s.repo.List(uid, d, td, status)
	if e != nil {
		return nil, fmt.Errorf("MoodCheckIn[user_id] list failed: %w", e)
	}
	s.logger.Info(constants.LogCheckInListed, "user_id", uid)
	return vs, nil
}
