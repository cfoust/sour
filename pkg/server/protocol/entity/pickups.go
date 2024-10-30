package entity

import (
	"github.com/cfoust/sour/pkg/server/protocol/sound"
)

type Pickup struct {
	Typ       ID
	Sound     sound.ID
	Amount    int32
	MaxAmount int32
}

var Pickups map[ID]Pickup = map[ID]Pickup{
	PickupShotgun:         Pickup{PickupShotgun, sound.PickUpAmmo, 10, 30},
	PickupMinigun:         Pickup{PickupMinigun, sound.PickUpAmmo, 20, 60},
	PickupRocketLauncher:  Pickup{PickupRocketLauncher, sound.PickUpAmmo, 5, 15},
	PickupRifle:           Pickup{PickupRifle, sound.PickUpAmmo, 5, 15},
	PickupGrenadeLauncher: Pickup{PickupGrenadeLauncher, sound.PickUpAmmo, 10, 30},
	PickupPistol:          Pickup{PickupPistol, sound.PickUpAmmo, 30, 120},
	PickupHealth:          Pickup{PickupHealth, sound.PickUpHealth, 25, 100},
	PickupBoost:           Pickup{PickupBoost, sound.PickUpHealth, 50, 200},
	PickupGreenArmour:     Pickup{PickupGreenArmour, sound.PickUpArmour, 100, 100},
	PickupYellowArmor:     Pickup{PickupYellowArmor, sound.PickUpArmour, 200, 200},
	PickupQuadDamage:      Pickup{PickupQuadDamage, sound.PickUpQuaddamage, 20000, 30000},
}
