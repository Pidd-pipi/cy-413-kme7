package repository

import (
	"errors"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
)

type MoodCheckInRepository interface {
	Create(*model.MoodCheckIn) error
	ByTriggerDate(uint, time.Time) (*model.MoodCheckIn, error)
	ByID(uint, uint) (*model.MoodCheckIn, error)
	List(uint, *time.Time, *time.Time, string) ([]model.MoodCheckIn, error)
	Update(*model.MoodCheckIn) error
	SubmitResult(uid, id uint, result, note string) (bool, error)
}

type moodCheckInRepository struct{ db *gorm.DB }

func NewMoodCheckInRepository(db *gorm.DB) MoodCheckInRepository { return &moodCheckInRepository{db} }

func (r *moodCheckInRepository) Create(v *model.MoodCheckIn) error { return r.db.Create(v).Error }

func (r *moodCheckInRepository) ByTriggerDate(uid uint, day time.Time) (*model.MoodCheckIn, error) {
	var v model.MoodCheckIn
	start := day.Truncate(24 * time.Hour)
	e := r.db.Where("user_id = ? AND trigger_date >= ? AND trigger_date < ?", uid, start, start.AddDate(0, 0, 1)).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *moodCheckInRepository) ByID(id, uid uint) (*model.MoodCheckIn, error) {
	var v model.MoodCheckIn
	e := r.db.Where("id = ? AND user_id = ?", id, uid).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

// List 返回用户的回访；date 给出时按回访日过滤，triggerDate 给出时按低落触发日过滤，status 非空时按状态过滤。
func (r *moodCheckInRepository) List(uid uint, date, triggerDate *time.Time, status string) (out []model.MoodCheckIn, e error) {
	q := r.db.Where("user_id = ?", uid)
	if date != nil {
		start := date.Truncate(24 * time.Hour)
		q = q.Where("check_in_date >= ? AND check_in_date < ?", start, start.AddDate(0, 0, 1))
	}
	if triggerDate != nil {
		start := triggerDate.Truncate(24 * time.Hour)
		q = q.Where("trigger_date >= ? AND trigger_date < ?", start, start.AddDate(0, 0, 1))
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	e = q.Order("check_in_date desc, id desc").Find(&out).Error
	return
}

func (r *moodCheckInRepository) Update(v *model.MoodCheckIn) error { return r.db.Save(v).Error }

// SubmitResult 用条件 UPDATE 原子地把待回访标记为本人结果：只有 status=pending 的行能被命中。
// 返回 applied=false 表示回访不存在或已被提交/撤销，调用方据此保证重复及并发提交只保留一条。
func (r *moodCheckInRepository) SubmitResult(uid, id uint, result, note string) (bool, error) {
	res := r.db.Model(&model.MoodCheckIn{}).
		Where("id = ? AND user_id = ? AND status = ?", id, uid, "pending").
		Updates(map[string]any{"status": result, "result_note": note, "responded_at": gorm.Expr("NOW()"), "updated_at": gorm.Expr("NOW()")})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
