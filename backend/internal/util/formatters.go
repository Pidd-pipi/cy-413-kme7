package util

import (
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"time"
)

func FormatDate(v time.Time) string { return v.Format("2006-01-02") }
func MoodText(tag string) string {
	m := map[string]string{constants.MoodHappy: "开心", constants.MoodAnxious: "焦虑", constants.MoodTired: "疲惫", constants.MoodAngry: "愤怒", constants.MoodCalm: "平静"}
	return m[tag]
}
func AssessmentText(c string) string {
	m := map[string]string{constants.AssessmentAnxiety: "焦虑", constants.AssessmentDepression: "抑郁", constants.AssessmentStress: "压力", constants.AssessmentSleep: "睡眠"}
	return m[c]
}
func ThemeColor(t string) string { return constants.ThemeColors[t] }

func FollowUpStatusText(s string) string {
	m := map[string]string{constants.FollowUpStatusPending: "待回访", constants.FollowUpStatusResponded: "已回访", constants.FollowUpStatusRevoked: "已撤销"}
	return m[s]
}
func FollowUpResultText(r string) string {
	m := map[string]string{constants.FollowUpResultBetter: "已好转", constants.FollowUpResultStruggling: "仍困扰"}
	return m[r]
}
func FollowUpSourceText(src string) string {
	m := map[string]string{constants.FollowUpSourceMood: "心情花园", constants.FollowUpSourceMoodList: "情绪记录"}
	return m[src]
}
