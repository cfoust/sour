package server

import (
	"encoding/json"
	"time"

	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
)

type _Config struct {
	FallbackGameModeID      gamemode.ID  `json:"fallback_game_mode"`
	ServerDescription       string       `json:"server_description"`
	MaxClients              int          `json:"max_clients"`
	MessageOfTheDay         string       `json:"message_of_the_day"`
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
