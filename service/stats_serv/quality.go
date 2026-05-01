package stats_serv

import (
	"fast-gin/global"
	"fast-gin/models"
	"math"
	"sort"
)

type QualityMetricSummary struct {
	Label          string  `json:"label"`
	AvgPacketsLost float64 `json:"avgPacketsLost"`
	AvgJitterMs    float64 `json:"avgJitterMs"`
	AvgRoundTripMs float64 `json:"avgRoundTripMs"`
	AvgBitrateKbps float64 `json:"avgBitrateKbps"`
	MinBitrateKbps float64 `json:"minBitrateKbps"`
	MaxBitrateKbps float64 `json:"maxBitrateKbps"`
	AvgFPS         float64 `json:"avgFps,omitempty"`
	MaxFrameWidth  int     `json:"maxFrameWidth,omitempty"`
	MaxFrameHeight int     `json:"maxFrameHeight,omitempty"`
	SampleCount    int64   `json:"sampleCount"`
}

type CandidateTypeDistribution struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

type UserQualitySummary struct {
	UserID        uint                 `json:"userId"`
	ClientID      string               `json:"clientId"`
	DisplayName   string               `json:"displayName"`
	Audio         *QualityMetricSummary `json:"audio,omitempty"`
	Video         *QualityMetricSummary `json:"video,omitempty"`
	CandidateType string               `json:"candidateType,omitempty"`
}

type MeetingQualityReport struct {
	MeetingID          uint                      `json:"meetingId"`
	RoomNo             uint                      `json:"roomNo"`
	Title              string                    `json:"title"`
	UserCount          int                       `json:"userCount"`
	Users              []UserQualitySummary      `json:"users"`
	CandidateDist      []CandidateTypeDistribution `json:"candidateDist"`
	OverallAvgJitterMs float64                   `json:"overallAvgJitterMs"`
	OverallAvgRttMs    float64                   `json:"overallAvgRttMs"`
	OverallAvgLossRate float64                   `json:"overallAvgPacketLossRate"`
}

