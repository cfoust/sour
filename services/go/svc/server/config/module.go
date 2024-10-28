package config

import (
	"encoding/json"
	"fmt"

	"github.com/cfoust/sour/pkg/server"
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
	Default bool
	Config  server.Config
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
	DBPath            string
	LogDirectory      string
	CacheDirectory    string
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

type StoreType uint8

const (
	StoreTypeFS StoreType = iota
)

type StoreConfig interface {
	Type() StoreType
}

type FSStoreConfig struct {
	Path string
}

func (f FSStoreConfig) Type() StoreType { return StoreTypeFS }

type Store struct {
	Name    string
	Default bool
	Config  StoreConfig
}

func (s *Store) UnmarshalJSON(data []byte) error {
	var obj map[string]*json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	err := json.Unmarshal(*obj["name"], &s.Name)
	if err != nil {
		return err
	}

	err = json.Unmarshal(*obj["default"], &s.Default)
	if err != nil {
		return err
	}

	config, ok := obj["config"]
	if !ok {
		return fmt.Errorf("config key missing")
	}

	// Read the config
	if err := json.Unmarshal(*obj["config"], &obj); err != nil {
		return err
	}

	var type_ string
	err = json.Unmarshal(*obj["type"], &type_)
	if err != nil {
		return err
	}

	switch type_ {
	case "fs":
		var fs FSStoreConfig
		if err := json.Unmarshal(*config, &fs); err != nil {
			return err
		}
		s.Config = fs
	default:
		return fmt.Errorf("invalid store type: %s", type_)
	}

	return nil
}

type Config struct {
	Redis       RedisSettings
	Cluster     ClusterSettings
	AssetStores []Store
	Discord     DiscordSettings
}

func GetSourConfig(data []byte) (*Config, error) {
	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
