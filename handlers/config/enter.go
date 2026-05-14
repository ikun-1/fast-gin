package config

type IceServerVO struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type IceServersResponse struct {
	IceServers []IceServerVO `json:"iceServers"`
}
