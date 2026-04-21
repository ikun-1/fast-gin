package room

import "github.com/gin-gonic/gin"

type CreateRoomView func(c *gin.Context)
type JoinRoomView func(c *gin.Context)
type LeaveRoomView func(c *gin.Context)
type RoomDetailView func(c *gin.Context)
