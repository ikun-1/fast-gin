package routers

import (
	"fast-gin/global"
	"fast-gin/handlers"
	notification_handler "fast-gin/handlers/notification"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/ws_serv"
	"fast-gin/utils/jwts"
	"fast-gin/utils/res"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var notifUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NotificationRouter(g *gin.RouterGroup) {
	Notification := handlers.Handlers.Notification

	// Inject WebSocket pusher for real-time notification delivery
	notification_handler.SetNotifPusher(notifHub)

	g.POST("notifications",
		middleware.AuthMiddleware,
		Notification.CreateView)

	g.GET("notifications",
		middleware.AuthMiddleware,
		Notification.ListView)

	g.GET("notifications/unread-count",
		middleware.AuthMiddleware,
		Notification.UnreadCountView)

	g.PUT("notifications/:id/read",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		Notification.ReadView)

	g.PUT("notifications/read-all",
		middleware.AuthMiddleware,
		Notification.ReadAllView)

	g.DELETE("notifications/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		Notification.DeleteView)

	// WebSocket endpoint for real-time notification push
	g.GET("ws/notifications", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			res.FailAuth(c)
			return
		}
		claims, err := jwts.CheckToken(token)
		if err != nil {
			res.FailAuth(c)
			return
		}

		conn, err := notifUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			zap.S().Errorf("notif WS upgrade failed: %s", err)
			return
		}

		client := &ws_serv.NotifClient{
			UserID: claims.UserID,
			Conn:   conn,
		}
		notifHub.Register(client)

		// Send initial unread count
		var count int64
		global.DB.Model(&models.Notification{}).
			Where("to_user_id = ? AND status = ?", claims.UserID, "unread").
			Count(&count)
		client.SendJSON(map[string]any{
			"type":  "unread-count",
			"count": count,
		})

		// Keep connection alive, read until close
		go func() {
			defer func() {
				notifHub.Unregister(client)
			}()
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()
	})
}
