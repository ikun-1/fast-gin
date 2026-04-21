package signal

import (
	"fast-gin/middleware"
	"fast-gin/service/signal_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

type SignalMessage struct {
	Type   string         `json:"type" binding:"required"`
	RoomNo string         `json:"room_no" binding:"required"`
	From   uint           `json:"from"`
	To     uint           `json:"to"`
	Data   map[string]any `json:"data"`
}

type SignalConnectRequest struct {
	RoomNo string `json:"room_no" binding:"required"`
	UserID uint   `json:"user_id" binding:"required"`
}

func (Signal) ConnectView(c *gin.Context) {
	req := middleware.GetJSON[SignalConnectRequest](c)
	info, err := signal_serv.Service.Connect(signal_serv.ConnectRequest{RoomNo: req.RoomNo, UserID: req.UserID})
	if err != nil {
		res.FailWithMsg(c, err.Error())
		return
	}
	res.OkWithData(c, info)
}

func (Signal) DisConnectView(c *gin.Context) {
	req := middleware.GetJSON[SignalConnectRequest](c)
	if err := signal_serv.Service.Disconnect(signal_serv.ConnectRequest{RoomNo: req.RoomNo, UserID: req.UserID}); err != nil {
		res.FailWithMsg(c, err.Error())
		return
	}
	res.OkSuccess(c)
}

func (Signal) OfferView(c *gin.Context) {
	msg := middleware.GetJSON[SignalMessage](c)
	_ = signal_serv.Service.HandleMessage(signal_serv.SignalMessage(msg))
	res.OkSuccess(c)
}

func (Signal) AnswerView(c *gin.Context) {
	msg := middleware.GetJSON[SignalMessage](c)
	_ = signal_serv.Service.HandleMessage(signal_serv.SignalMessage(msg))
	res.OkSuccess(c)
}

func (Signal) IceCandidateView(c *gin.Context) {
	msg := middleware.GetJSON[SignalMessage](c)
	_ = signal_serv.Service.HandleMessage(signal_serv.SignalMessage(msg))
	res.OkSuccess(c)
}
