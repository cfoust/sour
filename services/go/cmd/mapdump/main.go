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
	"strconv"
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

func CountTextures(cube maps.Cube, target map[int32]int) {
	if cube.Children != nil {
		CountChildTextures(*cube.Children, target)
		return
	}

	for i := 0; i < 6; i++ {
		texture := int32(cube.Texture[i])
		target[texture] = target[texture] + 1
	}
}

func CountChildTextures(cubes []maps.Cube, target map[int32]int) {
	for i := 0; i < 8; i++ {
		CountTextures(cubes[i], target)
	}
}

func GetChildTextures(cubes []maps.Cube, vslots []*VSlot) map[int32]int {
	vSlotRefs := make(map[int32]int)
	CountChildTextures(cubes, vSlotRefs)

	// Each VSlot can refer to two Slots:
	// * VSlot.Slot
	// * VSlot.Layer -> VSlot.Slot
	slotRefs := make(map[int32]int)
	for index, _ := range vSlotRefs {
		if index >= int32(len(vslots)) {
			continue
		}

		vslot := vslots[index]
		if vslot.Slot == nil {
			continue
		}

		slotRefs[vslot.Slot.Index]++

		layer := vslot.Layer
		if layer == 0 {
			continue
		}

		layerSlot := vslots[layer]
		if layerSlot.Slot == nil {
			continue
		}

		slotRefs[layerSlot.Slot.Index]++
	}

	return slotRefs
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

func NewProcessor(roots RootFlags, slots maps.VSlotData) *Processor {
	processor := Processor{}

	processor.Roots = roots

	vslots := fp.Map[*maps.VSlot, *VSlot](func(old *maps.VSlot) *VSlot {
		vslot := NewVSlot(nil, old.Index)
		vslot.Changed = old.Changed
		vslot.Layer = old.Layer
		return vslot
	})(slots.Slots)

	// Relink linked list
	for i, vslot := range vslots {
		prev := slots.Previous[i]
		if prev >= 0 && prev < int32(len(vslots)) {
			vslots[prev].Next = vslot
		}
	}

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
			log.Fatal(err)
		}

		if strings.Contains(relative, "..") {
			continue
		}

		return opt.Some[string](relative)
	}

	return opt.None[string]()
}

func (processor *Processor) AddSlot() *Slot {
	newSlot := NewSlot()
	newSlot.Index = int32(len(processor.Slots))
	processor.Slots = append(processor.Slots, newSlot)
	return newSlot
}

func (processor *Processor) ReassignVSlot(owner *Slot, vslot *VSlot) *VSlot {
	current := vslot
	owner.Variants = current

	for current != nil {
		current.Slot = owner
		current.Linked = false
		current = current.Next
	}

	return vslot
}

func (processor *Processor) EmptyVSlot(owner *Slot) *VSlot {
	var offset int32 = 0

	for i := len(processor.Slots) - 1; i >= 0; i-- {
		variants := processor.Slots[i].Variants
		if variants != nil {
			offset = variants.Index + 1
			break
		}
	}

	for i := offset; i < int32(len(processor.VSlots)); i++ {
		if processor.VSlots[i].Changed == 0 {
			return processor.ReassignVSlot(owner, processor.VSlots[i])
		}
	}

	vslot := NewVSlot(owner, int32(len(processor.VSlots)))
	processor.VSlots = append(processor.VSlots, vslot)
	return processor.VSlots[len(processor.VSlots)-1]
}

func (processor *Processor) ListVSlots() {
	for i, vslot := range processor.VSlots {
		fmt.Printf("vslot %d changed=%d layer=%d\n", i, vslot.Changed, vslot.Layer)
		if vslot.Slot != nil {
			for _, sts := range vslot.Slot.Sts {
				fmt.Printf("%d: %s\n", i, sts.Name)
			}
		} else {
			fmt.Printf("%d: null\n", i)
		}
	}
}

