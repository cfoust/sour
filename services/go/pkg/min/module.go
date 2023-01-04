package min

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/maps"

	"github.com/rs/zerolog/log"
)

type Reference struct {
	// The absolute path of the asset on the filesystem
	// Example: /home/blah/sauerbraten/packages/base/blah.cfg
	Absolute string

	// The path of the asset relative to the game's "root"
	// Example: packages/base/blah.cfg
	Relative string
}

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

func ReplaceExtension(file string, newExtension string) string {
	baseName := filepath.Base(file)
	extension := filepath.Ext(file)
	return fmt.Sprintf("%s.%s", filepath.Join(
		filepath.Dir(file),
		baseName[:len(baseName)-len(extension)],
	), newExtension)
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

	MODELTYPES = []string{
		"md2",
		"md3",
		"md5",
		"obj",
		"smd",
		"iqm",
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

type Model struct {
	Paths []string
}

func NewTexture() *Texture {
	texture := Texture{}
	texture.Paths = make([]string, 0)
	return &texture
}

type TexSlot struct {
	Name string
}

type Slot struct {
	Index    int32
	Sts      []TexSlot
	Variants *VSlot
	Loaded   bool
}

func NewSlot() *Slot {
	newSlot := Slot{}
	newSlot.Sts = make([]TexSlot, 0)
	newSlot.Loaded = false
	return &newSlot
}

func (slot *Slot) AddSts(name string) *TexSlot {
	sts := TexSlot{}
	sts.Name = name
	slot.Sts = append(slot.Sts, sts)
	return &slot.Sts[len(slot.Sts)-1]
}

type VSlot struct {
	Slot *Slot
	Next *VSlot

	Index   int32
	Changed int32
	Layer   int32
	Linked  bool
}

func (vslot *VSlot) AddVariant(slot *Slot) {
	if slot.Variants == nil {
		slot.Variants = vslot
	} else {
		prev := slot.Variants
		for prev != nil {
			prev = prev.Next
		}
		prev.Next = vslot
	}
}

func NewVSlot(owner *Slot, index int32) *VSlot {
	vslot := VSlot{
		Index: index,
		Slot:  owner,
	}
	if owner != nil {
		vslot.AddVariant(owner)
	}
	return &vslot
}

type Processor struct {
	Roots        RootFlags
	LastMaterial *Slot

	VSlots []*VSlot
	Slots  []*Slot
	// Cube faces reference slots inside of this
	Textures  []Texture
	Models    []Model
	Sounds    []string
	Materials map[string]*Slot
	// File references are guaranteed to be included and do not have a slot
	Files []string
}

func NewProcessor(roots RootFlags, slots []*maps.VSlot) *Processor {
	processor := Processor{}

	processor.Roots = roots

	vslots := fp.Map[*maps.VSlot, *VSlot](func(old *maps.VSlot) *VSlot {
		vslot := NewVSlot(nil, old.Index)
		vslot.Changed = old.Changed
		vslot.Layer = old.Layer
		return vslot
	})(slots)

	processor.VSlots = vslots

	processor.Slots = make([]*Slot, 0)
	processor.Models = make([]Model, 0)
	processor.Sounds = make([]string, 0)
	processor.Materials = make(map[string]*Slot)

	for _, material := range MATERIALS {
		processor.Materials[material] = NewSlot()
	}

	processor.Files = make([]string, 0)

	return &processor
}

// Search for a file in the roots, one at a time
func (processor *Processor) SearchFile(path string) opt.Option[string] {
	for i := 0; i < len(processor.Roots); i++ {
		unprefixed := filepath.Join(processor.Roots[i], path)
		prefixed := filepath.Join(processor.Roots[i], "packages", path)

		if FileExists(unprefixed) {
			return opt.Some[string](unprefixed)
		}

		if FileExists(prefixed) {
			return opt.Some[string](prefixed)
		}
	}

	return opt.None[string]()
}

func (processor *Processor) GetRootRelative(path string) opt.Option[string] {
	for _, root := range processor.Roots {
		relative, err := filepath.Rel(root, path)

		if err != nil {
			log.Fatal().Err(err)
		}

		if strings.Contains(relative, "..") {
			continue
		}

		return opt.Some[string](relative)
	}

	return opt.None[string]()
}

func (processor *Processor) ResetSounds() {
	processor.Sounds = make([]string, 0)
}

func (processor *Processor) AddSound(path string) {
	processor.Sounds = append(processor.Sounds, path)
}

func (processor *Processor) ResetMaterials() {
	for _, material := range MATERIALS {
		if material == "sky" {
			continue
		}
		processor.Materials[material] = NewSlot()
	}
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
			return strings.ReplaceAll(x[0], "\"", "")
		},
	)(matches)
}

