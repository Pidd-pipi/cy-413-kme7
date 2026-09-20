package integration

import (
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 真实 PostgreSQL 集成测试：MINDGARDEN_TEST_DSN 未设置时跳过（CI/本地无库环境）。
func openTestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("MINDGARDEN_TEST_DSN")
	if dsn == "" {
		t.Skip("MINDGARDEN_TEST_DSN not set; skipping PostgreSQL integration test")
	}
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if e != nil {
		t.Fatalf("open db: %v", e)
	}
	if e = db.AutoMigrate(&model.User{}, &model.Mood{}, &model.FollowUp{}); e != nil {
		t.Fatalf("migrate: %v", e)
	}
	t.Cleanup(func() {
		db.Exec("TRUNCATE follow_ups, moods, users RESTART IDENTITY CASCADE")
	})
	return db
}

type harness struct {
	uid     uint
	moodSvc *service.MoodService
	fuSvc   *service.FollowUpService
	fuRepo  repository.FollowUpRepository
}

func newHarness(t *testing.T, db *gorm.DB, email string) *harness {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	u := &model.User{Email: email, PasswordHash: "x", Nickname: "Tester", Role: "user"}
	if e := db.Create(u).Error; e != nil {
		t.Fatalf("create user: %v", e)
	}
	mr := repository.NewMoodRepository(db)
	fr := repository.NewFollowUpRepository(db)
	tx := repository.NewTxManager(db)
	fs := service.NewFollowUpService(fr, mr, logger)
	ms := service.NewMoodService(mr, tx, fs, logger)
	return &harness{uid: u.ID, moodSvc: ms, fuSvc: fs, fuRepo: fr}
}

func moodReq(level int, date, source string) dto.MoodRequest {
	return dto.MoodRequest{MoodLevel: level, MoodTags: []string{constants.MoodCalm}, Note: "", RecordDate: date, Source: source}
}

// 规则 1：低落（<=3）生成次日回访并记录首次来源；健康（>3）不生成。
func TestLowMoodSchedulesNextDayFollowUpWithSource(t *testing.T) {
	db := openTestDB(t)
	h := newHarness(t, db, "sched@example.com")
	day := "2026-09-10"

	if _, e := h.moodSvc.Create(h.uid, moodReq(8, day, constants.FollowUpSourceMood)); e != nil {
		t.Fatalf("healthy mood: %v", e)
	}
	if list, _ := h.fuSvc.List(h.uid, "", ""); len(list) != 0 {
		t.Fatalf("healthy mood must not schedule follow-up, got %d", len(list))
	}

	if _, e := h.moodSvc.Create(h.uid, moodReq(3, day, constants.FollowUpSourceMood)); e != nil {
		t.Fatalf("low mood: %v", e)
	}
	d, _ := time.Parse("2006-01-02", day)
	f, e := h.fuRepo.ByTriggerDate(h.uid, d)
	if e != nil {
		t.Fatalf("follow-up should exist: %v", e)
	}
	if f.Status != constants.FollowUpStatusPending {
		t.Fatalf("status=%s want pending", f.Status)
	}
	if f.Source != constants.FollowUpSourceMood {
		t.Fatalf("source=%s want mood", f.Source)
	}
	if f.FirstMoodID == 0 {
		t.Fatal("first_mood_id must be recorded")
	}
	if !f.ScheduledDate.Equal(d.AddDate(0, 0, 1)) {
		t.Fatalf("scheduled_date=%s want next day", f.ScheduledDate)
	}
}

// 规则 2：同日再次记录低落只合并一条；指数回升则撤销；撤销后再次低落重新挂起且保留首次来源。
func TestSameDayMergesRecoversRevokesAndReopens(t *testing.T) {
	db := openTestDB(t)
	h := newHarness(t, db, "merge@example.com")
	day := "2026-09-11"
	d, _ := time.Parse("2006-01-02", day)

	m1, _ := h.moodSvc.Create(h.uid, moodReq(2, day, constants.FollowUpSourceMood))
	_, _ = h.moodSvc.Create(h.uid, moodReq(3, day, constants.FollowUpSourceMoodList))
	// 第三次低落依旧只保留一条
	h.moodSvc.Create(h.uid, moodReq(1, day, constants.FollowUpSourceMoodList))
	if list, _ := h.fuSvc.List(h.uid, "", day); len(list) != 1 {
		t.Fatalf("same-day low moods must merge into one, got %d", len(list))
	}
	f, _ := h.fuRepo.ByTriggerDate(h.uid, d)
	if f.Source != constants.FollowUpSourceMood || f.FirstMoodID != m1.ID {
		t.Fatalf("first source/mood must be preserved: source=%s first=%d", f.Source, f.FirstMoodID)
	}

	// 同日的健康记录不应改变（仍存在低落记录），保持 pending
	h.moodSvc.Create(h.uid, moodReq(9, day, constants.FollowUpSourceMoodList))
	f, _ = h.fuRepo.ByTriggerDate(h.uid, d)
	if f.Status != constants.FollowUpStatusPending {
		t.Fatalf("pending must remain while any low mood exists, got %s", f.Status)
	}

	// 把当天所有低落记录逐条改成健康，最后一条回升后撤销
	var lows []model.Mood
	db.Where("user_id=? AND mood_level<=3", h.uid).Find(&lows)
	for _, low := range lows {
		req := moodReq(7, day, constants.FollowUpSourceMoodList)
		if _, e := h.moodSvc.Update(h.uid, low.ID, req); e != nil {
			t.Fatalf("recover mood %d: %v", low.ID, e)
		}
	}
	f, _ = h.fuRepo.ByTriggerDate(h.uid, d)
	if f.Status != constants.FollowUpStatusRevoked {
		t.Fatalf("status=%s want revoked after recovery", f.Status)
	}
	if f.Source != constants.FollowUpSourceMood || f.FirstMoodID != m1.ID {
		t.Fatalf("first source/mood must survive revoke")
	}

	// 撤销后再次低落：重新挂起，仍保留首次来源
	h.moodSvc.Create(h.uid, moodReq(2, day, constants.FollowUpSourceMoodList))
	f, _ = h.fuRepo.ByTriggerDate(h.uid, d)
	if f.Status != constants.FollowUpStatusPending || f.Source != constants.FollowUpSourceMood {
		t.Fatalf("reopen must keep first source, got status=%s source=%s", f.Status, f.Source)
	}
}

