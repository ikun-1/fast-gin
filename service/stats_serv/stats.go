package stats_serv

import (
	"fast-gin/global"
	"fast-gin/models"
	"time"
)

type OverviewStats struct {
	TotalMeetings      int64 `json:"totalMeetings"`
	ActiveMeetings     int64 `json:"activeMeetings"`
	TotalParticipants  int64 `json:"totalParticipants"`
	TotalRecordings    int64 `json:"totalRecordings"`
	TotalDurationMs    int64 `json:"totalDurationMs"`
}

type ParticipantStat struct {
	UserID      uint   `json:"userId"`
	DisplayName string `json:"displayName"`
	IsHost      bool   `json:"isHost"`
	JoinedAt    string `json:"joinedAt"`
	LeftAt      string `json:"leftAt,omitempty"`
	DurationMs  int64  `json:"durationMs"`
}

type MeetingStats struct {
	MeetingID         uint              `json:"meetingId"`
	Title             string            `json:"title"`
	RoomNo            uint              `json:"roomNo"`
	Status            string            `json:"status"`
	StartedAt         string            `json:"startedAt,omitempty"`
	EndedAt           string            `json:"endedAt,omitempty"`
	TotalDurationMs   int64             `json:"totalDurationMs"`
	ParticipantCount  int               `json:"participantCount"`
	Participants      []ParticipantStat `json:"participants"`
}

type UserMeetingStat struct {
	MeetingID  uint   `json:"meetingId"`
	Title      string `json:"title"`
	RoomNo     uint   `json:"roomNo"`
	IsHost     bool   `json:"isHost"`
	JoinedAt   string `json:"joinedAt"`
	LeftAt     string `json:"leftAt,omitempty"`
	DurationMs int64  `json:"durationMs"`
}

type UserStats struct {
	UserID          uint              `json:"userId"`
	Username        string            `json:"username"`
	Nickname        string            `json:"nickname"`
	TotalMeetings   int64             `json:"totalMeetings"`
	TotalDurationMs int64             `json:"totalDurationMs"`
	MeetingsHosted  int64             `json:"meetingsHosted"`
	RecentMeetings  []UserMeetingStat `json:"recentMeetings"`
}

type TrendDay struct {
	Date         string `json:"date"`
	Meetings     int64  `json:"meetings"`
	Participants int64  `json:"participants"`
}

type TrendStats struct {
	Days []TrendDay `json:"days"`
}

// GetOverviewStats returns system-wide aggregate statistics.
func GetOverviewStats() (*OverviewStats, error) {
	stats := new(OverviewStats)

	if err := global.DB.Model(&models.Meeting{}).Count(&stats.TotalMeetings).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Model(&models.Meeting{}).Where("status = ?", "active").Count(&stats.ActiveMeetings).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Model(&models.MeetingParticipant{}).Select("COUNT(DISTINCT user_id)").Scan(&stats.TotalParticipants).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Model(&models.Recording{}).Count(&stats.TotalRecordings).Error; err != nil {
		return nil, err
	}

	// Calculate total duration from ended meetings using Go-level time arithmetic
	var meetings []models.Meeting
	global.DB.Where("status = ? AND started_at IS NOT NULL AND ended_at IS NOT NULL", "ended").
		Find(&meetings)
	for _, m := range meetings {
		stats.TotalDurationMs += m.EndedAt.Sub(*m.StartedAt).Milliseconds()
	}

	return stats, nil
}

