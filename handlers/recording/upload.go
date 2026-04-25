package recording

import (
	"errors"
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/find"
	"fast-gin/utils/res"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var whiteExt = []string{".webm"}

func (Recording) UploadView(c *gin.Context) {
	claims := middleware.GetAuth(c)
	if claims == nil || claims.UserID == 0 {
		res.FailWithMsg(c, "用户认证失败")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		res.FailWithMsg(c, "请选择录制文件")
		return
	}

	// 大小限制
	maxBytes := int64(global.Config.Recording.MaxSize) * 1024 * 1024
	if fileHeader.Size > maxBytes {
		res.FailWithMsg(c, fmt.Sprintf("上传文件过大，最大支持%dMB", global.Config.Recording.MaxSize))
		return
	}

	// 扩展名校验
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !find.InList(whiteExt, ext) {
		res.FailWithMsg(c, "仅支持WebM格式的录制文件")
		return
	}

	// 读取表单字段
	meetingIDStr := c.PostForm("meetingId")
	meetingID, err := strconv.ParseUint(meetingIDStr, 10, 64)
	if err != nil || meetingID == 0 {
		res.FailWithMsg(c, "请提供有效的会议ID")
		return
	}

	durationStr := c.PostForm("duration")
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil || duration <= 0 {
		res.FailWithMsg(c, "请提供有效的录制时长")
		return
	}

	// 验证会议存在，且上传者是主持人
	var meeting models.Meeting
	if err := global.DB.Where("room_no = ?", meetingID).First(&meeting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailWithMsg(c, "会议不存在")
		} else {
			res.FailWithMsg(c, "查询会议失败")
		}
		return
	}
	if meeting.HostID != claims.UserID {
		res.FailWithMsg(c, "仅主持人可上传录制文件")
		return
	}

	// 确保存储目录存在
	dir := path.Join("uploads", global.Config.Recording.Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		res.FailWithMsg(c, "创建存储目录失败")
		return
	}

	// 处理文件名重复
	fp := path.Join(dir, fileHeader.Filename)
	counter := 0
	for {
		_, statErr := os.Stat(fp)
		if os.IsNotExist(statErr) {
			break
		}
		counter++
		nameWithoutExt := strings.TrimSuffix(fileHeader.Filename, ext)
		fp = path.Join(dir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
	}

	if err := c.SaveUploadedFile(fileHeader, fp); err != nil {
		res.FailWithMsg(c, "文件保存失败")
		return
	}

	recording := &models.Recording{
		MeetingID: uint(meetingID),
		UserID:    claims.UserID,
		FileName:  fileHeader.Filename,
		FilePath:  "/" + fp,
		FileSize:  fileHeader.Size,
		Duration:  duration,
	}
	if err := global.DB.Create(recording).Error; err != nil {
		os.Remove(fp)
		res.FailWithMsg(c, "录制记录保存失败")
		return
	}

	res.Ok(c, gin.H{
		"id":       recording.ID,
		"filePath": recording.FilePath,
		"fileSize": recording.FileSize,
	}, "上传成功")
}
