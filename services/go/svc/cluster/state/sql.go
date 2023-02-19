package state

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Entity struct {
	ID uint `gorm:"primaryKey"`
}

// A "type" of ELO tracking e.g. ffa or insta
type ELOType struct {
	Entity
	Name string `gorm:"size:16"`
}

type Ranking struct {
	Entity

	UserID uint `gorm:"not null"`
	TypeID uint `gorm:"not null"`
	Value  uint

	Type ELOType
}

type User struct {
	Entity

	// The most recent Sauer name the user used
	Nickname string `gorm:"size:15"`

	// Discord unique ID
	UUID string `gorm:"unique;size:32"`
	// The user's Discord username
	Username string `gorm:"size:32"`
	// #1234 or whatever
	Discriminator string `gorm:"size:32"`
	// The prefix for Discord's avatar scheme
	Avatar string `gorm:"size:128"`

	Token        string
	RefreshToken string
	RefreshAfter time.Time

	Ranking []*Ranking
	Spaces  []*Space `gorm:"foreignKey:OwnerID"`
}

type Map struct {
	Entity
	Hash      string `gorm:"not null;size:32"`
	Created   time.Time
	CreatorID uint `gorm:"not null"`

	Creator *User `gorm:"foreignKey:CreatorID"`
}

type Link struct {
	Entity
	Destination   *Space `gorm:"foreignKey:DestinationID"`
	SpaceID       uint   `gorm:"not null"`
	DestinationID uint   `gorm:"not null"`
	Teleport      uint   `gorm:"not null;size:255"`
	Teledest      uint   `gorm:"not null;size:255"`
}

type Space struct {
	Entity

	// Every space is assigned a unique identifier
	UUID string `gorm:"unique"`
	// But it can have a human-readable alias
	Alias       string `gorm:"size:16"`
	Description string `gorm:"size:25"`
	OwnerID     uint
	MapID       uint

	Owner *User `gorm:"foreignKey:OwnerID"`
	Map   *Map
	Links []*Link
}

func InitDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Entity{})
	db.AutoMigrate(&ELOType{})
	db.AutoMigrate(&Ranking{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Map{})
	db.AutoMigrate(&Link{})
	db.AutoMigrate(&Space{})

	return db, nil
}
