package handler

import (
	"errors"
	"strconv"

	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/middleware"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"github.com/gin-gonic/gin"
	"log/slog"
)

type FollowUpHandler struct {
	s      *service.FollowUpService
	logger *slog.Logger
}

func NewFollowUpHandler(s *service.FollowUpService, l *slog.Logger) *FollowUpHandler {
	return &FollowUpHandler{s: s, logger: l}
}

func (h *FollowUpHandler) List(c *gin.Context) {
	v, e := h.s.List(middleware.UserID(c), c.Query("status"), c.Query("trigger_date"))
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

// Respond 本人确认回访结果。重复/并发提交时服务端只保留第一条，
// 这里按幂等处理：始终把唯一那条记录返回给本人。
func (h *FollowUpHandler) Respond(c *gin.Context) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil {
		c.Error(e)
		return
	}
	var r dto.FollowUpRespondRequest
	if !bind(c, &r) {
		return
	}
	v, se := h.s.Respond(middleware.UserID(c), uint(id), r.Result)
	if se != nil {
		var conflict *service.FollowUpConflictError
		if errors.As(se, &conflict) {
			ok(c, conflict.FollowUp)
			return
		}
		c.Error(se)
		return
	}
	ok(c, v)
}
