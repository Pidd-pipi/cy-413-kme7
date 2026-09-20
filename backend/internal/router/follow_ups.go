package router

import (
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterFollowUps(g *gin.RouterGroup, h *handler.FollowUpHandler, auth gin.HandlerFunc) {
	p := g.Group("/follow-ups", auth)
	p.GET("", h.List)
	p.POST("/:id/respond", h.Respond)
}
