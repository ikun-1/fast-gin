package signal

import (
	"net/http"

	"fast-gin/middleware"
	"fast-gin/service/signal_serv"
	"fast-gin/utils/jwts"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSQuery struct {
	RoomNo string `form:"room_no" binding:"required"`
	UserID uint   `form:"user_id" binding:"required"`
	Token  string `form:"token" binding:"required"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (Signal) WSView(c *gin.Context) {
	req := middleware.GetQuery[WSQuery](c)
	if _, err := jwts.CheckToken("Bearer " + req.Token); err != nil {
		return
	}

	roomNo := req.RoomNo
	userID := req.UserID

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	conn := signal_serv.NewConnection(ws, signal_serv.HubInstance, roomNo, userID)
	signal_serv.HubInstance.Register(&signal_serv.Client{Conn: conn, RoomNo: roomNo, UserID: userID, Send: conn.Send})
	go conn.Start()
}
