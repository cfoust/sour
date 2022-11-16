package main

import (
	"log"
	"os"

	"github.com/cfoust/sour/pkg/maps"
)

func CountTextures(cube maps.Cube, target map[uint16]int) {
	if cube.Children != nil {
		CountChildTextures(*cube.Children, target)
		return
	}

	for i := 0; i < 6; i++ {
		texture := cube.Texture[i]
		existing, _ := target[texture]
		target[texture] = existing + 1
	}
}

func CountChildTextures(cubes []maps.Cube, target map[uint16]int) {
	for i := 0; i < 8; i++ {
		CountTextures(cubes[i], target)
	}
}

func GetChildTextures(cubes []maps.Cube) map[uint16]int {
	result := make(map[uint16]int)
	CountChildTextures(cubes, result)
	return result
}

func main() {
	args := os.Args[1:]

	if len(args) != 1 {
		log.Fatal("Please provide at least one argument.")
		return
	}

	filename := args[0]
	_map, err := maps.LoadMap(filename)

	if err != nil {
		log.Fatal(err)
	}

	textureRefs := GetChildTextures(_map.Cubes)
	for k, v := range textureRefs {
		log.Printf("[%d]=%d", k, v)
	}
}