// GetMeetingStats returns participation details for a single meeting.
func GetMeetingStats(meetingID uint) (*MeetingStats, error) {
	var meeting models.Meeting
	if err := global.DB.First(&meeting, meetingID).Error; err != nil {
		return nil, err
	}

	stats := &MeetingStats{
		MeetingID: meeting.ID,
		Title:     meeting.Title,
		RoomNo:    meeting.RoomNo,
		Status:    meeting.Status,
	}
	if meeting.StartedAt != nil {
		stats.StartedAt = meeting.StartedAt.Format("2006-01-02 15:04:05")
	}
	if meeting.EndedAt != nil {
		stats.EndedAt = meeting.EndedAt.Format("2006-01-02 15:04:05")
		stats.TotalDurationMs = meeting.EndedAt.Sub(*meeting.StartedAt).Milliseconds()
	}

	var participants []models.MeetingParticipant
	global.DB.Where("meeting_id = ?", meetingID).Order("joined_at ASC").Find(&participants)

	now := time.Now()
	for _, p := range participants {
		leftAt := p.LeftAt
		if leftAt == nil {
			leftAt = &now
		}
		durationMs := leftAt.Sub(p.JoinedAt).Milliseconds()
		if durationMs < 0 {
			durationMs = 0
		}

		ps := ParticipantStat{
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			IsHost:      p.IsHost,
			JoinedAt:    p.JoinedAt.Format("2006-01-02 15:04:05"),
			DurationMs:  durationMs,
		}
		if p.LeftAt != nil {
			ps.LeftAt = p.LeftAt.Format("2006-01-02 15:04:05")
		}
		stats.Participants = append(stats.Participants, ps)
	}
	stats.ParticipantCount = len(stats.Participants)

	return stats, nil
}

// GetUserStats returns meeting participation statistics for a specific user.
func GetUserStats(userID uint) (*UserStats, error) {
	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	stats := &UserStats{
		UserID:   user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
	}

	// Count total meetings attended
	global.DB.Model(&models.MeetingParticipant{}).
		Where("user_id = ?", userID).
		Select("COUNT(DISTINCT meeting_id)").Scan(&stats.TotalMeetings)

	// Count meetings hosted
	global.DB.Model(&models.Meeting{}).Where("host_id = ?", userID).Count(&stats.MeetingsHosted)

	// Fetch recent meeting participations
	var participants []models.MeetingParticipant
	global.DB.Where("user_id = ?", userID).Order("joined_at DESC").Limit(20).Find(&participants)

	now := time.Now()
	for _, p := range participants {
		leftAt := p.LeftAt
		if leftAt == nil {
			leftAt = &now
		}
		durationMs := leftAt.Sub(p.JoinedAt).Milliseconds()
		if durationMs < 0 {
			durationMs = 0
		}
		stats.TotalDurationMs += durationMs

		// Fetch meeting info
		var meeting models.Meeting
		title := ""
		if err := global.DB.First(&meeting, p.MeetingID).Error; err == nil {
			title = meeting.Title
		}

		ums := UserMeetingStat{
			MeetingID:  p.MeetingID,
			Title:      title,
			IsHost:     p.IsHost,
			JoinedAt:   p.JoinedAt.Format("2006-01-02 15:04:05"),
			DurationMs: durationMs,
		}
		if p.LeftAt != nil {
			ums.LeftAt = p.LeftAt.Format("2006-01-02 15:04:05")
		}
		stats.RecentMeetings = append(stats.RecentMeetings, ums)
	}

	return stats, nil
}

// GetTrendStats returns daily meeting and participant counts for the last N days.
func GetTrendStats(days int) (*TrendStats, error) {
	stats := &TrendStats{}

	startDate := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)

	for i := 0; i < days; i++ {
		dayStart := startDate.AddDate(0, 0, i)
		dayEnd := dayStart.Add(24 * time.Hour)

		day := TrendDay{
			Date: dayStart.Format("2006-01-02"),
		}

		global.DB.Model(&models.Meeting{}).
			Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
			Count(&day.Meetings)

		global.DB.Model(&models.MeetingParticipant{}).
			Where("joined_at >= ? AND joined_at < ?", dayStart, dayEnd).
			Select("COUNT(DISTINCT user_id)").Scan(&day.Participants)

		stats.Days = append(stats.Days, day)
	}

	return stats, nil
}
