package service

import (
	"encoding/json"
	"fmt"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"gorm.io/gorm"
	"log/slog"
	"strings"
	"time"
)

type MoodService struct {
	repo       repository.MoodRepository
	tx         *repository.TxManager
	reconciler FollowUpReconciler
	logger     *slog.Logger
}

func NewMoodService(r repository.MoodRepository, tx *repository.TxManager, rec FollowUpReconciler, l *slog.Logger) *MoodService {
	return &MoodService{repo: r, tx: tx, reconciler: rec, logger: l}
}
func validTags(tags []string) bool {
	allowed := map[string]bool{}
	for _, v := range constants.MoodTags {
		allowed[v] = true
	}
	for _, v := range tags {
		if !allowed[v] {
			return false
		}
	}
	return true
}

// normalizeSource 空来源默认来自情绪记录页；非法来源由 validator 提前拦截。
func normalizeSource(source string) string {
	if strings.TrimSpace(source) == "" {
		return constants.FollowUpSourceMoodList
	}
	return source
}

func (s *MoodService) Create(uid uint, req dto.MoodRequest) (*model.Mood, error) {
	if !validTags(req.MoodTags) {
		return nil, util.NewAppError(constants.CodeValidation, "Mood[mood_tags] create failed: unsupported tag", nil)
	}
	d, e := time.Parse("2006-01-02", req.RecordDate)
	if e != nil {
		return nil, util.NewAppError(constants.CodeValidation, "Mood[record_date] create failed: invalid date", e)
	}
	b, _ := json.Marshal(req.MoodTags)
	v := &model.Mood{UserID: uid, MoodLevel: req.MoodLevel, MoodTags: string(b), Note: strings.TrimSpace(req.Note), RecordDate: d}
	source := normalizeSource(req.Source)
	if e = s.tx.Run(func(tx *gorm.DB) error {
		// "用户+日期"咨询锁：即使当天还没有情绪行，同日并发记录也会串行，
		// 保证低落回访只合并为一条；事务结束自动释放。
		if e := repository.LockUserDay(tx, uid, d); e != nil {
			return fmt.Errorf("Mood[record_date] lock failed: %w", e)
		}
		if e := s.repo.CreateTx(tx, v); e != nil {
			return fmt.Errorf("Mood[user_id] create failed: %w", e)
		}
		return s.reconciler.Reconcile(tx, uid, d, source)
	}); e != nil {
		return nil, e
	}
	s.logger.Info(constants.LogMoodCreated, "user_id", uid, "mood_level", v.MoodLevel, "source", source)
	return v, nil
}
func (s *MoodService) List(uid uint, date string) ([]model.Mood, error) {
	var d *time.Time
	if date != "" {
		x, e := time.Parse("2006-01-02", date)
		if e != nil {
			return nil, util.NewAppError(constants.CodeValidation, "Mood[record_date] list failed: invalid date", e)
		}
		d = &x
	}
	vs, e := s.repo.List(uid, d)
	if e != nil {
		return nil, fmt.Errorf("Mood[user_id] list failed: %w", e)
	}
	s.logger.Info(constants.LogMoodListed, "user_id", uid)
	return vs, nil
}
func (s *MoodService) Update(uid, id uint, req dto.MoodRequest) (*model.Mood, error) {
	v, e := s.repo.ByID(id, uid)
	if e != nil {
		return nil, fmt.Errorf("Mood[id=%d] fetch failed: %w", id, e)
	}
	if !validTags(req.MoodTags) {
		return nil, util.WrapEntity("Mood", "mood_tags", id, constants.CodeValidation, nil)
	}
	d, e := time.Parse("2006-01-02", req.RecordDate)
	if e != nil {
		return nil, util.WrapEntity("Mood", "record_date", id, constants.CodeValidation, e)
	}
	b, _ := json.Marshal(req.MoodTags)
	oldDay := v.RecordDate
	newDay := d
	source := normalizeSource(req.Source)
	if e = s.tx.Run(func(tx *gorm.DB) error {
		cur, e := s.repo.ByIDForUpdate(tx, id, uid)
		if e != nil {
			return fmt.Errorf("Mood[id=%d] fetch failed: %w", id, e)
		}
		oldDay = cur.RecordDate
		// 按日期升序获取咨询锁，避免跨日修改时多个事务相互等待形成死锁。
		for _, day := range sortedDays(oldDay, newDay) {
			if e := repository.LockUserDay(tx, uid, day); e != nil {
				return fmt.Errorf("Mood[record_date] lock failed: %w", e)
			}
		}
		cur.MoodLevel = req.MoodLevel
		cur.MoodTags = string(b)
		cur.Note = req.Note
		cur.RecordDate = d
		if e := s.repo.UpdateTx(tx, cur); e != nil {
			return util.WrapEntity("Mood", "mood_level", id, constants.CodeInternal, e)
		}
		v = cur
		// 指数回升撤销原触发日回访；落到新日期时，新日期按当前来源参与联动。
		if e := s.reconciler.Reconcile(tx, uid, oldDay, source); e != nil {
			return e
		}
		if !sameDay(oldDay, newDay) {
			if e := s.reconciler.Reconcile(tx, uid, newDay, source); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		return nil, e
	}
	s.logger.Info(constants.LogMoodUpdated, "mood_id", id)
	return v, nil
}
func (s *MoodService) Delete(uid, id uint) error {
	if _, e := s.repo.ByID(id, uid); e != nil {
		return fmt.Errorf("Mood[id=%d] fetch failed: %w", id, e)
	}
	if e := s.tx.Run(func(tx *gorm.DB) error {
		cur, e := s.repo.ByIDForUpdate(tx, id, uid)
		if e != nil {
			return fmt.Errorf("Mood[id=%d] fetch failed: %w", id, e)
		}
		if e := repository.LockUserDay(tx, uid, cur.RecordDate); e != nil {
			return fmt.Errorf("Mood[record_date] lock failed: %w", e)
		}
		if e := s.repo.DeleteTx(tx, cur); e != nil {
			return util.WrapEntity("Mood", "id", id, constants.CodeInternal, e)
		}
		// 删除后当天若无剩余低落记录，待回访自动撤销；已提交结果不受影响。
		return s.reconciler.Reconcile(tx, uid, cur.RecordDate, "")
	}); e != nil {
		return e
	}
	s.logger.Info(constants.LogMoodDeleted, "mood_id", id)
	return nil
}

func sameDay(a, b time.Time) bool {
	return a.Truncate(24 * time.Hour).Equal(b.Truncate(24 * time.Hour))
}

// sortedDays 返回去重后按升序排列的日期，用于固定咨询锁获取顺序、避免死锁。
func sortedDays(days ...time.Time) []time.Time {
	uniq := make([]time.Time, 0, len(days))
	for _, d := range days {
		ds := d.Truncate(24 * time.Hour)
		exists := false
		for _, u := range uniq {
			if u.Equal(ds) {
				exists = true
				break
			}
		}
		if !exists {
			uniq = append(uniq, ds)
		}
	}
	for i := 0; i < len(uniq); i++ {
		for j := i + 1; j < len(uniq); j++ {
			if uniq[j].Before(uniq[i]) {
				uniq[i], uniq[j] = uniq[j], uniq[i]
			}
		}
	}
	return uniq
}
