package service

import (
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/model"
)

func sampleDay() time.Time { return time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC) }

func lowMood(id, uid uint, level int) model.Mood {
	return model.Mood{ID: id, UserID: uid, MoodLevel: level, RecordDate: sampleDay()}
}

func TestReconcileState(t *testing.T) {
	day := sampleDay()
	cases := []struct {
		name       string
		existing   *model.FollowUp
		lowMoods   []model.Mood
		wantAction followUpAction
		wantStatus string
		keepSource string // 期望保留的首次来源
		keepMoodID uint
	}{
		{
			name:       "low mood with no follow-up schedules next-day pending",
			lowMoods:   []model.Mood{lowMood(1, 7, 2)},
			wantAction: followUpCreate,
			wantStatus: constants.FollowUpStatusPending,
			keepSource: constants.FollowUpSourceMoodList,
			keepMoodID: 1,
		},
		{
			name:       "healthy mood with no follow-up does nothing",
			lowMoods:   nil,
			wantAction: followUpNone,
		},
		{
			name:       "second low mood same day only merges and keeps first source",
			existing:   &model.FollowUp{ID: 9, UserID: 7, TriggerDate: day, Status: constants.FollowUpStatusPending, Source: constants.FollowUpSourceMood, FirstMoodID: 1},
			lowMoods:   []model.Mood{lowMood(1, 7, 3), lowMood(2, 7, 2)},
			wantAction: followUpMerge,
			wantStatus: constants.FollowUpStatusPending,
			keepSource: constants.FollowUpSourceMood,
			keepMoodID: 1,
		},
		{
			name:       "level recovers revokes pending follow-up",
			existing:   &model.FollowUp{ID: 9, UserID: 7, TriggerDate: day, Status: constants.FollowUpStatusPending, Source: constants.FollowUpSourceMoodList, FirstMoodID: 1},
			lowMoods:   nil,
			wantAction: followUpRevoke,
			wantStatus: constants.FollowUpStatusRevoked,
			keepSource: constants.FollowUpSourceMoodList,
			keepMoodID: 1,
		},
		{
			name:       "low mood again after revoke reopens and keeps first source",
			existing:   &model.FollowUp{ID: 9, UserID: 7, TriggerDate: day, Status: constants.FollowUpStatusRevoked, Source: constants.FollowUpSourceMood, FirstMoodID: 1},
			lowMoods:   []model.Mood{lowMood(3, 7, 1)},
			wantAction: followUpReopen,
			wantStatus: constants.FollowUpStatusPending,
			keepSource: constants.FollowUpSourceMood,
			keepMoodID: 1,
		},
		{
			name:       "staying healthy after revoke does nothing",
			existing:   &model.FollowUp{ID: 9, UserID: 7, TriggerDate: day, Status: constants.FollowUpStatusRevoked},
			lowMoods:   nil,
			wantAction: followUpNone,
			wantStatus: constants.FollowUpStatusRevoked,
		},
		{
			name:       "submitted result is preserved when later moods change to healthy",
			existing:   &model.FollowUp{ID: 9, UserID: 7, TriggerDate: day, Status: constants.FollowUpStatusResponded, Result: constants.FollowUpResultStruggling, Source: constants.FollowUpSourceMood, FirstMoodID: 1},
			lowMoods:   nil,
			wantAction: followUpPreserve,
			wantStatus: constants.FollowUpStatusResponded,
			keepSource: constants.FollowUpSourceMood,
			keepMoodID: 1,
		},
		{
			name:       "submitted result is preserved even when low moods remain",
			existing:   &model.FollowUp{ID: 9, UserID: 7, TriggerDate: day, Status: constants.FollowUpStatusResponded, Result: constants.FollowUpResultBetter},
			lowMoods:   []model.Mood{lowMood(1, 7, 2)},
			wantAction: followUpPreserve,
			wantStatus: constants.FollowUpStatusResponded,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			plan := reconcileState(copyFollowUp(tt.existing), tt.lowMoods, day, constants.FollowUpSourceMoodList)
			if plan.Action != tt.wantAction {
				t.Fatalf("action=%s want=%s", plan.Action, tt.wantAction)
			}
			if tt.wantStatus != "" {
				if plan.FollowUp == nil || plan.FollowUp.Status != tt.wantStatus {
					t.Fatalf("status=%v want=%s", plan.FollowUp, tt.wantStatus)
				}
			}
			if tt.keepSource != "" && plan.FollowUp.Source != tt.keepSource {
				t.Fatalf("source=%s want=%s", plan.FollowUp.Source, tt.keepSource)
			}
			if tt.keepMoodID != 0 && plan.FollowUp.FirstMoodID != tt.keepMoodID {
				t.Fatalf("first_mood_id=%d want=%d", plan.FollowUp.FirstMoodID, tt.keepMoodID)
			}
			if tt.wantAction == followUpCreate {
				if !plan.FollowUp.ScheduledDate.Equal(day.AddDate(0, 0, 1)) {
					t.Fatalf("scheduled_date=%s want next day of %s", plan.FollowUp.ScheduledDate, day)
				}
			}
		})
	}
}

func copyFollowUp(f *model.FollowUp) *model.FollowUp {
	if f == nil {
		return nil
	}
	cp := *f
	return &cp
}

func TestThresholdIsInclusive(t *testing.T) {
	// 阈值固定为 <=3：3 触发，4 不触发。
	if constants.LowMoodLevelThreshold != 3 {
		t.Fatalf("threshold=%d want=3", constants.LowMoodLevelThreshold)
	}
}
