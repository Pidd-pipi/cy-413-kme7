package service

import (
	"fmt"

	"github.com/blueship581/mindgarden/backend/internal/model"
)

// FollowUpConflictError 表示回访已被本人处理过；随错误携带既有记录，
// 保证重复/并发提交只保留第一条时，调用方仍能返回唯一那条结果。
type FollowUpConflictError struct {
	FollowUp *model.FollowUp
}

func (e *FollowUpConflictError) Error() string {
	return fmt.Sprintf("FollowUp[id=%d] already responded with result=%s", e.FollowUp.ID, e.FollowUp.Result)
}
