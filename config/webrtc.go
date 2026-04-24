package config

type WebRTC struct {
	MaxParticipants int        `yaml:"max_participants"`
	ICEServers      []ICEServer `yaml:"ice_servers"`
}

type ICEServer struct {
	URLs       []string `yaml:"urls"`
	Username   string   `yaml:"username,omitempty"`
	Credential string   `yaml:"credential,omitempty"`
}
