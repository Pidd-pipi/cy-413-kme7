package router

import (
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterCheckIns(g *gin.RouterGroup, h *handler.MoodCheckInHandler, auth gin.HandlerFunc) {
	p := g.Group("/checkins", auth)
	p.GET("", h.List)
	p.PUT("/:id/respond", h.Respond)
}