func (processor *Processor) Texture(textureType string, name string) {
	texture := Find[string](func(x string) bool {
		return textureType == x
	})(PARAMS)

	material := Find[string](func(x string) bool {
		return textureType == x
	})(MATERIALS)

	isDiffuse := texture.Value == "c" || textureType == "0"

	var slot *Slot
	if isDiffuse {
		processor.LastMaterial = nil
	} else if processor.LastMaterial != nil {
		slot = processor.LastMaterial
	}

	if slot == nil {
		if opt.IsSome(material) {
			slot = processor.Materials[material.Value]
			processor.LastMaterial = slot
		} else {
			if isDiffuse {
				processor.AddSlot()
			}

			slot = processor.Slots[len(processor.Slots)-1]
		}
	}

	slot.Loaded = false

	slot.AddSts(name)

	if isDiffuse && opt.IsNone(material) {
		vslot := processor.EmptyVSlot(slot)
		var changed int32 = (1 << maps.VSLOT_NUM) - 1

		// propagatevslot
		next := vslot.Next
		for next != nil {
			diff := changed & ^next.Changed
			if diff != 0 {
				if (diff & (1 << maps.VSLOT_LAYER)) != 0 {
					next.Layer = vslot.Layer
				}
			}
			next = next.Next
		}
	}
}

func (processor *Processor) SetMaterial(material string) {
	texture := NewSlot()
	processor.Materials[material] = texture
	processor.LastMaterial = texture
}

var dummySlot = Slot{}

func (processor *Processor) ResetTextures(n int32) {
	limit := n
	max := int32(len(processor.Slots))
	if n < 0 {
		n = 0
	}
	if n > max {
		n = max
	}

	for i := limit; i < max; i++ {
		slot := processor.Slots[i]
		for vs := slot.Variants; vs != nil; vs = vs.Next {
			vs.Slot = &dummySlot
		}
	}

	processor.Slots = processor.Slots[:limit]

	for len(processor.VSlots) > 0 {
		vslot := processor.VSlots[len(processor.VSlots)-1]
		if vslot.Slot != &dummySlot || vslot.Changed != 0 {
			break
		}
		processor.VSlots = processor.VSlots[:len(processor.VSlots)-1]
	}
}

func (processor *Processor) ResetSounds() {
	processor.Sounds = make([]string, 0)
}

func (processor *Processor) AddSound(path string) {
	processor.Sounds = append(processor.Sounds, path)
}

func (processor *Processor) ResetModels() {
	processor.Models = make([]Model, 0)
}

func (processor *Processor) AddModel(textures []string) {
	model := Model{}
	model.Paths = textures
	processor.Models = append(processor.Models, model)
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

	// Textures can have some additional stuff to modify them but they
	// should refer to the same file
	// ex: <mix:1,1,1><mad:2/2/2>
	TEXTURE_REGEX = regexp.MustCompile(`((<[^>]*>)*)([^<]+)`)

	TEXTURE_COMMAND_REGEX = regexp.MustCompile(`<([^>]*)>`)
)

func NormalizeTexture(texture string) string {
	matches := TEXTURE_REGEX.FindStringSubmatch(texture)
	if len(matches) == 0 {
		return ""
	}
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
			return strings.ReplaceAll(x[0], "\"", "")
		},
	)(matches)
}

func (processor *Processor) FindTexture(texture string) opt.Option[string] {
	for _, extension := range []string{"png", "jpg"} {
		resolved := processor.SearchFile(
			filepath.Join("packages", fmt.Sprintf("%s.%s", texture, extension)),
		)

		if opt.IsSome(resolved) {
			return resolved
		}
	}

	withoutExtension := processor.SearchFile(
		filepath.Join("packages", texture),
	)

	if opt.IsSome(withoutExtension) {
		return withoutExtension
	}

	return opt.None[string]()
}

