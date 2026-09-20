package model

import "time"

// MoodCheckIn 低落情绪次日回访。
//
// 闭环规则：
//   - 用户某天（TriggerDate）存在心情指数 <= 低落阈值的情绪记录时，生成次日（CheckInDate）回访；
//   - (user_id, trigger_date) 唯一，同日再次记录低落只合并到同一条，首次触发来源保持不变；
//   - 当天指数回升（不再有低落记录）时，待回访记录被撤销；低落再次出现可重新激活；
//   - 一旦本人提交结果（improved / still_troubled），记录冻结，不再受后续情绪修改影响。
type MoodCheckIn struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"uniqueIndex:uniq_user_trigger_date;not null" json:"user_id"`
	TriggerDate time.Time  `gorm:"type:date;uniqueIndex:uniq_user_trigger_date;not null" json:"trigger_date"`
	CheckInDate time.Time  `gorm:"type:date;index;not null" json:"check_in_date"`
	Status      string     `gorm:"size:20;index;not null;default:pending" json:"status"`
	Source      string     `gorm:"size:20;not null;default:moods" json:"source"`
	MoodLevel   int        `gorm:"not null;default:0" json:"mood_level"`
	ResultNote  string     `gorm:"size:500" json:"result_note"`
	RespondedAt *time.Time `json:"responded_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
