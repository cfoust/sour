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
	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/maps"
)

func Find[T any](handler func(x T) bool) func(list []T) opt.Option[T] {
	return func(list []T) opt.Option[T] {
		for _, item := range list {
			if handler(item) {
				return opt.Some[T](item)
			}
		}

		return opt.None[T]()
	}
}

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
func SearchFile(roots []string, path string) opt.Option[string] {
	for i := 0; i < len(roots); i++ {
		needle := filepath.Join(roots[i], path)
		if FileExists(needle) {
			return opt.Some[string](needle)
		}
	}

	return opt.None[string]()
}

var (
	// All of the valid material slots
	MATERIALS = []string{
		"air",
		"water",
		"water1",
		"water2",
		"water3",
		"water4",
		"glass",
		"glass1",
		"glass2",
		"glass3",
		"glass4",
		"lava",
		"lava1",
		"lava2",
		"lava3",
		"lava4",
		"clip",
		"noclip",
		"gameclip",
		"death",
		"alpha",

		// This does not exist in the Sauer code but simplifies our
		// logic a bit.
		"sky",
	}

	CUBEMAPSIDES = []string{
		"lf",
		"rt",
		"ft",
		"bk",
		"dn",
		"up",
	}

	// The valid parameters to texture slots
	PARAMS = []string{
		"c",
		"u",
		"d",
		"n",
		"g",
		"s",
		"z",
		"a",
		"e",

		// This only seems to appear after materials, so is this
		// actually a param?
		"1",
	}
)

type Texture struct {
	Paths     []string
	Autograss opt.Option[string]
}

func NewTexture() *Texture {
	texture := Texture{}
	texture.Paths = make([]string, 0)
	return &texture
}

type Processor struct {
	Roots   RootFlags
	Current *Texture
	// Cube faces reference slots inside of this
	Slots     []Texture
	Materials map[string]*Texture
	// File references are guaranteed to be included and do not have a slot
	Files []string
}

func NewProcessor(roots RootFlags) *Processor {
	processor := Processor{}

	processor.Roots = roots
	processor.Slots = make([]Texture, 0)
	processor.Materials = make(map[string]*Texture)

	for _, material := range MATERIALS {
		processor.Materials[material] = NewTexture()
	}

	processor.Files = make([]string, 0)

	return &processor
}

func (processor *Processor) NewSlot() {
	texture := NewTexture()
	processor.Slots = append(processor.Slots, *texture)
	processor.Current = &processor.Slots[len(processor.Slots)-1]
}

func (processor *Processor) SetMaterial(material string) {
	texture := NewTexture()
	processor.Materials[material] = texture
	processor.Current = texture
}

func (processor *Processor) AddTexture(path string) {
	processor.Current.Paths = append(processor.Current.Paths, path)
}

func (processor *Processor) ResetTextures() {
	processor.Slots = make([]Texture, 0)
}

func (processor *Processor) ResetMaterials() {
	for _, material := range MATERIALS {
		if material == "sky" {
			continue
		}
		processor.Materials[material] = NewTexture()
	}
}

func (processor *Processor) AddFile(path string) {
	processor.Files = append(processor.Files, path)
}

var (
	COMMAND_REGEX = regexp.MustCompile(`(("[^"]*")|([^\s]+))`)

	// Textures can have some additional stuff to modify them but they
	// should refer to the same file
	// ex: <mix:1,1,1><mad:2/2/2>
	TEXTURE_REGEX = regexp.MustCompile(`((<[^>]*>)*)([^<]+)`)
)

func NormalizeTexture(texture string) string {
	matches := TEXTURE_REGEX.FindStringSubmatch(texture)
	return matches[3]
}

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

