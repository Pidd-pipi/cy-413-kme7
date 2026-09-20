package model

import "time"

// FollowUp 低落情绪的次日回访。
// 同一用户同一触发日在数据库层面只允许一条有效记录（uniqueIndex 作用于 user_id + trigger_date）。
// 状态机：pending（待回访）-> responded（本人已确认结果，终态）/ revoked（情绪指数回升，已撤销，终态）。
type FollowUp struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"uniqueIndex:uniq_followup_user_trigger;index;not null" json:"user_id"`
	TriggerDate   time.Time  `gorm:"type:date;uniqueIndex:uniq_followup_user_trigger;not null" json:"trigger_date"`
	ScheduledDate time.Time  `gorm:"type:date;index;not null" json:"scheduled_date"`
	Source        string     `gorm:"size:32;not null" json:"source"`
	FirstMoodID   uint       `gorm:"index" json:"first_mood_id"`
	Status        string     `gorm:"size:16;index;not null" json:"status"`
	Result        string     `gorm:"size:16" json:"result"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