// 规则 3：已提交的回访结果不受后续情绪修改影响。
func TestRespondedResultIsImmutable(t *testing.T) {
	db := openTestDB(t)
	h := newHarness(t, db, "immutable@example.com")
	day := "2026-09-12"
	d, _ := time.Parse("2006-01-02", day)

	h.moodSvc.Create(h.uid, moodReq(2, day, constants.FollowUpSourceMood))
	f, _ := h.fuRepo.ByTriggerDate(h.uid, d)
	// 到期日由测试时间（2026-09-20）决定，此处直接把计划日改为过去以模拟次日
	now := time.Now()
	db.Model(&model.FollowUp{}).Where("id=?", f.ID).Update("scheduled_date", now.AddDate(0, 0, -1))

	if _, e := h.fuSvc.Respond(h.uid, f.ID, constants.FollowUpResultBetter); e != nil {
		t.Fatalf("respond: %v", e)
	}
	// 后续把所有情绪改为健康，已提交结果保持不变
	var lows []model.Mood
	db.Where("user_id=? AND mood_level<=3", h.uid).Find(&lows)
	for _, low := range lows {
		h.moodSvc.Update(h.uid, low.ID, moodReq(10, day, constants.FollowUpSourceMoodList))
	}
	got, _ := h.fuRepo.ByID(h.uid, f.ID)
	if got.Status != constants.FollowUpStatusResponded || got.Result != constants.FollowUpResultBetter {
		t.Fatalf("responded result mutated: status=%s result=%s", got.Status, got.Result)
	}
}

// 规则 4：重复及并发提交只保留一条。
func TestDuplicateAndConcurrentRespondKeepOne(t *testing.T) {
	db := openTestDB(t)
	h := newHarness(t, db, "dup@example.com")
	day := "2026-09-13"
	d, _ := time.Parse("2006-01-02", day)
	h.moodSvc.Create(h.uid, moodReq(2, day, constants.FollowUpSourceMood))
	f, _ := h.fuRepo.ByTriggerDate(h.uid, d)
	db.Model(&model.FollowUp{}).Where("id=?", f.ID).Update("scheduled_date", time.Now().AddDate(0, 0, -1))

	// 先提交 better
	if _, e := h.fuSvc.Respond(h.uid, f.ID, constants.FollowUpResultBetter); e != nil {
		t.Fatalf("first respond: %v", e)
	}
	// 重复提交 struggling：冲突错误中携带的仍是第一条 better
	_, e := h.fuSvc.Respond(h.uid, f.ID, constants.FollowUpResultStruggling)
	if e == nil {
		t.Fatal("duplicate respond must surface a conflict")
	}
	got, _ := h.fuRepo.ByID(h.uid, f.ID)
	if got.Result != constants.FollowUpResultBetter {
		t.Fatalf("duplicate submission overwrote result: %s", got.Result)
	}

	// 并发提交：N 个 goroutine 同时回应，只有一条 UPDATE 生效
	h2 := newHarness(t, db, "concurrent@example.com")
	h2.moodSvc.Create(h2.uid, moodReq(2, day, constants.FollowUpSourceMood))
	f2, _ := h2.fuRepo.ByTriggerDate(h2.uid, d)
	db.Model(&model.FollowUp{}).Where("id=?", f2.ID).Update("scheduled_date", time.Now().AddDate(0, 0, -1))
	var wg sync.WaitGroup
	var wins int32
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := constants.FollowUpResultBetter
			_, err := h2.fuSvc.Respond(h2.uid, f2.ID, res)
			if err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("exactly one concurrent submission should win, got %d", wins)
	}
	final, _ := h2.fuRepo.ByID(h2.uid, f2.ID)
	if final.Status != constants.FollowUpStatusResponded {
		t.Fatalf("concurrent result not persisted once: %s", final.Status)
	}
}

// 规则 5：同日并发记录低落只生成一条回访。
func TestConcurrentSameDayCreatesProduceSingleFollowUp(t *testing.T) {
	db := openTestDB(t)
	h := newHarness(t, db, "concday@example.com")
	day := "2026-09-14"
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(level int) {
			defer wg.Done()
			_, _ = h.moodSvc.Create(h.uid, moodReq(level, day, constants.FollowUpSourceMoodList))
		}(2)
	}
	wg.Wait()
	d, _ := time.Parse("2006-01-02", day)
	list, _ := h.fuSvc.List(h.uid, "", day)
	if len(list) != 1 {
		t.Fatalf("concurrent same-day creates must yield exactly one follow-up, got %d", len(list))
	}
	_ = d
}
