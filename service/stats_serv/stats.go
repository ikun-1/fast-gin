package stats_serv

import (
	"context"
	"fast-gin/dal/query"
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
func GetOverviewStats(ctx context.Context) (*OverviewStats, error) {
	stats := new(OverviewStats)
	var err error

	stats.TotalMeetings, err = query.Meeting.WithContext(ctx).Count()
	if err != nil {
		return nil, err
	}

	stats.ActiveMeetings, err = query.Meeting.WithContext(ctx).Where(query.Meeting.Status.Eq("active")).Count()
	if err != nil {
		return nil, err
	}

	stats.TotalParticipants, err = query.MeetingParticipant.WithContext(ctx).Distinct(query.MeetingParticipant.UserID).Count()
	if err != nil {
		return nil, err
	}

	stats.TotalRecordings, err = query.Recording.WithContext(ctx).Count()
	if err != nil {
		return nil, err
	}

	// Calculate total duration from ended meetings using Go-level time arithmetic
	meetings, err := query.Meeting.WithContext(ctx).Where(
		query.Meeting.Status.Eq("ended"),
		query.Meeting.StartedAt.IsNotNull(),
		query.Meeting.EndedAt.IsNotNull(),
	).Find()
	if err != nil {
		return nil, err
	}
	for _, m := range meetings {
		stats.TotalDurationMs += m.EndedAt.Sub(*m.StartedAt).Milliseconds()
	}

	return stats, nil
}

// GetMeetingStats returns participation details for a single meeting.
func GetMeetingStats(ctx context.Context, meetingID uint) (*MeetingStats, error) {
	meeting, err := query.Meeting.WithContext(ctx).Where(query.Meeting.ID.Eq(meetingID)).First()
	if err != nil {
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

	participants, err := query.MeetingParticipant.WithContext(ctx).
		Where(query.MeetingParticipant.MeetingID.Eq(meetingID)).
		Order(query.MeetingParticipant.JoinedAt.Asc()).
		Find()
	if err != nil {
		return nil, err
	}

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
func GetUserStats(ctx context.Context, userID uint) (*UserStats, error) {
	user, err := query.User.WithContext(ctx).Where(query.User.ID.Eq(userID)).First()
	if err != nil {
		return nil, err
	}

	stats := &UserStats{
		UserID:   user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
	}

	// Count total meetings attended
	stats.TotalMeetings, _ = query.MeetingParticipant.WithContext(ctx).
		Where(query.MeetingParticipant.UserID.Eq(userID)).
		Distinct(query.MeetingParticipant.MeetingID).
		Count()

	// Count meetings hosted
	stats.MeetingsHosted, _ = query.Meeting.WithContext(ctx).
		Where(query.Meeting.HostID.Eq(userID)).
		Count()

	// Fetch recent meeting participations
	participants, err := query.MeetingParticipant.WithContext(ctx).
		Where(query.MeetingParticipant.UserID.Eq(userID)).
		Order(query.MeetingParticipant.JoinedAt.Desc()).
		Limit(20).
		Find()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, p := range participants {
		leftAt := p.LeftAt
		if leftAt == nil {
			leftAt = &now
		}
		durationMs := max(leftAt.Sub(p.JoinedAt).Milliseconds(), 0)
		stats.TotalDurationMs += durationMs

		// Fetch meeting info
		title := ""
		meeting, err := query.Meeting.WithContext(ctx).Where(query.Meeting.ID.Eq(p.MeetingID)).First()
		if err == nil {
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
func GetTrendStats(ctx context.Context, days int) (*TrendStats, error) {
	stats := &TrendStats{}

	startDate := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)

	for i := 0; i < days; i++ {
		dayStart := startDate.AddDate(0, 0, i)
		dayEnd := dayStart.Add(24 * time.Hour)

		day := TrendDay{
			Date: dayStart.Format("2006-01-02"),
		}

		day.Meetings, _ = query.Meeting.WithContext(ctx).
			Where(query.Meeting.CreatedAt.Gte(dayStart), query.Meeting.CreatedAt.Lt(dayEnd)).
			Count()

		day.Participants, _ = query.MeetingParticipant.WithContext(ctx).
			Where(query.MeetingParticipant.JoinedAt.Gte(dayStart), query.MeetingParticipant.JoinedAt.Lt(dayEnd)).
			Distinct(query.MeetingParticipant.UserID).
			Count()

		stats.Days = append(stats.Days, day)
	}

	return stats, nil
}
