package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"gorm.io/gorm"
)

// followUpAction 描述状态机对某触发日回访的处置。
type followUpAction string

const (
	followUpNone     followUpAction = "none"     // 无需变更
	followUpCreate   followUpAction = "create"   // 新建次日回访
	followUpMerge    followUpAction = "merge"    // 同日再次低落：只合并，保留首次来源
	followUpRevoke   followUpAction = "revoke"   // 指数回升：撤销待回访
	followUpReopen   followUpAction = "reopen"   // 撤销后再次低落：重新挂起，保留首次来源
	followUpPreserve followUpAction = "preserve" // 已提交结果：后续修改不再影响
)

// reconcilePlan 是纯函数 reconcileState 的输出，便于表驱动单测。
type reconcilePlan struct {
	Action followUpAction
	// 待持久化的回访（create 为新对象，reopen/revoke 为更新后的既有对象）
	FollowUp *model.FollowUp
}

// reconcileState 根据触发日当前是否仍有低落记录、既有回访状态，决定闭环动作。
// 已有回访（含 revoked）始终保留首次触发来源 source 与 first_mood_id；
// 已 responded 的回访结果不受后续情绪修改影响。
func reconcileState(existing *model.FollowUp, lowMoods []model.Mood, triggerDate time.Time, source string) reconcilePlan {
	hasLow := len(lowMoods) > 0
	if existing != nil {
		switch existing.Status {
		case constants.FollowUpStatusResponded:
			return reconcilePlan{Action: followUpPreserve, FollowUp: existing}
		case constants.FollowUpStatusRevoked:
			if hasLow {
				existing.Status = constants.FollowUpStatusPending
				existing.Result = ""
				existing.RespondedAt = nil
				return reconcilePlan{Action: followUpReopen, FollowUp: existing}
			}
			return reconcilePlan{Action: followUpNone, FollowUp: existing}
		default: // pending
			if hasLow {
				return reconcilePlan{Action: followUpMerge, FollowUp: existing}
			}
			existing.Status = constants.FollowUpStatusRevoked
			existing.Result = ""
			existing.RespondedAt = nil
			return reconcilePlan{Action: followUpRevoke, FollowUp: existing}
		}
	}
	if !hasLow {
		return reconcilePlan{Action: followUpNone}
	}
	return reconcilePlan{Action: followUpCreate, FollowUp: &model.FollowUp{
		UserID:        lowMoods[0].UserID,
		TriggerDate:   dayStart(triggerDate),
		ScheduledDate: dayStart(triggerDate).AddDate(0, 0, 1),
		Source:        source,
		FirstMoodID:   lowMoods[0].ID,
		Status:        constants.FollowUpStatusPending,
	}}
}

func dayStart(t time.Time) time.Time { return t.Truncate(24 * time.Hour) }

// FollowUpReconciler 是 MoodService 依赖的回访联动接口。
type FollowUpReconciler interface {
	Reconcile(tx *gorm.DB, uid uint, triggerDate time.Time, source string) error
}

type FollowUpService struct {
	repo   repository.FollowUpRepository
	moods  repository.MoodRepository
	logger *slog.Logger
}

func NewFollowUpService(r repository.FollowUpRepository, m repository.MoodRepository, l *slog.Logger) *FollowUpService {
	return &FollowUpService{repo: r, moods: m, logger: l}
}

