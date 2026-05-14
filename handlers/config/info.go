package config

import (
	"fast-gin/global"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

type Config struct{}

func (Config) GetIceServers(c *gin.Context) {
	servers := global.Config.WebRTC.ICEServers
	vo := make([]IceServerVO, 0, len(servers))
	for _, s := range servers {
		vo = append(vo, IceServerVO{
			URLs:       s.URLs,
			Username:   s.Username,
			Credential: s.Credential,
		})
	}
	res.OkWithData(c, IceServersResponse{IceServers: vo})
}
