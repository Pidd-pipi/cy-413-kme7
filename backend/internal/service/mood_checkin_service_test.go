package service

import (
	"errors"
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
)

// ---- 内存假仓储：与 GORM 仓储实现相同的可见性与条件更新语义 ----

type fakeMoodRepo struct {
	moods map[uint]*model.Mood
	next  uint
}

func newFakeMoodRepo() *fakeMoodRepo { return &fakeMoodRepo{moods: map[uint]*model.Mood{}} }

func (f *fakeMoodRepo) Create(v *model.Mood) error {
	f.next++
	v.ID = f.next
	cp := *v
	f.moods[v.ID] = &cp
	return nil
}
func (f *fakeMoodRepo) List(uid uint, date *time.Time) ([]model.Mood, error) {
	out := []model.Mood{}
	for _, v := range f.moods {
		if v.UserID != uid {
			continue
		}
		if date != nil && !sameDayTime(v.RecordDate, *date) {
			continue
		}
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RecordDate.After(out[j].RecordDate) })
	return out, nil
}
func (f *fakeMoodRepo) CountLow(uid uint, day time.Time, threshold int) (int64, error) {
	var n int64
	for _, v := range f.moods {
		if v.UserID == uid && sameDayTime(v.RecordDate, day) && v.MoodLevel <= threshold {
			n++
		}
	}
	return n, nil
}
func (f *fakeMoodRepo) ByID(id, uid uint) (*model.Mood, error) {
	v, ok := f.moods[id]
	if !ok || v.UserID != uid {
		return nil, repository.ErrNotFound
	}
	cp := *v
	return &cp, nil
}
func (f *fakeMoodRepo) Update(v *model.Mood) error {
	cp := *v
	f.moods[v.ID] = &cp
	return nil
}
func (f *fakeMoodRepo) Delete(v *model.Mood) error {
	delete(f.moods, v.ID)
	return nil
}

type fakeCheckInRepo struct {
	items map[uint]*model.MoodCheckIn
	next  uint
}

func newFakeCheckInRepo() *fakeCheckInRepo {
	return &fakeCheckInRepo{items: map[uint]*model.MoodCheckIn{}}
}

