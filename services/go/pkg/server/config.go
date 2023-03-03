package server

import (
	"encoding/json"
	"time"

	"github.com/cfoust/sour/pkg/server/maprot"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
)

type _Config struct {
	ListenAddress string `json:"listen_address"`
	ListenPort    int    `json:"listen_port"`

	MasterServerAddress     string       `json:"master_server_address"`
	StatsServerAddress      string       `json:"stats_server_address"`
	StatsServerAuthDomain   string       `json:"stats_server_auth_domain"`
	FallbackGameModeID      gamemode.ID  `json:"fallback_game_mode"`
	ServerDescription       string       `json:"server_description"`
	MaxClients              int          `json:"max_clients"`
	SendClientIPsViaExtinfo bool         `json:"send_client_ips_via_extinfo"`
	MessageOfTheDay         string       `json:"message_of_the_day"`
	AuthDomain              string       `json:"auth_domain"`
	MapPools                maprot.Pools `json:"maps"`
}

type Config struct {
	_Config
	GameDuration time.Duration
}

func (c *Config) UnmarshalJSON(data []byte) error {
	proxy := struct {
		_Config
		GameDuration string `json:"game_duration"`
	}{}
	err := json.Unmarshal(data, &proxy)
	if err != nil {
		return err
	}

	c._Config = proxy._Config

	c.GameDuration, err = time.ParseDuration(proxy.GameDuration)
	if err != nil {
		return err
	}

	return nil
}
