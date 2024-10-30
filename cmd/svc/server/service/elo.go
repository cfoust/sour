package service

import (
	"context"
	"sync"

	"github.com/cfoust/sour/svc/server/config"
	"github.com/cfoust/sour/svc/server/state"

	"gorm.io/gorm"
)

type ELO struct {
	Rating uint
	Wins   uint
	Draws  uint
	Losses uint
}

func NewELO() *ELO {
	return &ELO{
		Rating: 1200,
	}
}

func getType(ctx context.Context, db *gorm.DB, matchType string) (*state.ELOType, error) {
	var type_ state.ELOType
	err := db.WithContext(ctx).Where(state.ELOType{
		Name: matchType,
	}).First(&type_).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err == nil {
		return &type_, nil
	}

	// it doesn't exist
	type_ = state.ELOType{
		Name: matchType,
	}

	err = db.WithContext(ctx).Create(&type_).Error
	if err != nil {
		return nil, err
	}

	return &type_, nil
}

func getRanking(ctx context.Context, db *gorm.DB, user *state.User, matchType string) (*state.Ranking, error) {
	type_, err := getType(ctx, db, matchType)
	if err != nil {
		return nil, err
	}

	var ranking state.Ranking
	err = db.WithContext(ctx).Where(state.Ranking{
		UserID: user.ID,
		TypeID: type_.ID,
	}).First(&ranking).Error
	if err != nil {
		return nil, err
	}

	return &ranking, nil
}

func (e *ELO) SaveState(ctx context.Context, db *gorm.DB, user *state.User, matchType string) error {
	ranking, err := getRanking(ctx, db, user, matchType)

	if err == gorm.ErrRecordNotFound {
		type_, err := getType(ctx, db, matchType)
		if err != nil {
			return err
		}

		ranking := state.Ranking{
			UserID: user.ID,
			TypeID: type_.ID,
			Rating: e.Rating,
			Wins:   e.Wins,
			Losses: e.Losses,
			Draws:  e.Draws,
		}

		return db.WithContext(ctx).Create(&ranking).Error
	}

	if err != nil {
		return err
	}

	ranking.Rating = e.Rating
	ranking.Wins = e.Wins
	ranking.Losses = e.Losses
	ranking.Draws = e.Draws

	return db.WithContext(ctx).Save(&ranking).Error
}

func LoadELOState(ctx context.Context, db *gorm.DB, user *state.User, matchType string) (*ELO, error) {
	ranking, err := getRanking(ctx, db, user, matchType)

	if err == gorm.ErrRecordNotFound {
		type_, err := getType(ctx, db, matchType)
		if err != nil {
			return nil, err
		}

		e := NewELO()

		ranking := state.Ranking{
			UserID: user.ID,
			TypeID: type_.ID,
			Rating: e.Rating,
			Wins:   e.Wins,
			Losses: e.Losses,
			Draws:  e.Draws,
		}

		err = db.WithContext(ctx).Create(&ranking).Error
		if err != nil {
			return nil, err
		}

		return e, nil
	}

	if err != nil {
		return nil, err
	}

	return &ELO{
		Rating: ranking.Rating,
		Wins:   ranking.Wins,
		Losses: ranking.Losses,
		Draws:  ranking.Draws,
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
