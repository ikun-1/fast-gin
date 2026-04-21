package signal_serv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fast-gin/dal/query"
)

type ConnectRequest struct {
	RoomNo string
	UserID uint
}

type SignalMessage struct {
	Type   string
	RoomNo string
	From   uint
	To     uint
	Data   map[string]any
}

type ConnectionInfo struct {
	RoomNo string `json:"room_no"`
	UserID uint   `json:"user_id"`
	Online int    `json:"online"`
}

func (s *ServiceStruct) Connect(req ConnectRequest) (*ConnectionInfo, error) {
	room, err := query.MeetingRoom.WithContext(context.TODO()).Where(query.MeetingRoom.RoomNo.Eq(req.RoomNo)).Take()
	if err != nil {
		return nil, err
	}
	_ = room
	HubInstance.Register(&Client{RoomNo: req.RoomNo, UserID: req.UserID, Send: make(chan []byte, 32)})
	return &ConnectionInfo{RoomNo: req.RoomNo, UserID: req.UserID, Online: HubInstance.ClientCount(req.RoomNo)}, nil
}

func (s *ServiceStruct) Disconnect(req ConnectRequest) error {
	HubInstance.Unregister(&Client{RoomNo: req.RoomNo, UserID: req.UserID})
	return nil
}

func (s *ServiceStruct) HandleMessage(msg SignalMessage) error {
	if msg.RoomNo == "" {
		return errors.New("room no required")
	}

	payload := map[string]any{
		"type":      msg.Type,
		"roomNo":    msg.RoomNo,
		"from":      msg.From,
		"to":        msg.To,
		"data":      msg.Data,
		"timestamp": time.Now().Unix(),
	}

	if msg.Type == "raise_hand" {
		payload["type"] = "member_state_change"
		payload["data"] = map[string]any{
			"userId":     msg.From,
			"raisedHand": msg.Data["raisedHand"],
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	HubInstance.Broadcast(msg.RoomNo, b)
	return nil
}

func (s *ServiceStruct) NewWSURL(roomNo string) string {
	return fmt.Sprintf("/api/signal/ws?room_no=%s", roomNo)
}