func (f *fakeCheckInRepo) Create(v *model.MoodCheckIn) error {
	for _, ex := range f.items {
		if ex.UserID == v.UserID && sameDayTime(ex.TriggerDate, v.TriggerDate) {
			return errors.New("duplicate key value violates unique constraint uniq_user_trigger_date")
		}
	}
	f.next++
	v.ID = f.next
	cp := *v
	f.items[v.ID] = &cp
	return nil
}
func (f *fakeCheckInRepo) ByTriggerDate(uid uint, day time.Time) (*model.MoodCheckIn, error) {
	for _, v := range f.items {
		if v.UserID == uid && sameDayTime(v.TriggerDate, day) {
			cp := *v
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeCheckInRepo) ByID(id, uid uint) (*model.MoodCheckIn, error) {
	v, ok := f.items[id]
	if !ok || v.UserID != uid {
		return nil, repository.ErrNotFound
	}
	cp := *v
	return &cp, nil
}
func (f *fakeCheckInRepo) List(uid uint, date, triggerDate *time.Time, status string) ([]model.MoodCheckIn, error) {
	out := []model.MoodCheckIn{}
	for _, v := range f.items {
		if v.UserID != uid {
			continue
		}
		if date != nil && !sameDayTime(v.CheckInDate, *date) {
			continue
		}
		if triggerDate != nil && !sameDayTime(v.TriggerDate, *triggerDate) {
			continue
		}
		if status != "" && v.Status != status {
			continue
		}
		out = append(out, *v)
	}
	return out, nil
}
func (f *fakeCheckInRepo) Update(v *model.MoodCheckIn) error {
	cp := *v
	f.items[v.ID] = &cp
	return nil
}

// SubmitResult 复刻仓储层的条件 UPDATE：仅 pending 行可被命中，天然抵御重复与并发提交。
func (f *fakeCheckInRepo) SubmitResult(uid, id uint, result, note string) (bool, error) {
	v, ok := f.items[id]
	if !ok || v.UserID != uid || v.Status != constants.CheckInStatusPending {
		return false, nil
	}
	now := time.Now()
	v.Status = result
	v.ResultNote = note
	v.RespondedAt = &now
	return true, nil
}

func sameDayTime(a, b time.Time) bool {
	return a.Truncate(24 * time.Hour).Equal(b.Truncate(24 * time.Hour))
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func day(n int) time.Time { return time.Date(2026, 9, n, 0, 0, 0, 0, time.UTC) }

func TestMoodCheckInClosedLoop(t *testing.T) {
	mr := newFakeMoodRepo()
	cr := newFakeCheckInRepo()
	cs := NewMoodCheckInService(cr, mr, testLogger())
	const uid = uint(1)
	// 测试通过假情绪仓储驱动闭环：CountLow 完全以当天真实情绪记录为准。
	seedMood := func(level int) {
		mr.Create(&model.Mood{UserID: uid, MoodLevel: level, MoodTags: "[]", RecordDate: day(10)})
	}
	clearMoods := func() { mr.moods = map[uint]*model.Mood{} }

	// 1. 指数不高于 3：生成次日回访，并记录首次触发来源 dashboard。
	seedMood(3)
	c1, e := cs.Reconcile(uid, day(10), constants.CheckInSourceDashboard, 3)
	if e != nil || c1.Status != constants.CheckInStatusPending {
		t.Fatalf("expected pending check-in, got %+v err=%v", c1, e)
	}
	if !sameDayTime(c1.CheckInDate, day(11)) {
		t.Fatalf("check-in should be scheduled next day, got %s", c1.CheckInDate)
	}
	if c1.Source != constants.CheckInSourceDashboard || c1.MoodLevel != 3 {
		t.Fatalf("first trigger source/level not kept: %+v", c1)
	}

	// 2. 同日再次低落：只合并一次，不新增，来源保持首次的 dashboard。
	seedMood(2)
	before := len(cr.items)
	c2, e := cs.Reconcile(uid, day(10), constants.CheckInSourceMoods, 2)
	if e != nil || c2.ID != c1.ID || len(cr.items) != before {
		t.Fatalf("same-day low mood must merge into one check-in")
	}
	if c2.Source != constants.CheckInSourceDashboard {
		t.Fatalf("first trigger source must not be overwritten on merge")
	}

	// 3. 指数回升：待回访撤销（当天不再有低落记录）。
	clearMoods()
	seedMood(8)
	c3, e := cs.Reconcile(uid, day(10), constants.CheckInSourceMoods, 8)
	if e != nil || c3.Status != constants.CheckInStatusRevoked || c3.RevokedAt == nil {
		t.Fatalf("expected revoked after recovery, got %+v", c3)
	}

	// 4. 撤销后当天再次低落：重新激活，本轮来源更新为 moods。
	clearMoods()
	seedMood(1)
	c4, e := cs.Reconcile(uid, day(10), constants.CheckInSourceMoods, 1)
	if e != nil || c4.Status != constants.CheckInStatusPending || c4.Source != constants.CheckInSourceMoods {
		t.Fatalf("expected reactivated pending with new source, got %+v", c4)
	}

	// 5. 本人确认好转。
	responded, applied, e := cs.Respond(uid, c4.ID, dto.CheckInRespondRequest{Result: constants.CheckInStatusImproved})
	if e != nil || !applied || responded.Status != constants.CheckInStatusImproved || responded.RespondedAt == nil {
		t.Fatalf("respond failed: %+v applied=%v err=%v", responded, applied, e)
	}

	// 6. 已提交结果不受后续修改影响：回升不再撤销它。
	clearMoods()
	seedMood(9)
	c6, e := cs.Reconcile(uid, day(10), constants.CheckInSourceMoods, 9)
	if e != nil || c6.Status != constants.CheckInStatusImproved {
		t.Fatalf("submitted result must be frozen, got %+v", c6)
	}

	// 7. 完全重复提交幂等；不同结果的重复/并发提交冲突，库里仍是第一条。
	same, appliedSame, e := cs.Respond(uid, c4.ID, dto.CheckInRespondRequest{Result: constants.CheckInStatusImproved})
	if e != nil || !appliedSame || same.Status != constants.CheckInStatusImproved {
		t.Fatalf("identical repeat should be idempotent, got %+v err=%v", same, e)
	}
	conflicted, appliedConflict, e := cs.Respond(uid, c4.ID, dto.CheckInRespondRequest{Result: constants.CheckInStatusStillTroubled})
	if e == nil || appliedConflict || conflicted.Status != constants.CheckInStatusImproved {
		t.Fatalf("concurrent/different result must conflict and keep first, got %+v applied=%v", conflicted, appliedConflict)
	}
}

func TestReconcileHighMoodCreatesNothing(t *testing.T) {
	mr, cr := newFakeMoodRepo(), newFakeCheckInRepo()
	cs := NewMoodCheckInService(cr, mr, testLogger())

	c, e := cs.Reconcile(7, day(20), constants.CheckInSourceDashboard, 7)
	if e != nil {
		t.Fatalf("unexpected err: %v", e)
	}
	if c != nil || len(cr.items) != 0 {
		t.Fatalf("no check-in should be created when mood level is above threshold")
	}
}

// TestMoodServiceDrivesCheckInLifecycle 通过 MoodService 走完整情绪 CRUD，
// 验证创建低落 -> 回升编辑 -> 删除的每一步都会驱动回访闭环。
func TestMoodServiceDrivesCheckInLifecycle(t *testing.T) {
	mr := newFakeMoodRepo()
	cr := newFakeCheckInRepo()
	cs := NewMoodCheckInService(cr, mr, testLogger())
	ms := NewMoodService(mr, cs, testLogger())
	const uid = uint(42)

	low, e := ms.Create(uid, dto.MoodRequest{MoodLevel: 2, MoodTags: []string{constants.MoodAnxious}, RecordDate: "2026-09-10", Source: constants.CheckInSourceDashboard})
	if e != nil {
		t.Fatalf("create low mood: %v", e)
	}
	ci, e := cr.ByTriggerDate(uid, day(10))
	if e != nil || ci.Status != constants.CheckInStatusPending || ci.Source != constants.CheckInSourceDashboard {
		t.Fatalf("low mood create should schedule sourced check-in, got %+v err=%v", ci, e)
	}

	// 编辑为指数回升（>3），回访应被撤销。
	if _, e = ms.Update(uid, low.ID, dto.MoodRequest{MoodLevel: 8, MoodTags: []string{constants.MoodCalm}, RecordDate: "2026-09-10"}); e != nil {
		t.Fatalf("update mood: %v", e)
	}
	ci, _ = cr.ByTriggerDate(uid, day(10))
	if ci.Status != constants.CheckInStatusRevoked {
		t.Fatalf("expected revoked after edit recovery, got %s", ci.Status)
	}

	// 删除当天最后一条记录（无低落）后状态保持撤销；再补一条低落并删除，应再次撤销。
	low2, e := ms.Create(uid, dto.MoodRequest{MoodLevel: 1, MoodTags: []string{constants.MoodTired}, RecordDate: "2026-09-10", Source: constants.CheckInSourceMoods})
	if e != nil {
		t.Fatalf("create second low mood: %v", e)
	}
	ci, _ = cr.ByTriggerDate(uid, day(10))
	if ci.Status != constants.CheckInStatusPending {
		t.Fatalf("expected reactivated pending, got %s", ci.Status)
	}
	if e = ms.Delete(uid, low2.ID); e != nil {
		t.Fatalf("delete mood: %v", e)
	}
	ci, _ = cr.ByTriggerDate(uid, day(10))
	if ci.Status != constants.CheckInStatusRevoked {
		t.Fatalf("deleting the low record after recovery data must revoke, got %s", ci.Status)
	}
}
