package constants

// LowMoodLevelThreshold 心情指数不高于该值即视为低落，触发次日回访。
const LowMoodLevelThreshold = 3

const (
	// FollowUpSourceMood 触发来源：心情花园快速记录（Dashboard）
	FollowUpSourceMood = "mood"
	// FollowUpSourceMoodList 触发来源：情绪记录页（Moods）
	FollowUpSourceMoodList = "mood_list"
)

var FollowUpSources = []string{FollowUpSourceMood, FollowUpSourceMoodList}

const (
	FollowUpStatusPending   = "pending"
	FollowUpStatusResponded = "responded"
	FollowUpStatusRevoked   = "revoked"
)

var FollowUpStatuses = []string{FollowUpStatusPending, FollowUpStatusResponded, FollowUpStatusRevoked}

const (
	// FollowUpResultBetter 本人确认：已好转
	FollowUpResultBetter = "better"
	// FollowUpResultStruggling 本人确认：仍困扰
	FollowUpResultStruggling = "struggling"
)

var FollowUpResults = []string{FollowUpResultBetter, FollowUpResultStruggling}