func GetMeetingQualityReport(meetingID uint) (*MeetingQualityReport, error) {
	var meeting models.Meeting
	if err := global.DB.First(&meeting, meetingID).Error; err != nil {
		return nil, err
	}

	var snapshots []models.MeetingQualitySnapshot
	global.DB.Where("meeting_id = ?", meetingID).
		Order("user_id, label, snapshot_at ASC").
		Find(&snapshots)

	if len(snapshots) == 0 {
		return &MeetingQualityReport{
			MeetingID: meeting.ID,
			RoomNo:    meeting.RoomNo,
			Title:     meeting.Title,
		}, nil
	}

	// Group by user+clientID, then by label
	type userKey struct {
		UserID   uint
		ClientID string
	}
	userGroups := make(map[userKey]map[string][]models.MeetingQualitySnapshot)
	userDisplayNames := make(map[userKey]string)

	for _, s := range snapshots {
		key := userKey{UserID: s.UserID, ClientID: s.ClientID}
		if userGroups[key] == nil {
			userGroups[key] = make(map[string][]models.MeetingQualitySnapshot)
		}
		userGroups[key][s.Label] = append(userGroups[key][s.Label], s)

		// Get display name from any snapshot of this user
		if userDisplayNames[key] == "" {
			// Try to get from participant record
			var participant models.MeetingParticipant
			if err := global.DB.Where("meeting_id = ? AND user_id = ?", meetingID, s.UserID).
				First(&participant).Error; err == nil {
				userDisplayNames[key] = participant.DisplayName
			}
		}
	}

	// Compute per-user summaries
	type labelSummary struct {
		label   string
		summary *QualityMetricSummary
	}
	var allJitters []float64
	var allRTTs []float64
	var totalPacketsLost int64
	var totalPackets int64

	users := make([]UserQualitySummary, 0)
	for key, labelMap := range userGroups {
		u := UserQualitySummary{
			UserID:   key.UserID,
			ClientID: key.ClientID,
		}
		if name, ok := userDisplayNames[key]; ok {
			u.DisplayName = name
		}

		for label, samples := range labelMap {
			summary := computeMetricSummary(label, samples)
			switch label {
			case "audio":
				u.Audio = summary
			case "video":
				u.Video = summary
			}

			// Collect overall aggregates
			if label == "audio" || label == "video" {
				for _, s := range samples {
					if s.JitterMs > 0 {
						allJitters = append(allJitters, s.JitterMs)
					}
					if s.RoundTripMs > 0 {
						allRTTs = append(allRTTs, s.RoundTripMs)
					}
					if s.PacketsLost > 0 || s.PacketsReceived > 0 {
						totalPacketsLost += s.PacketsLost
						totalPackets += s.PacketsLost + s.PacketsReceived
					}
				}
			}

			// Get candidate type from connection samples
			if label == "connection" && len(samples) > 0 {
				for _, s := range samples {
					if s.CandidateType != "" {
						u.CandidateType = s.CandidateType
						break
					}
				}
			}
		}

		users = append(users, u)
	}

	// Sort users by UserID for stable output
	sort.Slice(users, func(i, j int) bool {
		return users[i].UserID < users[j].UserID
	})

	// Compute candidate type distribution
	candidateCounts := make(map[string]int64)
	for _, s := range snapshots {
		if s.Label == "connection" && s.CandidateType != "" {
			candidateCounts[s.CandidateType]++
		}
	}
	candidateDist := make([]CandidateTypeDistribution, 0)
	for ct, count := range candidateCounts {
		candidateDist = append(candidateDist, CandidateTypeDistribution{Type: ct, Count: count})
	}
	sort.Slice(candidateDist, func(i, j int) bool {
		return candidateDist[i].Count > candidateDist[j].Count
	})

	// Compute overall averages
	overallAvgJitter := mean(allJitters)
	overallAvgRTT := mean(allRTTs)
	overallLossRate := 0.0
	if totalPackets > 0 {
		overallLossRate = float64(totalPacketsLost) / float64(totalPackets) * 100
	}

	return &MeetingQualityReport{
		MeetingID:          meeting.ID,
		RoomNo:             meeting.RoomNo,
		Title:              meeting.Title,
		UserCount:          len(users),
		Users:              users,
		CandidateDist:      candidateDist,
		OverallAvgJitterMs: math.Round(overallAvgJitter*100) / 100,
		OverallAvgRttMs:    math.Round(overallAvgRTT*100) / 100,
		OverallAvgLossRate: math.Round(overallLossRate*100) / 100,
	}, nil
}

func computeMetricSummary(label string, samples []models.MeetingQualitySnapshot) *QualityMetricSummary {
	if len(samples) == 0 {
		return nil
	}

	s := &QualityMetricSummary{Label: label, SampleCount: int64(len(samples))}

	var jitterSum, rttSum, bitrateSum, fpsSum float64
	var bitrateCount, fpsCount int64
	first := true

	for _, sample := range samples {
		s.AvgPacketsLost += float64(sample.PacketsLost)
		jitterSum += sample.JitterMs
		rttSum += sample.RoundTripMs

		if sample.BitrateKbps > 0 {
			bitrateSum += sample.BitrateKbps
			bitrateCount++
			if first || sample.BitrateKbps < s.MinBitrateKbps {
				s.MinBitrateKbps = sample.BitrateKbps
			}
			if first || sample.BitrateKbps > s.MaxBitrateKbps {
				s.MaxBitrateKbps = sample.BitrateKbps
			}
		}
		if label == "video" {
			fpsSum += sample.FPS
			if sample.FPS > 0 {
				fpsCount++
			}
			if sample.FrameWidth > s.MaxFrameWidth {
				s.MaxFrameWidth = sample.FrameWidth
			}
			if sample.FrameHeight > s.MaxFrameHeight {
				s.MaxFrameHeight = sample.FrameHeight
			}
		}
		first = false
	}

	s.AvgJitterMs = safeAvg(jitterSum, int64(len(samples)))
	s.AvgRoundTripMs = safeAvg(rttSum, int64(len(samples)))
	s.AvgBitrateKbps = safeAvg(bitrateSum, bitrateCount)
	s.AvgFPS = safeAvg(fpsSum, fpsCount)

	return s
}

func safeAvg(sum float64, count int64) float64 {
	if count == 0 {
		return 0
	}
	return math.Round(sum/float64(count)*100) / 100
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
