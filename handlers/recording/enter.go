package recording

import (
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/common"
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

	// Recording list defaults to sort by started_at desc
	page.SortBy = "started_at"
	page.SortDir = "desc"

	recordings, count, err := common.QueryList(models.Recording{}, common.QueryOption{
		PageInfo: page,
		Where:    global.DB.Where(query.Recording.HostID.Eq(claims.UserID)),
	})
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	list := make([]RecordingListItem, 0, len(recordings))
	for _, rec := range recordings {
		var title string
		meeting, err := query.Meeting.WithContext(c).Where(query.Meeting.ID.Eq(rec.MeetingID)).First()
		if err == nil {
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

	rec, err := query.Recording.WithContext(c).Where(query.Recording.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailNotFound(c)
		return
	}

	var title string
	meeting, err := query.Meeting.WithContext(c).Where(query.Meeting.ID.Eq(rec.MeetingID)).First()
	if err == nil {
		title = meeting.Title
	}

	files, err := query.RecordingFile.WithContext(c).Where(query.RecordingFile.RecordingID.Eq(rec.ID)).Find()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

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

	file, err := query.RecordingFile.WithContext(c).Where(query.RecordingFile.ID.Eq(uri.FileID)).First()
	if err != nil {
		res.FailNotFound(c)
		return
	}

	c.FileAttachment(file.FilePath, filepath.Base(file.FilePath))
}

func (Recording) FilePlayView(c *gin.Context) {
	uri := middleware.GetUri[models.BindFileId](c)

	file, err := query.RecordingFile.WithContext(c).Where(query.RecordingFile.ID.Eq(uri.FileID)).First()
	if err != nil {
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

	rec, err := query.Recording.WithContext(c).Where(query.Recording.ID.Eq(uri.ID)).First()
	if err != nil {
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
	query.RecordingFile.WithContext(c).Where(query.RecordingFile.RecordingID.Eq(rec.ID)).Delete()
	query.Recording.WithContext(c).Delete(rec)

	res.OkSuccess(c)
}