func (processor *Processor) ProcessFile(file string) error {
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

		return processor.ProcessFile(shim)
	}

	for _, line := range strings.Split(string(src), "\n") {
		args := ParseLine(line)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "texturereset":
			processor.ResetTextures()

		case "materialreset":
			processor.ResetMaterials()

		case "autograss":
			if len(args) < 2 {
				break
			}

			processor.Slots[len(processor.Slots)-1].Autograss = opt.Some[string](
				NormalizeTexture(args[1]),
			)

		case "loadsky":
			if len(args) < 2 {
				break
			}

			oldCurrent := processor.Current

			processor.SetMaterial("sky")

			prefix := filepath.Join("packages", NormalizeTexture(args[1]))
			wildcard := strings.Index(prefix, "*")
			for _, side := range CUBEMAPSIDES {
				if wildcard != -1 {
					path := fmt.Sprintf(
						"%s%s%s",
						prefix[:wildcard],
						side,
						prefix[wildcard+1:],
					)

					processor.AddTexture(path)
					continue
				}

				// Otherwise normal
				jpgPath := fmt.Sprintf(
					"%s_%s.jpg",
					prefix,
					side,
				)

				resolvedJpg := SearchFile(processor.Roots, jpgPath)
				if opt.IsSome(resolvedJpg) {
					processor.AddTexture(jpgPath)
					continue
				}

				pngPath := fmt.Sprintf(
					"%s_%s.png",
					prefix,
					side,
				)

				resolvedPng := SearchFile(processor.Roots, pngPath)
				if opt.IsSome(resolvedPng) {
					processor.AddTexture(pngPath)
					continue
				}

				log.Printf("No texture for skybox %s side %s (%s %s)", prefix, side, jpgPath, pngPath)
			}

			processor.Current = oldCurrent

		case "exec":
			if len(args) != 2 {
				break
			}
			execPath := args[1]

			resolved := SearchFile(processor.Roots, execPath)

			if opt.IsNone(resolved) {
				log.Printf("Could not find %s", execPath)
			} else {
				err := processor.ProcessFile(resolved.Value)
				if err != nil {
					return err
				}
			}

		case "include":
			if len(args) != 2 {
				break
			}

			processor.AddFile(args[1])

		case "texture":
			if len(args) < 3 {
				break
			}

			flag := args[1]

			material := Find[string](func(x string) bool {
				return flag == x
			})(MATERIALS)

			param := Find[string](func(x string) bool {
				return flag == x
			})(PARAMS)

			if flag == "0" {
				// "0" always means a new texture slot
				processor.NewSlot()
			} else if opt.IsSome(material) {
				processor.SetMaterial(material.Value)
			} else if opt.IsNone(param) {
				// At this point it is not 0, not a material,
				// and not in the list of params, so that can
				// only mean that it is wrong somehow
				log.Printf("Invalid param: %s", line)
				break
			}

			processor.AddTexture(NormalizeTexture(args[2]))

		case "alias":
		case "blurskylight":
		case "fog":
		case "fogcolour":
		case "setshader":
		case "setshaderparam":
		case "skytexture":
		case "texcolor":
		case "texlayer":
		case "texscale":
		case "texscroll":
		case "waterfog":
			break

		default:
			log.Printf("Unhandled command: %s", args[0])
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

	if opt.IsNone(defaultPath) {
		log.Fatal("Root with data/default_map_settings.cfg not provided")
	}

	processor := NewProcessor(roots)
	err = processor.ProcessFile(defaultPath.Value)
	if err != nil {
		log.Fatal(err)
	}

	baseName := filepath.Base(filename)
	cfgName := fmt.Sprintf("%s.cfg", filepath.Join(
		filepath.Dir(filename),
		baseName[:len(baseName)-len(extension)],
	))
	if FileExists(cfgName) {
		err = processor.ProcessFile(cfgName)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Slots: %d", len(processor.Slots))
	log.Printf("Refs: %d", len(textureRefs))
	for i, texture := range processor.Slots {
		if refs, ok := textureRefs[uint16(i)]; ok {
			for _, path := range texture.Paths {
				log.Printf("%d: %s (%d)", i, path, refs)
			}
		}

		if opt.IsSome(texture.Autograss) {
			log.Printf("%d: grass %s", i, texture.Autograss.Value)
		}
	}

	for material, texture := range processor.Materials {
		log.Printf("%s: %d", material, len(texture.Paths))
		for _, path := range texture.Paths {
			log.Printf("%s: %s", material, path)
		}
	}
}
