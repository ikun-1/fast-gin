package routers

import (
	"fast-gin/dal/query"
	"fast-gin/service/ws_serv"
	"fast-gin/utils/jwts"
	"fast-gin/utils/res"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins during development
	},
}

func SignalingRouter(g *gin.RouterGroup) {
	g.GET("ws/meeting", func(c *gin.Context) {
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

		// Get user display name
		displayName := ""
		user, err := query.User.WithContext(c).
			Where(query.User.ID.Eq(claims.UserID)).
			Select(query.User.Nickname, query.User.Username).
			First()
		if err == nil {
			if user.Nickname != "" {
				displayName = user.Nickname
			} else {
				displayName = user.Username
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			zap.S().Errorf("WebSocket upgrade failed: %s", err)
			return
		}

		client := ws_serv.NewClient(hub, conn, claims.UserID, displayName)
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	})
}
