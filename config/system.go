package config

import "fmt"

type System struct {
	Mode    string  `yaml:"mode"`
	IP      string  `yaml:"ip"`
	Port    int     `yaml:"port"`
	Swagger Swagger `yaml:"swagger"`
}

type Swagger struct {
	Enabled bool   `yaml:"enabled"`
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

func (s System) Addr() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port)
}
