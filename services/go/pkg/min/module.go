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

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/cs"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"

	"github.com/rs/zerolog/log"
)

// A reference to a file on the FS or in a root.
type Reference struct {
	Path string
	Root assets.Root
}

func NewReference(root assets.Root, path string) *Reference {
	return &Reference{
		Path: path,
		Root: root,
	}
}

func (r *Reference) String() string {
	if r.Root == nil {
		return fmt.Sprintf("%s", r.Path)
	}
	return fmt.Sprintf("%s root=!nil", r.Path)
}

func (r *Reference) Resolve() (string, error) {
	if r.Root == nil {
		return r.Path, nil
	}
	return r.Root.Reference(r.Path)
}

func (r *Reference) ReadFile() ([]byte, error) {
	if r.Root == nil {
		return os.ReadFile(r.Path)
	}

	return r.Root.ReadFile(r.Path)
}

func (r *Reference) Exists() bool {
	if r.Root == nil {
		return assets.FileExists(r.Path)
	}

	return r.Root.Exists(r.Path)
}

type Mapping struct {
	// The absolute path of the asset on the filesystem
	// Example: /home/blah/sauerbraten/packages/base/blah.cfg
	From *Reference

	// The path of the asset relative to the game's "root"
	// Example: packages/base/blah.cfg
	To string
}