func (processor *Processor) ProcessFile(file string) error {
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

		if strings.HasPrefix(file, "shims/") {
			log.Fatal().Msgf("Shim %s contained dynamic code; please fill it in", file)
		}

		if !FileExists(shim) {
			log.Printf("Shim %s for %s did not exist. Creating it and exiting.", shim, file)
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
			limit := 0

			if len(args) == 2 {
				parsed, _ := strconv.Atoi(args[1])
				limit = parsed
			}

			processor.ResetTextures(int32(limit))

		case "materialreset":
			processor.ResetMaterials()

		case "mapmodelreset":
			processor.ResetModels()

		case "mapmodel":
		case "mmodel":
			if len(args) < 2 {
				break
			}

			modelFile := args[len(args)-1]
			textures, err := processor.ProcessModel(modelFile)

			if err != nil {
				log.Printf("Failed to process model %s", args[1])
				processor.AddModel(make([]string, 0))
				continue
			}

			if opt.IsSome(textures) {
				processor.AddModel(textures.Value)
			}

		case "autograss":
			if len(args) < 2 {
				break
			}

			processor.AddFile(NormalizeTexture(args[1]))

		case "mapsoundreset":
			processor.ResetSounds()

		case "registersound":
		case "mapsound":
			if len(args) < 2 {
				break
			}

			name := args[1]

			for _, _type := range []string{"", ".wav", ".ogg"} {
				path := fmt.Sprintf(
					"packages/sounds/%s%s",
					name,
					_type,
				)

				resolved := processor.SearchFile(path)
				if opt.IsSome(resolved) {
					processor.AddSound(path)
					break
				}
			}

		case "cloudbox": // <- should this actually be here?
		case "skybox":
		case "loadsky":
			if len(args) < 2 {
				break
			}

			for _, texture := range processor.FindCubemap(NormalizeTexture(args[1])) {
				processor.AddFile(texture)
			}

		case "exec":
			if len(args) != 2 {
				break
			}
			execPath := args[1]

			resolved := processor.SearchFile(execPath)

			if opt.IsNone(resolved) {
				log.Printf("Could not find %s", execPath)
			} else {
				err := processor.ProcessFile(resolved.Value)
				processor.AddFile(resolved.Value)
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

			processor.Texture(args[1], NormalizeTexture(args[2]))

		case "cloudlayer":
			if len(args) != 2 {
				break
			}

			texture := args[1]
			resolved := processor.FindTexture(texture)

			if opt.IsSome(resolved) {
				processor.AddFile(resolved.Value)
			}

		case "adaptivesample":
		case "alias":
		case "ambient":
		case "blurlms":
		case "blurskylight":
		case "causticmillis":
		case "causticscale":
		case "cloudalpha":
		case "cloudboxalpha":
		case "cloudboxcolour":
		case "cloudcolour":
		case "cloudfade":
		case "cloudheight":
		case "cloudscale":
		case "cloudscrollx":
		case "cloudscrolly":
		case "edgetolerance":
		case "elevcontag":
		case "fog":
		case "fogcolour":
		case "fogdomecap":
		case "fogdomeclip":
		case "fogdomeclouds":
		case "fogdomecolour":
		case "fogdomeheight":
		case "fogdomemax":
		case "fogdomemin":
		case "grassalpha":
		case "grasscolour":
		case "lightlod":
		case "lightprecision":
		case "lmshadows":
		case "mapmsg":
		case "maptitle":
		case "maxmerge":
		case "minimapclip":
		case "minimapcolour":
		case "minimapheight":
		case "panelset":
		case "setshader":
		case "setshaderparam":
		case "shadowmapambient":
		case "shadowmapangle":
		case "skill":
		case "skyboxcolour":
		case "skylight":
		case "skytexture":
		case "skytexturelight":
		case "smoothangle":
		case "spinclouds":
		case "spinsky":
		case "sunlight":
		case "sunlightpitch":
		case "sunlightscale":
		case "sunlightyaw":
		case "texalpha":
		case "texcolor":
		case "texlayer":
		case "texoffset":
		case "texrotate":
		case "texscale":
		case "texscroll":
		case "texsmooth":
		case "water2colour":
		case "water2fog":
		case "watercolour":
		case "waterfallcolour":
		case "waterfog":
		case "waterspec":
		case "yawsky":
			break

		default:
			log.Printf("Unhandled command: %s", args[0])
		}
	}

	return nil
}

func (p *Processor) NormalizeFile(file string) opt.Option[Reference] {
	reference := Reference{}

	if filepath.IsAbs(file) {
		reference.Absolute = file

		relative := p.GetRootRelative(file)

		if opt.IsNone(relative) {
			return opt.None[Reference]()
		}

		reference.Relative = relative.Value
		return opt.Some[Reference](reference)
	}

	// This might just be a file (like a config) that was specified with a relative path
	absolute, err := filepath.Abs(file)
	if err != nil {
		log.Fatal().Err(err)
	}

	if FileExists(absolute) {
		return p.NormalizeFile(absolute)
	}

	// If it's relative, it must be inside of a root
	resolved := p.SearchFile(file)

	if opt.IsNone(resolved) {
		log.Printf("Failed to find relative file in roots: %s", file)
		return opt.None[Reference]()
	}

	// Sometimes a file was specified without packages/ so we need
	// to normalize it
	return p.NormalizeFile(resolved.Value)
}

// Ensure each source file only appears in the destination once.
func (p *Processor) CrunchReferences(references []Reference) []Reference {
	unique := make(map[string]string)

	for _, reference := range references {
		unique[reference.Relative] = reference.Absolute
	}

	result := make([]Reference, 0)
	for relative, absolute := range unique {
		result = append(result, Reference{
			Absolute: absolute,
			Relative: relative,
		})
	}

	return result
}
