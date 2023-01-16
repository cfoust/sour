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

type SpaceLink struct {
	ID          uint8
	Destination string
}

type SpaceConfig struct {
	Alias       string
	Description string
	Links       []SpaceLink
}

type PresetSpace struct {
	Preset string
	Config SpaceConfig
}

type ENetServerInfo struct {
	Enabled bool
	Master  bool
	Cluster bool
}

type ENetIngress struct {
	Port       int
	Target     string
	ServerInfo ENetServerInfo
}

type ClusterIngress struct {
	Desktop []ENetIngress
	Web     struct {
		Port int
	}
}

type ClusterServerInfo struct {
	Map         string
	Description string
	TimeLeft    int
	GameSpeed   int
}

type MatchmakingSettings struct {
	Duel []DuelType
}

type ClusterSettings struct {
	Enabled           bool
	LogSessions       bool
	ServerInfo        ClusterServerInfo
	Assets            []string
	Presets           []ServerPreset
	Spaces            []PresetSpace
	Matchmaking       MatchmakingSettings
	ServerDescription string
	Ingress           ClusterIngress
}

type DiscordSettings struct {
	Enabled     bool
	Domain      string
	Id          string
	Secret      string
	RedirectURI string
}

type RedisSettings struct {
	Address  string
	Password string
	DB       int
}

type Config struct {
	Redis   RedisSettings
	Cluster ClusterSettings
	Discord DiscordSettings
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