func Find[T any](handler func(x T) bool) func(list []T) opt.Option[T] {
	return func(list []T) opt.Option[T] {
		for _, item := range list {
			if handler(item) {
				return opt.Some(item)
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

func ReplaceExtension(ref *Reference, newExtension string) *Reference {
	baseName := filepath.Base(ref.Path)
	extension := filepath.Ext(ref.Path)
	return &Reference{
		Path: fmt.Sprintf("%s.%s", filepath.Join(
			filepath.Dir(ref.Path),
			baseName[:len(baseName)-len(extension)],
		), newExtension),
		Root: ref.Root,
	}
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

type Model struct {
	Paths []*Reference
}

const (
	TEXTURE_CHANGE_ADD = iota
	TEXTURE_CHANGE_RESET
)

type TextureChangeType byte

type TextureChange interface {
	Op() TextureChangeType
}

type Texture struct {
	Ref      *Reference
	Line     int
	Type     string
	Name     string
	Rotation int
	Xoffset  int
	Yoffset  int
	Scale    float32
}

func (t Texture) Op() TextureChangeType {
	return TEXTURE_CHANGE_ADD
}

type TextureReset struct {
	Limit int
}

func (t TextureReset) Op() TextureChangeType {
	return TEXTURE_CHANGE_RESET
}

type TextureIndex []TextureChange

func (t TextureIndex) Marshal(p *game.Packet) error {
	err := p.Put(len(t))
	if err != nil {
		return err
	}

	for _, change := range t {
		p.Put(change.Op())

		switch change.Op() {
		case TEXTURE_CHANGE_ADD:
			add := change.(Texture)
			err = p.Put(
				add.Type,
				add.Name,
				add.Rotation,
				add.Xoffset,
				add.Yoffset,
				add.Scale,
			)
		case TEXTURE_CHANGE_RESET:
			reset := change.(TextureReset)
			err = p.Put(reset)
		}
	}
	return nil
}

func (t *TextureIndex) Unmarshal(p *game.Packet) error {
	numChanges, ok := p.GetInt()
	if !ok {
		return fmt.Errorf("could not read number of textures")
	}

	for i := 0; i < int(numChanges); i++ {
		op, ok := p.GetByte()
		if !ok {
			return fmt.Errorf("could not read op")
		}

		var err error
		switch op {
		case TEXTURE_CHANGE_ADD:
			var add Texture
			err = p.Get(
				&add.Type,
				&add.Name,
				&add.Rotation,
				&add.Xoffset,
				&add.Yoffset,
				&add.Scale,
			)
			*t = append(*t, add)
		case TEXTURE_CHANGE_RESET:
			var reset TextureReset
			err = p.Get(&reset)
			*t = append(*t, reset)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type Processor struct {
	Roots        []assets.Root
	LastMaterial *maps.Slot
	VSlots       []*maps.VSlot
	Slots        []*maps.Slot

	Textures  TextureIndex
	Models    []Model
	Sounds    []*Reference
	Materials map[string]*maps.Slot
	// File references are guaranteed to be included and do not have a slot
	Files []*Reference

	cfgVM *cs.VM
}

func NewProcessor(roots []assets.Root, slots []*maps.VSlot) *Processor {
	processor := Processor{}

	processor.Roots = roots
	processor.VSlots = slots
	processor.Slots = make([]*maps.Slot, 0)
	processor.Models = make([]Model, 0)
	processor.Sounds = make([]*Reference, 0)
	processor.Materials = make(map[string]*maps.Slot)
	processor.Textures = make([]TextureChange, 0)

	for _, material := range MATERIALS {
		processor.Materials[material] = maps.NewSlot()
	}

	processor.Files = make([]*Reference, 0)

	vm := cs.NewVM()
	processor.cfgVM = vm
	processor.setupVM()

	return &processor
}

// Search for a file in the roots, one at a time
func (processor *Processor) SearchFile(path string) *Reference {
	for _, root := range processor.Roots {
		unprefixed := NewReference(root, path)
		prefixed := NewReference(root, filepath.Join("packages", path))

		if unprefixed.Exists() {
			return unprefixed
		}

		if prefixed.Exists() {
			return prefixed
		}
	}

	return nil
}

func (processor *Processor) ResetSounds() {
	processor.Sounds = make([]*Reference, 0)
}

func (processor *Processor) AddSound(ref *Reference) {
	processor.Sounds = append(processor.Sounds, ref)
}

func (processor *Processor) ResetMaterials() {
	for _, material := range MATERIALS {
		if material == "sky" {
			continue
		}
		processor.Materials[material] = maps.NewSlot()
	}
}

func (processor *Processor) AddFile(ref *Reference) {
	processor.Files = append(processor.Files, ref)
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

	return fp.Map(
		func(x []string) string {
			return strings.ReplaceAll(x[0], "\"", "")
		},
	)(matches)
}

func (processor *Processor) ProcessFile(ref *Reference) error {
	if !ref.Exists() {
		return errors.New(fmt.Sprintf("File %s did not exist", ref))
	}

	src, err := ref.ReadFile()
	if err != nil {
		return err
	}

	processor.cfgVM.Run(string(src))
	return nil

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

		if strings.HasPrefix(ref.Path, "shims/") {
			log.Fatal().Msgf("Shim %s contained dynamic code; please fill it in", ref.Path)
		}

		if !FileExists(shim) {
			log.Printf("Shim %s for %s did not exist. Creating it and exiting.", shim, ref.Path)
			os.WriteFile(shim, []byte(src), 0666)
			os.Exit(1)
		}

		return processor.ProcessFile(NewReference(nil, shim))
	}

	for _, line := range strings.Split(string(src), "\n") {
		args := ParseLine(line)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "texture":

		case "texturereset":
			limit := 0

			reset := TextureReset{}
			if len(args) == 2 {
				value, _ := strconv.Atoi(args[1])
				limit = value
				reset.Limit = value
			}

			processor.Textures = append(processor.Textures, reset)
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
				processor.AddModel(make([]*Reference, 0))
				continue
			}

			if textures != nil {
				processor.AddModel(textures)
			}

		case "autograss":
			if len(args) < 2 {
				break
			}

			texture := processor.SearchFile(NormalizeTexture(args[1]))

			if texture != nil {
				processor.AddFile(texture)
			}

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
				if resolved != nil {
					processor.AddSound(resolved)
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

			if resolved == nil {
				log.Printf("Could not find %s", execPath)
			} else {
				err := processor.ProcessFile(resolved)
				processor.AddFile(resolved)
				if err != nil {
					return err
				}
			}

		case "include":
			if len(args) != 2 {
				break
			}

			file := processor.SearchFile(args[1])

			if file != nil {
				processor.AddFile(file)
			}

		case "cloudlayer":
			if len(args) != 2 {
				break
			}

			texture := args[1]
			resolved := processor.FindTexture(texture)

			if resolved != nil {
				processor.AddFile(resolved)
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

// Ensure each source file only appears in the destination once.
func CrunchReferences(references []Mapping) []Mapping {
	unique := make(map[string]*Reference)

	for _, reference := range references {
		unique[reference.To] = reference.From
	}

	result := make([]Mapping, 0)
	for relative, absolute := range unique {
		result = append(result, Mapping{
			From: absolute,
			To:   relative,
		})
	}

	return result
}
