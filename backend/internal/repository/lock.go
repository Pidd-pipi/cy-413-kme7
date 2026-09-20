package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LockUserDay 获取"用户 + 日期"维度的事务级咨询锁。
// 即便当天还没有任何情绪行（行锁无对象可锁），也能保证同一用户同一天的
// 情绪写入严格串行，使"同日只合并一条回访"在并发下成立；锁在事务结束时自动释放。
func LockUserDay(tx *gorm.DB, uid uint, day time.Time) error {
	key := fmt.Sprintf("mood-day:%d:%s", uid, day.Truncate(24*time.Hour).Format("2006-01-02"))
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}