func (processor *Processor) FindCubemap(cubemap string) []string {
	prefix := filepath.Join("packages", cubemap)
	wildcard := strings.Index(prefix, "*")

	textures := make([]string, 0)

	for _, side := range CUBEMAPSIDES {
		if wildcard != -1 {
			path := fmt.Sprintf(
				"%s%s%s",
				prefix[:wildcard],
				side,
				prefix[wildcard+1:],
			)

			textures = append(textures, path)
			continue
		}

		// Otherwise normal
		jpgPath := fmt.Sprintf(
			"%s_%s.jpg",
			prefix,
			side,
		)

		resolvedJpg := processor.SearchFile(jpgPath)
		if opt.IsSome(resolvedJpg) {
			textures = append(textures, jpgPath)
			continue
		}

		pngPath := fmt.Sprintf(
			"%s_%s.png",
			prefix,
			side,
		)

		resolvedPng := processor.SearchFile(pngPath)
		if opt.IsSome(resolvedPng) {
			textures = append(textures, pngPath)
			continue
		}

		log.Printf("No texture for skybox %s side %s (%s %s)", prefix, side, jpgPath, pngPath)
	}

	return textures
}

func (processor *Processor) ProcessModel(path string) (opt.Option[[]string], error) {
	results := make([]string, 0)

	modelDir := filepath.Join(
		"packages/models",
		path,
	)

	// This is slightly different from the other Normalize because models
	// specifically use relative paths for some stuff
	normalizePath := func(path string) string {
		return filepath.Clean(filepath.Join(modelDir, path))
	}

	resolveRelative := func(file string) opt.Option[string] {
		path := normalizePath(file)
		resolved := processor.SearchFile(path)

		if opt.IsSome(resolved) {
			return resolved
		}

		// Also check the parent dir (Cube does this, too)
		parent := filepath.Join(
			filepath.Dir(path),
			"..",
			filepath.Base(path),
		)
		return processor.SearchFile(parent)
	}

	addRootFile := func(file string) {
		resolved := processor.SearchFile(file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find root-relative model path %s", file)
			return
		}

		results = append(results, resolved.Value)
	}

	// Some references are relative to the model config
	addRelative := func(file string) {
		resolved := resolveRelative(file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find cfg-relative model path %s (%s)", file, path)
			return
		}

		results = append(results, resolved.Value)
	}

	// Model textures tend to also come with a DDS counterpart
	expandTexture := func(texture string) []string {
		normalized := NormalizeTexture(texture)

		hasDDS := fp.Some(
			func(x []string) bool {
				return x[1] == "dds"
			},
		)(TEXTURE_COMMAND_REGEX.FindAllStringSubmatch(texture, -1))

		if hasDDS {
			extension := filepath.Ext(normalized)
			ddsPath := fmt.Sprintf(
				"%s.dds",
				normalized[:len(normalized)-len(extension)],
			)
			return []string{normalized, ddsPath}
		}

		return []string{normalized}
	}

	addTexture := func(texture string) {
		for _, file := range expandTexture(texture) {
			addRelative(file)
		}
	}

	_type := Find[string](func(x string) bool {
		// First look for the cfg
		cfg := fmt.Sprintf(
			"%s/%s.cfg",
			modelDir,
			x,
		)

		resolved := processor.SearchFile(cfg)

		if opt.IsSome(resolved) {
			return true
		}

		// Then tris, since that is also there
		tris := fmt.Sprintf(
			"%s/tris.%s",
			modelDir,
			x,
		)

		resolved = processor.SearchFile(tris)

		if opt.IsSome(resolved) {
			return true
		}

		return false
	})(MODELTYPES)

	if opt.IsNone(_type) {
		return opt.None[[]string](), errors.New(fmt.Sprintf("Failed to infer type for model %s", path))
	}

	modelType := _type.Value

	defaultFiles := []string{
		fmt.Sprintf("tris.%s", modelType),
		"skin.png",
		"skin.jpg",
		"mask.png",
		"mask.jpg",
	}

	hadDefault := false
	for _, _default := range defaultFiles {
		resolved := resolveRelative(_default)

		if opt.IsNone(resolved) {
			continue
		}

		hadDefault = true
		addRelative(_default)
	}

	cfgPath := fmt.Sprintf(
		"%s/%s.cfg",
		modelDir,
		modelType,
	)

	resolved := processor.SearchFile(cfgPath)

	if opt.IsNone(resolved) {
		if !hadDefault {
			return opt.None[[]string](), errors.New(fmt.Sprintf("Model %s had neither defaults nor a .cfg", path))
		}

		return opt.Some[[]string](results), nil
	}

	addRootFile(cfgPath)

	src, err := os.ReadFile(resolved.Value)
	if err != nil {
		return opt.None[[]string](), errors.New(fmt.Sprintf("Failed to read %s", resolved.Value))
	}

	for _, line := range strings.Split(string(src), "\n") {
		args := ParseLine(line)

		if len(args) == 0 {
			continue
		}

		command := args[0]

		if strings.HasPrefix(command, modelType) {
			command = command[len(modelType):]
		}

		switch command {
		case "anim":
			if len(args) < 3 {
				break
			}

			// `anim` uses anim indices and files, so no need to
			// error if it's not found
			for i := 2; i < len(args); i++ {
				resolved := resolveRelative(args[i])
				if opt.IsNone(resolved) {
					continue
				}

				addTexture(args[i])
			}

		case "bumpmap":
			if len(args) < 3 {
				break
			}

			addTexture(args[2])

		case "load":
			if len(args) < 2 {
				break
			}

			addTexture(args[1])

		case "skin":
			if len(args) < 3 {
				break
			}

			for i := 2; i < 4; i++ {
				if i == len(args) {
					break
				}

				addTexture(args[i])
			}

		case "mdlenvmap":
			if len(args) != 4 {
				break
			}

			for _, texture := range processor.FindCubemap(NormalizeTexture(args[3])) {
				addRootFile(texture)
			}

		case "basemodelcfg": // TODO dynamic code in models?
		case "ambient":
		case "cullface":
		case "dir":
		case "mdlalphablend":
		case "mdlalphadepth":
		case "mdlalphatest":
		case "mdlambient":
		case "mdlbb":
		case "mdlcollide":
		case "mdlcullface":
		case "mdldepthoffset":
		case "mdlellipsecollide":
		case "mdlextendbb":
		case "mdlfullbright":
		case "mdlglare":
		case "mdlglow":
		case "mdlpitch":
		case "mdlscale":
		case "mdlshader":
		case "mdlshadow":
		case "mdlspec":
		case "mdlspin":
		case "mdltrans":
		case "mdlyaw":
		case "noclip":
		case "pitch":
		case "scroll":
		case "spec":
			break

		default:
			log.Printf("Unhandled modelcommand: %s", command)
		}
	}

	return opt.Some[[]string](results), nil
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
			log.Fatalf("Shim %s contained dynamic code; please fill it in", file)
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

			textures, err := processor.ProcessModel(args[len(args)-1])

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

type Reference struct {
	// The absolute path of the asset on the filesystem
	// Example: /home/blah/sauerbraten/packages/base/blah.cfg
	Absolute string

	// The path of the asset relative to the game's "root"
	// Example: packages/base/blah.cfg
	Relative string
}

func main() {
	var roots RootFlags

	flag.Var(&roots, "root", "Specify an explicit asset root directory. Roots are searched in order of appearance.")
	flag.Parse()

	absoluteRoots := fp.Map[string, string](func(root string) string {
		absolute, err := filepath.Abs(root)
		if err != nil {
			log.Fatal(err)
		}
		return absolute
	})(roots)

	args := flag.Args()

	if len(args) != 1 {
		log.Fatal("You must provide only a single argument.")
	}

	filename, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatal(err)
	}

	extension := filepath.Ext(filename)

	if extension != ".ogz" {
		log.Fatal("Map must end in .ogz")
	}

	_map, err := maps.LoadMap(filename)

	if err != nil {
		log.Fatal("Failed to parse map file")
	}

	processor := NewProcessor(absoluteRoots, _map.VSlots)

	references := make([]Reference, 0)

	// File paths are strange in Sauer: certain types of assets omit the
	// packages/, others are relative to the config file (models), and this
	// program also accepts map files not inside of a Sauer directory
	// structure. On top of that, we ultimately need to map assets into the
	// game's filesystem correctly. This function normalizes all paths so
	// we can do that more easily.
	var addFile func(file string)
	addFile = func(file string) {
		reference := Reference{}

		if filepath.IsAbs(file) {
			reference.Absolute = file

			relative := processor.GetRootRelative(file)

			if opt.IsNone(relative) {
				log.Fatal(fmt.Sprintf("File absolute but not in root: %s", file))
			}

			reference.Relative = relative.Value
			references = append(references, reference)
			return
		}

		// This might just be a file (like a config) that was specified with a relative path
		absolute, err := filepath.Abs(file)
		if err != nil {
			log.Fatal(err)
		}

		if FileExists(absolute) {
			addFile(absolute)
			return
		}

		// If it's relative, it must be inside of a root
		resolved := processor.SearchFile(file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find relative file in roots: %s", file)
			return
		}

		// Sometimes a file was specified without packages/ so we need
		// to normalize it
		addFile(resolved.Value)
	}

	// Map files can be mapped into packages/base/
	addMapFile := func(file string) {
		target := file

		if filepath.IsAbs(file) {
			absolute, err := filepath.Abs(file)

			if err != nil {
				log.Fatal(err)
			}
			target = absolute
		}

		if !FileExists(target) {
			return
		}

		relative := processor.GetRootRelative(target)

		if opt.IsSome(relative) {
			addFile(relative.Value)
			return
		}

		reference := Reference{}
		reference.Absolute = target
		reference.Relative = fmt.Sprintf("packages/base/%s", filepath.Base(file))
		references = append(references, reference)
	}

	addMapFile(filename)

	// Some variables contain textures
	if skybox, ok := _map.SVars["skybox"]; ok {
		for _, path := range processor.FindCubemap(NormalizeTexture(skybox)) {
			addFile(path)
		}
	}

	if cloudlayer, ok := _map.SVars["cloudlayer"]; ok {
		resolved := processor.FindTexture(NormalizeTexture(cloudlayer))

		if opt.IsSome(resolved) {
			addFile(resolved.Value)
		}
	}

	modelRefs := make(map[int16]int)
	for _, entity := range _map.Entities {
		if entity.Type != maps.ET_MAPMODEL {
			continue
		}

		modelRefs[entity.Attr2] += 1
	}

	// Always load the default map settings
	defaultPath := processor.SearchFile("data/default_map_settings.cfg")

	if opt.IsNone(defaultPath) {
		log.Fatal("Root with data/default_map_settings.cfg not provided")
	}

	err = processor.ProcessFile(defaultPath.Value)
	if err != nil {
		log.Fatal(err)
	}

	cfgName := ReplaceExtension(filename, "cfg")
	if FileExists(cfgName) {
		err = processor.ProcessFile(cfgName)
		if err != nil {
			log.Fatal(err)
		}

		addMapFile(cfgName)
	}

	for _, extension := range []string{"png", "jpg"} {
		shotName := ReplaceExtension(filename, extension)
		addMapFile(shotName)
	}

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			addFile(path.Name)
		}
	}

	for _, sound := range processor.Sounds {
		addFile(sound)
	}

	for i, model := range processor.Models {
		if _, ok := modelRefs[int16(i)]; ok {
			for _, path := range model.Paths {
				addFile(path)
			}
		}
	}

	textureRefs := GetChildTextures(_map.Cubes, processor.VSlots)

	for i, slot := range processor.Slots {
		if _, ok := textureRefs[int32(i)]; ok {
			for _, path := range slot.Sts {
				addFile(path.Name)
			}
		}
	}

	for _, path := range references {
		fmt.Printf("%s->%s\n", path.Absolute, path.Relative)
	}
}
