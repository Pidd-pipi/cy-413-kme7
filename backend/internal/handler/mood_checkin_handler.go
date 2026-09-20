package handler

import (
	"log/slog"
	"strconv"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/middleware"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type MoodCheckInHandler struct {
	s      *service.MoodCheckInService
	logger *slog.Logger
}

func NewMoodCheckInHandler(s *service.MoodCheckInService, l *slog.Logger) *MoodCheckInHandler {
	return &MoodCheckInHandler{s: s, logger: l}
}

func (h *MoodCheckInHandler) List(c *gin.Context) {
	v, e := h.s.List(middleware.UserID(c), c.Query("date"), c.Query("trigger_date"), c.Query("status"))
	if e != nil {
		c.Error(e)
		return
	}
	h.logger.Info(constants.LogCheckInListed)
	ok(c, v)
}

// Respond 本人确认好转或仍困扰。重复提交幂等返回；与已有结果冲突的并发提交返回 409。
func (h *MoodCheckInHandler) Respond(c *gin.Context) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil {
		c.Error(e)
		return
	}
	var r dto.CheckInRespondRequest
	if !bind(c, &r) {
		return
	}
	v, _, e := h.s.Respond(middleware.UserID(c), uint(id), r)
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}
