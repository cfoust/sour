package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/repeale/fp-go"

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

type RootFlags []string

func (flags *RootFlags) String() string {
	return "N/I"
}

func (flags *RootFlags) Set(value string) error {
	*flags = append(*flags, value)
	return nil
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

// Search for a file in the roots, one at a time
func SearchFile(roots []string, path string) *string {
	for i := 0; i < len(roots); i++ {
		needle := filepath.Join(roots[i], path)
		if FileExists(needle) {
			return &needle
		}
	}

	return nil
}

type Texture struct {
	Slot int
	Path string
}

type Processor struct {
	Slot int
	// All textures referenced and their calculated slot
	Textures []Texture
	// File references are guaranteed to be included and do not have a slot
	Files []string
}

func NewProcessor() *Processor {
	processor := Processor{}

	processor.Slot = 0
	processor.Textures = make([]Texture, 0)
	processor.Files = make([]string, 0)

	return &processor
}

func (processor *Processor) AddTexture(path string) {
	texture := Texture{}

	texture.Slot = processor.Slot
	texture.Path = path
	processor.Slot++

	processor.Textures = append(processor.Textures, texture)
}

func (processor *Processor) ResetTextures() {
	processor.Slot = 0
	processor.Textures = make([]Texture, 0)
}

func (processor *Processor) AddFile(path string) {
	processor.Files = append(processor.Files, path)
}

var (
	COMMAND_REGEX = regexp.MustCompile(`(("[^"]*")|([^\s]+))`)
)

func ParseLine(line string) []string {
	empty := make([]string, 0)

	// Split off the comments
	parts := strings.Split(line, "//")

	if len(parts) == 0 {
		return empty
	}

	command := strings.TrimSpace(parts[0])

	if len(command) == 0 {
		return empty
	}

	// Break the command up into pieces, preserving quoted arguments
	matches := COMMAND_REGEX.FindAllStringSubmatch(command, -1)

	return fp.Map[[]string, string](
		func(x []string) string {
			if strings.HasPrefix(x[0], "\"") && strings.HasSuffix(x[0], "\"") {
				return x[0][1 : len(x[0])-1]
			}
			return x[0]
		},
	)(matches)
}

func ProcessFile(roots RootFlags, processor *Processor, file string) error {
	log.Printf("Processing %s", file)

	if !FileExists(file) {
		return errors.New(fmt.Sprintf("File %s did not exist", file))
	}

	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// Before interpreting the file, we need to look for non-deterministic behavior
	// Some .cfg files (notably single player maps) can change textures dynamically
	dynamic := false
	for _, line := range strings.Split(string(src), "\n") {
		args := ParseLine(line)

		if len(args) == 0 {
			continue
		}

		// Can't do conditionals
		if args[0] == "if" {
			dynamic = true
			break
		}

		for _, arg := range args {
			if strings.HasPrefix(arg, "(") || strings.HasSuffix(arg, ")") || strings.HasPrefix(arg, "[") || strings.HasSuffix(arg, "]") {
				dynamic = true
				break
			}
		}
	}

	// Don't parse this file, attempt to parse the shimmed version
	if dynamic {
		hash := fmt.Sprintf("%x", sha256.Sum256(src))
		shim := filepath.Join("shims/", hash)

		log.Printf("File %s contained dynamic code. Falling back to shim %s", file, shim)

		if !FileExists(shim) {
			log.Printf("Shim %s did not exist. Creating it and exiting.", shim)
			os.WriteFile(shim, []byte(src), 0666)
			os.Exit(1)
		}

		return ProcessFile(roots, processor, shim)
	}

	for _, line := range strings.Split(string(src), "\n") {
		args := ParseLine(line)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "texturereset":
			processor.ResetTextures()

		case "exec":
			if len(args) > 1 {
				execPath := args[1]

				resolved := SearchFile(roots, execPath)

				if resolved == nil {
					log.Printf("Could not find %s", execPath)
				} else {
					err := ProcessFile(roots, processor, *resolved)
					if err != nil {
						return err
					}
				}
			}

		case "include":
			if len(args) > 1 {
				processor.AddFile(args[1])
			}

		case "texture":
			if len(args) > 2 {
				flag := args[1]

				// Other shader inputs don't get slots
				if flag == "0" {
					processor.AddTexture(args[2])
				} else {
					processor.AddFile(args[2])
				}
			}
		}
	}

	return nil
}

func main() {
	var roots RootFlags

	flag.Var(&roots, "root", "Specify an explicit asset root directory. Roots are searched in order of appearance.")
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		log.Fatal("You must provide only a single argument.")
	}

	filename := args[0]
	extension := filepath.Ext(filename)

	if extension != ".ogz" && extension != ".cgz" {
		log.Fatal("Map must end in .ogz or .cgz")
	}

	_map, err := maps.LoadMap(filename)

	if err != nil {
		log.Fatal("Failed to parse map file")
	}

	textureRefs := GetChildTextures(_map.Cubes)

	// Always load the default map settings
	defaultPath := SearchFile(roots, "data/default_map_settings.cfg")

	if defaultPath == nil {
		log.Fatal("Root with data/default_map_settings.cfg not provided")
	}

	processor := NewProcessor()
	err = ProcessFile(roots, processor, *defaultPath)
	if err != nil {
		log.Fatal(err)
	}

	baseName := filepath.Base(filename)
	cfgName := fmt.Sprintf("%s.cfg", filepath.Join(
		filepath.Dir(filename),
		baseName[:len(baseName)-len(extension)],
	))
	if FileExists(cfgName) {
		err = ProcessFile(roots, processor, cfgName)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Refs: %d", len(textureRefs))
	for _, texture := range processor.Textures {
		if refs, ok := textureRefs[uint16(texture.Slot)]; ok {
			log.Printf("%d: %s (%d)", texture.Slot, texture.Path, refs)
		}
	}
	//for k, v := range textureRefs {
	//}
}
