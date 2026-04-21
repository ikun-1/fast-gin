package room

import (
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/room_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

type CreateRoomRequest struct {
	Title      string `json:"title" binding:"required"`
	MaxMembers int    `json:"maxMembers"`
}

type JoinRoomRequest struct {
	RoomNo string `json:"roomNo" binding:"required"`
}

func (Room) CreateRoomView(c *gin.Context) {
	req := middleware.GetJSON[CreateRoomRequest](c)
	claims := middleware.GetAuth(c)
	room, err := room_serv.Service.CreateRoom(c, req.Title, req.MaxMembers, claims.UserID)
	if err != nil {
		res.FailWithMsg(c, err.Error())
		return
	}
	res.OkWithData(c, room)
}

func (Room) JoinRoomView(c *gin.Context) {
	req := middleware.GetJSON[JoinRoomRequest](c)
	claims := middleware.GetAuth(c)
	room, err := room_serv.Service.JoinRoom(c, req.RoomNo, claims.UserID)
	if err != nil {
		res.FailWithMsg(c, err.Error())
		return
	}
	res.OkWithData(c, room)
}

func (Room) RoomDetailView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	claims := middleware.GetAuth(c)
	detail, err := room_serv.Service.GetMeetingDetail(c, uri.ID, claims.UserID)
	if err != nil {
		res.FailWithMsg(c, err.Error())
		return
	}
	res.OkWithData(c, detail)
}

func (Room) RoomMembersView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	claims := middleware.GetAuth(c)
	detail, err := room_serv.Service.GetMeetingDetail(c, uri.ID, claims.UserID)
	if err != nil {
		res.FailWithMsg(c, err.Error())
		return
	}
	res.OkWithData(c, gin.H{"members": detail.Members})
}
