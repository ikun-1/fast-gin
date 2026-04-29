package recording

import (
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/res"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type Recording struct{}

type StartRecordingResponse struct {
	RecordingID uint   `json:"recordingId"`
	StartedAt   string `json:"startedAt"`
}

type RecordingListItem struct {
	ID         uint   `json:"id"`
	RoomNo     uint   `json:"roomNo"`
	Title      string `json:"title"`
	StartedAt  string `json:"startedAt"`
	EndedAt    string `json:"endedAt,omitempty"`
	DurationMs int64  `json:"durationMs"`
	Status     string `json:"status"`
	FileCount  int    `json:"fileCount"`
}

type RecordingFileVO struct {
	ID          uint   `json:"id"`
	ClientID    string `json:"clientId"`
	DisplayName string `json:"displayName"`
	Kind        string `json:"kind"`
	Codec       string `json:"codec"`
	FileSize    int64  `json:"fileSize"`
	DownloadURL string `json:"downloadUrl"`
	PlayableURL string `json:"playableUrl,omitempty"`
}

type RecordingDetailVO struct {
	RecordingListItem
	Files []RecordingFileVO `json:"files"`
}

func (Recording) ListView(c *gin.Context) {
	claims := middleware.GetAuth(c)
	page := middleware.GetQuery[models.PageInfo](c)

	var recordings []models.Recording
	query := global.DB.WithContext(c).Where("host_id = ?", claims.UserID)
	var count int64
	if err := query.Model(&models.Recording{}).Count(&count).Error; err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	if err := query.Order("started_at desc").Offset((page.Page - 1) * page.Limit).Limit(page.Limit).Find(&recordings).Error; err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	list := make([]RecordingListItem, 0, len(recordings))
	for _, rec := range recordings {
		var title string
		var meeting models.Meeting
		if err := global.DB.Where("id = ?", rec.MeetingID).First(&meeting).Error; err == nil {
			title = meeting.Title
		}
		item := RecordingListItem{
			ID:         rec.ID,
			RoomNo:     rec.RoomNo,
			Title:      title,
			StartedAt:  rec.StartedAt.Format("2006-01-02 15:04:05"),
			DurationMs: rec.DurationMs,
			Status:     rec.Status,
			FileCount:  rec.FileCount,
		}
		if rec.EndedAt != nil {
			item.EndedAt = rec.EndedAt.Format("2006-01-02 15:04:05")
		}
		list = append(list, item)
	}

	res.OkWithList(c, list, count)
}

func (Recording) DetailView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	var rec models.Recording
	if err := global.DB.WithContext(c).First(&rec, uri.ID).Error; err != nil {
		res.FailNotFound(c)
		return
	}

	var title string
	var meeting models.Meeting
	if err := global.DB.Where("id = ?", rec.MeetingID).First(&meeting).Error; err == nil {
		title = meeting.Title
	}

	var files []models.RecordingFile
	global.DB.WithContext(c).Where("recording_id = ?", rec.ID).Find(&files)

	fileVOs := make([]RecordingFileVO, 0, len(files))
	for _, f := range files {
		vo := RecordingFileVO{
			ID:          f.ID,
			ClientID:    f.ClientID,
			DisplayName: f.DisplayName,
			Kind:        f.Kind,
			Codec:       f.Codec,
			FileSize:    f.FileSize,
			DownloadURL: filepath.Base(f.FilePath),
		}
		if f.Kind == "webm" && rec.Status == "completed" {
			vo.PlayableURL = fmt.Sprintf("/recordings/%d/files/%d/play", rec.ID, f.ID)
		}
		fileVOs = append(fileVOs, vo)
	}

	detail := RecordingDetailVO{
		RecordingListItem: RecordingListItem{
			ID:         rec.ID,
			RoomNo:     rec.RoomNo,
			Title:      title,
			StartedAt:  rec.StartedAt.Format("2006-01-02 15:04:05"),
			DurationMs: rec.DurationMs,
			Status:     rec.Status,
			FileCount:  rec.FileCount,
		},
		Files: fileVOs,
	}
	if rec.EndedAt != nil {
		detail.EndedAt = rec.EndedAt.Format("2006-01-02 15:04:05")
	}

	res.OkWithData(c, detail)
}

func (Recording) FileDownloadView(c *gin.Context) {
	uri := middleware.GetUri[models.BindFileId](c)

	var file models.RecordingFile
	if err := global.DB.WithContext(c).First(&file, uri.FileID).Error; err != nil {
		res.FailNotFound(c)
		return
	}

	c.FileAttachment(file.FilePath, filepath.Base(file.FilePath))
}

func (Recording) FilePlayView(c *gin.Context) {
	uri := middleware.GetUri[models.BindFileId](c)

	var file models.RecordingFile
	if err := global.DB.WithContext(c).First(&file, uri.FileID).Error; err != nil {
		res.FailNotFound(c)
		return
	}

	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		res.FailWithMsg(c, "录制文件尚未就绪")
		return
	}

	c.File(file.FilePath)
}

func (Recording) DeleteView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	claims := middleware.GetAuth(c)

	var rec models.Recording
	if err := global.DB.WithContext(c).First(&rec, uri.ID).Error; err != nil {
		res.FailNotFound(c)
		return
	}

	if rec.HostID != claims.UserID {
		res.FailPermission(c)
		return
	}

	// Delete files from disk
	os.RemoveAll(rec.StoragePath)

	// Delete file records
	global.DB.WithContext(c).Where("recording_id = ?", rec.ID).Delete(&models.RecordingFile{})
	global.DB.WithContext(c).Delete(&rec)

	res.OkSuccess(c)
}
