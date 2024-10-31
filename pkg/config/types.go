package config

import (
	"github.com/cfoust/sour/pkg/gameserver"
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

type Preset struct {
	Name    string
	Virtual bool
	Default bool
	Config  gameserver.Config
}

type SpaceLink struct {
	Teleport    uint8
	Teledest    uint8
	Destination string
}

type SpaceConfig struct {
	Alias       string
	Description string
	Links       []SpaceLink
}

type PresetSpace struct {
	Preset          string
	VotingCreates   bool
	ExploreMode     bool
	ExploreModeSkip string
	Config          SpaceConfig
}

type ENetServerInfo struct {
	Enabled bool
	Master  bool
	Server  bool
}

type ENetIngress struct {
	Port       int
	Target     string
	ServerInfo ENetServerInfo
}

type ServerIngress struct {
	Desktop []ENetIngress
	Web     struct {
		Port int
	}
}

type ServerServerInfo struct {
	Map         string
	Description string
	TimeLeft    int
	GameSpeed   int
}

type MatchmakingSettings struct {
	Duel []DuelType
}

type ServerSettings struct {
	LogSessions       bool
	DBPath            string
	LogDirectory      string
	CacheDirectory    string
	ServerInfo        ServerServerInfo
	Assets            []string
	Presets           []Preset
	Spaces            []PresetSpace
	Matchmaking       MatchmakingSettings
	ServerDescription string
	Ingress           ServerIngress
}

type ClientSettings struct {
	Assets      []string `json:"assets"`
	Servers     []string `json:"servers"`
	Proxy       string   `json:"proxy"`
	MenuOptions string   `json:"menuOptions"`
}

type Config struct {
	Server ServerSettings
	Client ClientSettings
}
