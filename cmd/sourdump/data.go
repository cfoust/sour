package main

import (
	_ "embed"
	"fmt"
)

//go:embed base.list
var baseListData string

var HUDGUNS = []string{
	"chaing",
	"fist",
	"gl",
	"pistol",
	"rifle",
	"rocket",
	"shotg",
}

func expandHudguns(prefix string) []string {
	result := []string{prefix}
	for _, gun := range HUDGUNS {
		for _, suffix := range []string{"", "/blue", "/red"} {
			result = append(result, fmt.Sprintf("%s/%s%s", prefix, gun, suffix))
		}
	}
	return result
}

var BASE_MODELS = []string{
	"ammo/bullets",
	"ammo/cartridges",
	"ammo/grenades",
	"ammo/rockets",
	"ammo/rrounds",
	"ammo/shells",
	"armor/green",
	"armor/yellow",
	"boost",
	"carrot",
	"checkpoint",
	"health",
	"quad",
	"teleporter",
	"flags/neutral",
	"flags/red",
	"flags/blue",
	"base/red",
	"base/neutral",
	"base/blue",
	"skull/red",
	"skull/blue",
}

var SNOUT_MODELS = append([]string{
	"snoutx10k",
	"snoutx10k/armor/blue",
	"snoutx10k/armor/green",
	"snoutx10k/armor/yellow",
	"snoutx10k/blue",
	"snoutx10k/red",
	"snoutx10k/wings",
}, expandHudguns("snoutx10k/hudguns")...)

func initOtherModels() []string {
	models := []string{
		"captaincannon",
		"captaincannon/armor/blue",
		"captaincannon/armor/green",
		"captaincannon/armor/yellow",
		"captaincannon/blue",
		"captaincannon/quad",
		"captaincannon/red",
		"inky",
		"inky/armor/blue",
		"inky/armor/green",
		"inky/armor/yellow",
		"inky/blue",
		"inky/quad",
		"inky/red",
		"mrfixit",
		"mrfixit/armor/blue",
		"mrfixit/armor/green",
		"mrfixit/armor/yellow",
		"mrfixit/blue",
		"mrfixit/horns",
		"mrfixit/red",
		"ogro2",
		"ogro2/armor/blue",
		"ogro2/armor/green",
		"ogro2/armor/yellow",
		"ogro2/blue",
		"ogro2/quad",
		"ogro2/red",
	}

	models = append(models, expandHudguns("captaincannon/hudguns")...)
	models = append(models, expandHudguns("inky/hudguns")...)
	models = append(models, expandHudguns("mrfixit/hudguns")...)

	return models
}

var OTHER_MODELS = initOtherModels()
