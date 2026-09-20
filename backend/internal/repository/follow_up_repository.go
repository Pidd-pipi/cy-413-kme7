package repository

import (
	"errors"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type FollowUpRepository interface {
	Create(*model.FollowUp) error
	// CreateTx / UpdateTx 在情绪写入事务内联动回访状态。
	CreateTx(tx *gorm.DB, v *model.FollowUp) error
	UpdateTx(tx *gorm.DB, v *model.FollowUp) error
	// ByTriggerDate 返回某用户某触发日的回访；不存在返回 ErrNotFound。
	ByTriggerDate(uid uint, day time.Time) (*model.FollowUp, error)
	// ByID 返回属于该用户的回访；不存在返回 ErrNotFound。
	ByID(uid, id uint) (*model.FollowUp, error)
	// ByTriggerDateForUpdate 在事务内锁定目标行，无行时返回 ErrNotFound。
	ByTriggerDateForUpdate(tx *gorm.DB, uid uint, day time.Time) (*model.FollowUp, error)
	// ListTx 在事务内查询回访，可按状态与触发日过滤。
	ListTx(tx *gorm.DB, uid uint, status string, triggerDate *time.Time) ([]model.FollowUp, error)
	// List 查询回访列表，可按状态与触发日过滤；triggerDate 为 nil 时不过滤日期。
	List(uid uint, status string, triggerDate *time.Time) ([]model.FollowUp, error)
	// RespondIfPending 仅当回访属于本人、处于 pending 且已到回访日时写入结果（条件更新）。
	// 返回 affected==false 表示回访不存在、不属于本人、未到期或已被处理（重复/并发提交）。
	RespondIfPending(uid, id uint, result string, respondedAt time.Time) (bool, error)
}

type followUpRepository struct{ db *gorm.DB }

func NewFollowUpRepository(db *gorm.DB) FollowUpRepository { return &followUpRepository{db} }

func dayRange(d time.Time) (time.Time, time.Time) {
	start := d.Truncate(24 * time.Hour)
	return start, start.AddDate(0, 0, 1)
}

func (r *followUpRepository) Create(v *model.FollowUp) error {
	return r.db.Create(v).Error
}

func (r *followUpRepository) CreateTx(tx *gorm.DB, v *model.FollowUp) error {
	return tx.Create(v).Error
}
func (r *followUpRepository) UpdateTx(tx *gorm.DB, v *model.FollowUp) error { return tx.Save(v).Error }

func (r *followUpRepository) ByTriggerDate(uid uint, day time.Time) (*model.FollowUp, error) {
	var v model.FollowUp
	start, end := dayRange(day)
	e := r.db.Where("user_id = ? AND trigger_date >= ? AND trigger_date < ?", uid, start, end).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *followUpRepository) ByID(uid, id uint) (*model.FollowUp, error) {
	var v model.FollowUp
	e := r.db.Where("id = ? AND user_id = ?", id, uid).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *followUpRepository) ByTriggerDateForUpdate(tx *gorm.DB, uid uint, day time.Time) (*model.FollowUp, error) {
	var v model.FollowUp
	start, end := dayRange(day)
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND trigger_date >= ? AND trigger_date < ?", uid, start, end).
		First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *followUpRepository) List(uid uint, status string, triggerDate *time.Time) (out []model.FollowUp, e error) {
	return r.ListTx(r.db, uid, status, triggerDate)
}

func (r *followUpRepository) ListTx(tx *gorm.DB, uid uint, status string, triggerDate *time.Time) (out []model.FollowUp, e error) {
	q := tx.Where("user_id = ?", uid)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if triggerDate != nil {
		start, end := dayRange(*triggerDate)
		q = q.Where("trigger_date >= ? AND trigger_date < ?", start, end)
	}
	e = q.Order("trigger_date desc, id desc").Find(&out).Error
	return
}

func (r *followUpRepository) RespondIfPending(uid, id uint, result string, respondedAt time.Time) (bool, error) {
	day := respondedAt.Truncate(24 * time.Hour)
	res := r.db.Model(&model.FollowUp{}).
		Where("id = ? AND user_id = ? AND status = ? AND scheduled_date <= ?", id, uid, constants.FollowUpStatusPending, day).
		Updates(map[string]any{"status": "responded", "result": result, "responded_at": respondedAt})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
