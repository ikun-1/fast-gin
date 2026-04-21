package routers

import (
	"fast-gin/handlers"
	"fast-gin/handlers/signal"
	"fast-gin/middleware"

	"github.com/gin-gonic/gin"
)

func SignalRouter(g *gin.RouterGroup) {
	Signal := handlers.Handlers.Signal
	g.GET("signal/ws", middleware.ShouldBindQuery[signal.WSQuery], Signal.WSView)
	g.POST("signal/connect", middleware.AuthMiddleware, middleware.ShouldBindJSON[signal.SignalConnectRequest], Signal.ConnectView)
	g.POST("signal/disconnect", middleware.AuthMiddleware, middleware.ShouldBindJSON[signal.SignalConnectRequest], Signal.DisConnectView)
	g.POST("signal/offer", middleware.AuthMiddleware, middleware.ShouldBindJSON[signal.SignalMessage], Signal.OfferView)
	g.POST("signal/answer", middleware.AuthMiddleware, middleware.ShouldBindJSON[signal.SignalMessage], Signal.AnswerView)
	g.POST("signal/ice-candidate", middleware.AuthMiddleware, middleware.ShouldBindJSON[signal.SignalMessage], Signal.IceCandidateView)
}
