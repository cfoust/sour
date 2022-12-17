package config

import (
	"encoding/json"
	"errors"
	"os"
)

type ServerPreset struct {
	Default bool
	Config string
}

type ServerConfig struct {
	Alias   string
	Preset  string
}

type ENetIngress struct {
	Port    int
	Command string
}

type ClusterIngress struct {
	Desktop []ENetIngress
	Web     struct {
		Port int
	}
}

type ClusterSettings struct {
	Enabled           bool
	Assets            []string
	Presets           map[string]ServerPreset
	Servers           []ServerConfig
	ServerDescription string
	Ingress           ClusterIngress
}

type Config struct {
	Cluster ClusterSettings
}

func GetSourConfig() (*Config, error) {
	configJson, ok := os.LookupEnv("SOUR_CONFIG")
	if !ok {
		return nil, errors.New("SOUR_CONFIG not defined")
	}

	var config Config
	err := json.Unmarshal([]byte(configJson), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
