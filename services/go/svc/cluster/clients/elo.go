package clients

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/cfoust/sour/svc/cluster/config"

	"github.com/go-redis/redis/v9"
)

type ELO struct {
	Rating int
	Wins   int
	Draws  int
	Losses int
}

func NewELO() *ELO {
	return &ELO{
		Rating: 1200,
	}
}

func getField(matchType string, id string, field string) string {
	return fmt.Sprintf("elo-%s-%s-%s", matchType, id, field)
}

func (e *ELO) SaveState(ctx context.Context, redis *redis.Client, id string, matchType string) error {
	pipe := redis.Pipeline()

	pipe.Set(ctx, getField(matchType, id, "rating"), e.Rating, 0)
	pipe.Set(ctx, getField(matchType, id, "wins"), e.Wins, 0)
	pipe.Set(ctx, getField(matchType, id, "draws"), e.Draws, 0)
	pipe.Set(ctx, getField(matchType, id, "losses"), e.Losses, 0)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func LoadELOState(ctx context.Context, redis *redis.Client, id string, matchType string) (*ELO, error) {
	pipe := redis.Pipeline()

	rating := pipe.Get(ctx, getField(matchType, id, "rating"))
	wins := pipe.Get(ctx, getField(matchType, id, "wins"))
	draws := pipe.Get(ctx, getField(matchType, id, "draws"))
	losses := pipe.Get(ctx, getField(matchType, id, "losses"))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	ratingVal, _ := strconv.Atoi(rating.Val())
	winsVal, _ := strconv.Atoi(wins.Val())
	drawsVal, _ := strconv.Atoi(draws.Val())
	lossesVal, _ := strconv.Atoi(losses.Val())

	return &ELO{
		Rating: ratingVal,
		Wins:   winsVal,
		Draws:  drawsVal,
		Losses: lossesVal,
	}, nil
}

type ELOState struct {
	Ratings map[string]*ELO
	Mutex   sync.Mutex
}

func NewELOState(duels []config.DuelType) *ELOState {
	state := ELOState{
		Ratings: make(map[string]*ELO),
	}

	for _, type_ := range duels {
		state.Ratings[type_.Name] = NewELO()
	}

	return &state
}
