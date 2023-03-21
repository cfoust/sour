package state

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	Rating uint
	Wins   uint
	Draws  uint
	Losses uint

	Type *ELOType `gorm:"foreignKey:TypeID"`
}

type Span struct {
	Start time.Time
	End   time.Time
}

// A session in a particular space, server, or map.
type Visit struct {
	Entity
	Span
	SessionID uint   `gorm:"not null"`
	Type      string // map, space, server?
	Location  string
}

// Logs a player's game session from start to finish.
type Session struct {
	Entity
	Span

	UserID uint   // null OK
	UUID   string `gorm:"not null"`
	// The hash of the user's IP
	Address string `gorm:"not null"`
	// desktop, mobile, web
	Device string `gorm:"not null"`

	Visits []*Visit `gorm:"foreignKey:SessionID"`
}

type User struct {
	Entity

	// The most recent Sauer name the user used
	Nickname string `gorm:"size:15"`

	// Discord unique ID
	UUID string `gorm:"unique;uniqueIndex;size:32"`
	// The user's Discord username
	Username string `gorm:"size:32"`
	// #1234 or whatever
	Discriminator string `gorm:"size:32"`
	// The prefix for Discord's avatar scheme
	Avatar string `gorm:"size:128"`

	// For desktop auth
	PublicKey  string
	PrivateKey string

	LastLogin time.Time

	HomeID uint
	Home   *Space `gorm:"foreignKey:HomeID"`

	Ranking  []*Ranking    `gorm:"foreignKey:UserID"`
	Maps     []*MapPointer `gorm:"foreignKey:CreatorID"`
	Spaces   []*Space      `gorm:"foreignKey:OwnerID"`
	Sessions []*Session    `gorm:"foreignKey:UserID"`
}

func (u *User) Reference() string {
	return fmt.Sprintf("%s (%s#%s)", u.Nickname, u.Username, u.Discriminator)
}

// A Discord login code for a user.
type AuthCode struct {
	Entity
	UserID  uint   `gorm:"not null"`
	User    *User  `gorm:"foreignKey:UserID"`
	Value   string `gorm:"not null;unique;uniqueIndex"`
	Expires time.Time
}

type Creatable struct {
	Entity
	Created   time.Time
	CreatorID uint  `gorm:"not null"`
	Creator   *User `gorm:"foreignKey:CreatorID"`
}

func NewCreatable(creator *User) Creatable {
	return Creatable{
		Created:   time.Now(),
		CreatorID: creator.ID,
	}
}

type Asset struct {
	Creatable
	Hash string `gorm:"not null;size:32;unique;uniqueIndex"`
	// The ID of the asset store in the cluster config where this asset can
	// be found
	Location  string `gorm:"not null"`
	Extension string `gorm:"not null"`
	Size      uint   `gorm:"not null"`
	Accessed  time.Time
}

type Aliasable struct {
	UUID string `gorm:"not null;size:32;unique;uniqueIndex"`
	// A human-readable alias
	Alias string `gorm:"unique;uniqueIndex"`
}

type Map struct {
	Creatable

	UUID string `gorm:"not null;size:32;unique;uniqueIndex"`

	OgzID uint   `gorm:"not null"`
	Ogz   *Asset `gorm:"foreignKey:OgzID"`

	CfgID uint   // null OK
	Cfg   *Asset `gorm:"foreignKey:CfgID"`

	DiffID uint     // null OK
	Diff   *MapDiff `gorm:"foreignKey:DiffID"`
}

type MapPointer struct {
	Creatable
	Aliasable

	MapID uint `gorm:"not null"`
	Map   *Map `gorm:"foreignKey:MapID"`
}

type MapDiff struct {
	Entity
	Span

	OldID uint `gorm:"not null"`
	Old   *Map `gorm:"foreignKey:OldID"`

	NewID uint `gorm:"not null"`
	New   *Map `gorm:"foreignKey:NewID"`

	EditsID uint   `gorm:"not null"`
	Edits   *Asset `gorm:"foreignKey:EditsID"`
}

type Link struct {
	Entity
	SpaceID uint `gorm:"not null"`

	DestinationID uint   `gorm:"not null"`
	Destination   *Space `gorm:"foreignKey:DestinationID"`

	Teleport uint `gorm:"not null;size:255"`
	Teledest uint `gorm:"not null;size:255"`
}

type Space struct {
	Creatable
	Aliasable

	Description string `gorm:"size:25"`

	OwnerID uint
	Owner   *User `gorm:"foreignKey:OwnerID"`

	MapPointerID uint `gorm:"not null"`
	MapPointer   *MapPointer

	Links []*Link `gorm:"foreignKey:SpaceID"`
}

func InitDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Entity{})
	db.AutoMigrate(&ELOType{})
	db.AutoMigrate(&Ranking{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&AuthCode{})
	db.AutoMigrate(&Session{})
	db.AutoMigrate(&Visit{})
	db.AutoMigrate(&Asset{})
	db.AutoMigrate(&Map{})
	db.AutoMigrate(&MapPointer{})
	db.AutoMigrate(&MapDiff{})
	db.AutoMigrate(&Link{})
	db.AutoMigrate(&Space{})

	return db, nil
}
