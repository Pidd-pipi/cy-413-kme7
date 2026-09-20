package repository

import (
	"errors"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type MoodRepository interface {
	Create(*model.Mood) error
	List(uint, *time.Time) ([]model.Mood, error)
	ByID(uint, uint) (*model.Mood, error)
	Update(*model.Mood) error
	Delete(*model.Mood) error

	// 以下方法在事务（*gorm.DB）内调用，保证情绪写入与回访联动原子完成。
	CreateTx(tx *gorm.DB, v *model.Mood) error
	ByIDForUpdate(tx *gorm.DB, id, uid uint) (*model.Mood, error)
	UpdateTx(tx *gorm.DB, v *model.Mood) error
	DeleteTx(tx *gorm.DB, v *model.Mood) error
	// LowMoodsByDayTx 返回触发日当天心情指数不高于阈值的情绪记录。
	// 调用方已持有"用户+日期"咨询锁，此处无需再加行锁。
	LowMoodsByDayTx(tx *gorm.DB, uid uint, day time.Time) ([]model.Mood, error)
}
type moodRepository struct{ db *gorm.DB }

func NewMoodRepository(db *gorm.DB) MoodRepository   { return &moodRepository{db} }
func (r *moodRepository) Create(v *model.Mood) error { return r.db.Create(v).Error }
func (r *moodRepository) List(uid uint, date *time.Time) (out []model.Mood, e error) {
	q := r.db.Where("user_id = ?", uid)
	if date != nil {
		q = q.Where("record_date >= ? AND record_date < ?", date.Truncate(24*time.Hour), date.Truncate(24*time.Hour).AddDate(0, 0, 1))
	}
	e = q.Order("record_date desc, id desc").Find(&out).Error
	return
}
func (r *moodRepository) ByID(id, uid uint) (*model.Mood, error) {
	var v model.Mood
	e := r.db.Where("id = ? AND user_id = ?", id, uid).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}
func (r *moodRepository) Update(v *model.Mood) error { return r.db.Save(v).Error }
func (r *moodRepository) Delete(v *model.Mood) error { return r.db.Delete(v).Error }

func (r *moodRepository) CreateTx(tx *gorm.DB, v *model.Mood) error { return tx.Create(v).Error }

func (r *moodRepository) ByIDForUpdate(tx *gorm.DB, id, uid uint) (*model.Mood, error) {
	var v model.Mood
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ?", id, uid).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *moodRepository) UpdateTx(tx *gorm.DB, v *model.Mood) error { return tx.Save(v).Error }
func (r *moodRepository) DeleteTx(tx *gorm.DB, v *model.Mood) error { return tx.Delete(v).Error }

func (r *moodRepository) LowMoodsByDayTx(tx *gorm.DB, uid uint, day time.Time) (out []model.Mood, e error) {
	start, end := dayRange(day)
	e = tx.Where("user_id = ? AND record_date >= ? AND record_date < ? AND mood_level <= ?", uid, start, end, constants.LowMoodLevelThreshold).
		Order("id ASC").
		Find(&out).Error
	return
}
