package min

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/cs"
	"github.com/cfoust/sour/pkg/game/io"
	"github.com/cfoust/sour/pkg/maps"
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

func (r *Reference) Resolve(ctx context.Context) (string, error) {
	if r.Root == nil {
		return r.Path, nil
	}
	return r.Root.Reference(ctx, r.Path)
}

func (r *Reference) ReadFile(ctx context.Context) ([]byte, error) {
	if r.Root == nil {
		return os.ReadFile(r.Path)
	}

	return r.Root.ReadFile(ctx, r.Path)
}

func (r *Reference) Exists(ctx context.Context) bool {
	if r.Root == nil {
		return assets.FileExists(r.Path)
	}

	return r.Root.Exists(ctx, r.Path)
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
	Name string
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

func (t TextureIndex) Marshal(p *io.Packet) error {
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

func (t *TextureIndex) Unmarshal(p *io.Packet) error {
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

	// The reference to the file we're currently processing
	current         *Reference
	modelName       string
	modelDir        string
	processingModel bool
	ModelFiles      []*Reference

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
func (processor *Processor) SearchFile(ctx context.Context, path string) *Reference {
	for _, root := range processor.Roots {
		unprefixed := NewReference(root, path)
		prefixed := NewReference(root, filepath.Join("packages", path))

		if unprefixed.Exists(ctx) {
			return unprefixed
		}

		if prefixed.Exists(ctx) {
			return prefixed
		}

		// Look relative to the current path if we're processing a file
		if processor.current != nil {
			pwdRef := NewReference(
				root,
				filepath.Clean(filepath.Join(
					filepath.Dir(processor.current.Path),
					path,
				)),
			)

			if pwdRef.Exists(ctx) {
				return pwdRef
			}

			// Also check the parent dir (Cube does this, too)
			pwdRef = NewReference(
				root,
				filepath.Clean(filepath.Join(
					filepath.Dir(processor.current.Path),
					"..",
					path,
				)),
			)

			if pwdRef.Exists(ctx) {
				return pwdRef
			}
		}

		// Finally, look in the current modelDir and its parent Models
		// do not always have a "current" file so the above search will
		// not catch these
		modelDir := filepath.Join(
			"packages",
			"models",
			processor.modelDir,
		)

		modelRef := NewReference(
			root,
			filepath.Join(
				modelDir,
				path,
			),
		)

		if modelRef.Exists(ctx) {
			return modelRef
		}

		modelRef = NewReference(
			root,
			filepath.Clean(filepath.Join(
				modelDir,
				"..",
				path,
			)),
		)

		if modelRef.Exists(ctx) {
			return modelRef
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

func (processor *Processor) ProcessFile(ctx context.Context, ref *Reference) error {
	if !ref.Exists(ctx) {
		return errors.New(fmt.Sprintf("File %s did not exist", ref))
	}

	previous := processor.current
	processor.current = ref

	src, err := ref.ReadFile(ctx)
	if err != nil {
		return err
	}

	processor.cfgVM.Run(string(src))
	processor.current = previous
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
