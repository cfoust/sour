package config

import (
	"encoding/json"
	"errors"
	"os"
)

type RespawnType string

const (
	RespawnTypeAll  = "all"
	RespawnTypeDead = "dead"
	RespawnTypeNone = "none"
)

type DuelType struct {
	Name            string
	Preset          string
	ForceRespawn    RespawnType
	WarmupSeconds   uint
	GameSeconds     uint
	WinThreshold    uint
	OvertimeSeconds uint
	PauseOnDeath    bool
	Default         bool
}

type ServerPreset struct {
	Name    string
	Virtual bool
	Inherit string
	Default bool
	Config  string
}

type ServerConfig struct {
	Alias  string
	Preset string
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

type MatchmakingSettings struct {
	Duel []DuelType
}

type ClusterSettings struct {
	Enabled           bool
	Assets            []string
	Presets           []ServerPreset
	Servers           []ServerConfig
	Matchmaking       MatchmakingSettings
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
