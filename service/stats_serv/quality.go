package stats_serv

import (
	"fast-gin/global"
	"fast-gin/models"
	"math"
	"sort"

	"go.uber.org/zap"
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

	zap.S().Infof("GetMeetingQualityReport: meetingID=%d snapshots=%d", meetingID, len(snapshots))

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
	var allJitters []float64
	var allRTTs []float64
	// Cumulative packet loss: use the latest snapshot values per user+label
	// since browser getStats() reports cumulative counters, not deltas.
	type lossKey struct {
		userID uint
		label  string
	}
	latestLoss := make(map[lossKey]struct {
		packetsLost     int64
		packetsReceived int64
	})

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
					// Only inbound samples (PacketsReceived > 0) have meaningful loss data
					if s.PacketsReceived > 0 {
						lk := lossKey{userID: key.UserID, label: label}
						cur := latestLoss[lk]
						if s.PacketsReceived > cur.packetsReceived {
							lost := s.PacketsLost
							if lost < 0 {
								lost = 0
							}
							latestLoss[lk] = struct {
								packetsLost     int64
								packetsReceived int64
							}{packetsLost: lost, packetsReceived: s.PacketsReceived}
						}
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
	var sumLost, sumTotal int64
	for _, v := range latestLoss {
		sumLost += v.packetsLost
		sumTotal += v.packetsLost + v.packetsReceived
	}
	if sumTotal > 0 {
		overallLossRate = float64(sumLost) / float64(sumTotal) * 100
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
	var jitterCount, rttCount, bitrateCount, fpsCount int64
	var packetsLostSum int64
	var framesWithRecv int64
	first := true

	for _, sample := range samples {
		// Only count JitterMs from inbound (bytesReceived > 0) or non-zero values
		if sample.JitterMs > 0 || sample.BytesReceived > 0 {
			jitterSum += sample.JitterMs
			jitterCount++
		}
		// Only count RoundTripMs from non-zero values
		if sample.RoundTripMs > 0 {
			rttSum += sample.RoundTripMs
			rttCount++
		}
		// PacketsLost: only count inbound samples (PacketsReceived > 0);
		// browser may report -1 during initial connection, clamp to 0
		if sample.PacketsReceived > 0 {
			lost := sample.PacketsLost
			if lost < 0 {
				lost = 0
			}
			packetsLostSum += lost
			framesWithRecv++
		}

		if sample.BitrateKbps > 0 {
			bitrateSum += sample.BitrateKbps
			bitrateCount++
			if first || sample.BitrateKbps < s.MinBitrateKbps {
				s.MinBitrateKbps = sample.BitrateKbps
			}
			if first || sample.BitrateKbps > s.MaxBitrateKbps {
				s.MaxBitrateKbps = sample.BitrateKbps
			}
			first = false
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
	}

	s.AvgJitterMs = safeAvg(jitterSum, jitterCount)
	s.AvgRoundTripMs = safeAvg(rttSum, rttCount)
	s.AvgBitrateKbps = safeAvg(bitrateSum, bitrateCount)
	s.AvgFPS = safeAvg(fpsSum, fpsCount)
	// Average packets lost per sample that has receive/loss data
	if framesWithRecv > 0 {
		s.AvgPacketsLost = math.Round(float64(packetsLostSum)/float64(framesWithRecv)*100) / 100
	}

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