// Reconcile 在情绪写入事务内执行：按当日低落记录与既有回访驱动状态机。
// 调用方（MoodService）已持有"用户+日期"咨询锁，因此并发的同日记录只会串行合并为一条。
func (s *FollowUpService) Reconcile(tx *gorm.DB, uid uint, triggerDate time.Time, source string) error {
	day := dayStart(triggerDate)
	lowMoods, e := s.moods.LowMoodsByDayTx(tx, uid, day)
	if e != nil {
		return fmt.Errorf("FollowUp[trigger_date] reconcile failed: %w", e)
	}
	existing, e := s.repo.ByTriggerDateForUpdate(tx, uid, day)
	if errors.Is(e, repository.ErrNotFound) {
		existing = nil
	} else if e != nil {
		return fmt.Errorf("FollowUp[user_id] reconcile failed: %w", e)
	}

	plan := reconcileState(existing, lowMoods, day, source)
	switch plan.Action {
	case followUpCreate:
		if e = s.repo.CreateTx(tx, plan.FollowUp); e != nil {
			return fmt.Errorf("FollowUp[first_mood_id] schedule failed: %w", e)
		}
		s.logger.Info(constants.LogFollowUpScheduled, "user_id", uid, "trigger_date", day.Format("2006-01-02"), "source", source, "first_mood_id", plan.FollowUp.FirstMoodID)
	case followUpMerge:
		s.logger.Info(constants.LogFollowUpMerged, "user_id", uid, "trigger_date", day.Format("2006-01-02"), "source", plan.FollowUp.Source)
	case followUpRevoke:
		if e = s.repo.UpdateTx(tx, plan.FollowUp); e != nil {
			return fmt.Errorf("FollowUp[status] revoke failed: %w", e)
		}
		s.logger.Info(constants.LogFollowUpRevoked, "user_id", uid, "trigger_date", day.Format("2006-01-02"))
	case followUpReopen:
		if e = s.repo.UpdateTx(tx, plan.FollowUp); e != nil {
			return fmt.Errorf("FollowUp[status] reopen failed: %w", e)
		}
		s.logger.Info(constants.LogFollowUpReopened, "user_id", uid, "trigger_date", day.Format("2006-01-02"), "source", plan.FollowUp.Source)
	case followUpPreserve:
		s.logger.Info(constants.LogFollowUpReconciled, "user_id", uid, "trigger_date", day.Format("2006-01-02"), "status", constants.FollowUpStatusResponded)
	}
	return nil
}

// Respond 仅允许本人提交一次回访结果；重复及并发提交只保留第一条。
func (s *FollowUpService) Respond(uid, id uint, result string) (*model.FollowUp, error) {
	ok, e := s.repo.RespondIfPending(uid, id, result, time.Now())
	if e != nil {
		return nil, fmt.Errorf("FollowUp[id=%d] respond failed: %w", id, e)
	}
	if !ok {
		// 区分不存在（404）与已处理（409），重复提交时把既有结果原样返回。
		f, fe := s.repo.ByID(uid, id)
		if fe != nil {
			return nil, fmt.Errorf("FollowUp[id=%d] fetch failed: %w", id, fe)
		}
		if f.Status == constants.FollowUpStatusRevoked {
			return nil, util.NewAppError(constants.CodeConflict, fmt.Sprintf("FollowUp[id=%d] respond failed: status revoked before response", id), nil)
		}
		today := dayStart(time.Now())
		if dayStart(f.ScheduledDate).After(today) {
			return nil, util.NewAppError(constants.CodeConflict, fmt.Sprintf("FollowUp[id=%d] respond failed: not due until %s", id, f.ScheduledDate.Format("2006-01-02")), nil)
		}
		s.logger.Info(constants.LogFollowUpDuplicate, "user_id", uid, "follow_up_id", id, "result", f.Result)
		return nil, &FollowUpConflictError{FollowUp: f}
	}
	s.logger.Info(constants.LogFollowUpResponded, "user_id", uid, "follow_up_id", id, "result", result)
	return s.repo.ByID(uid, id)
}

// List 查询当前用户的回访，可按状态与触发日过滤。
func (s *FollowUpService) List(uid uint, status, date string) ([]model.FollowUp, error) {
	if status != "" && !contains(constants.FollowUpStatuses, status) {
		return nil, util.NewAppError(constants.CodeValidation, "FollowUp[status] list failed: unsupported status", nil)
	}
	var d *time.Time
	if date != "" {
		parsed, e := time.Parse("2006-01-02", date)
		if e != nil {
			return nil, util.NewAppError(constants.CodeValidation, "FollowUp[trigger_date] list failed: invalid date", e)
		}
		d = &parsed
	}
	out, e := s.repo.List(uid, status, d)
	if e != nil {
		return nil, fmt.Errorf("FollowUp[user_id] list failed: %w", e)
	}
	s.logger.Info(constants.LogFollowUpListed, "user_id", uid, "status", status)
	return out, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
