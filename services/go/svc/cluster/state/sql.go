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

type Span struct {
	Start time.Time
	End   time.Time
}

type Visit struct {
	Entity
	Span
	SessionID uint   `gorm:"not null"`
	Type      string // map, space, server?
	Location  string
}

type Session struct {
	Entity
	Span

	UserID uint
	UUID   string
	IP     string

	Visits []*Visit `gorm:"foreignKey:SessionID"`
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

	HomeID uint
	Home   *Space `gorm:"foreignKey:HomeID"`

	Ranking  []*Ranking
	Spaces   []*Space   `gorm:"foreignKey:OwnerID"`
	Sessions []*Session `gorm:"foreignKey:UserID"`
}

type Creatable struct {
	Entity
	Created   time.Time
	CreatorID uint  `gorm:"not null"`
	Creator   *User `gorm:"foreignKey:CreatorID"`
}

type Asset struct {
	Creatable
	Hash string `gorm:"not null;size:32"`
	// The ID of the asset store in the cluster config where this asset can
	// be found
	Location  string `gorm:"not null"`
	Extension string `gorm:"not null"`
	Size      uint   `gorm:"not null"`
}

type Map struct {
	Creatable
	Hash string `gorm:"not null;size:32"`

	// The asset's hash will be the same as the map hash
	AssetID uint   `gorm:"not null"`
	Asset   *Asset `gorm:"foreignKey:AssetID"`
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
	Creatable

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
	db.AutoMigrate(&Session{})
	db.AutoMigrate(&Visit{})
	db.AutoMigrate(&Asset{})
	db.AutoMigrate(&Map{})
	db.AutoMigrate(&Link{})
	db.AutoMigrate(&Space{})

	return db, nil
}
