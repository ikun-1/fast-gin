package config

type Recording struct {
	Dir        string `yaml:"dir"`
	MaxSize    int    `yaml:"max_size"`
	FFmpegPath string `yaml:"ffmpeg_path"`
}
