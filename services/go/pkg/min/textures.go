package min

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"os"

	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/game"

	"github.com/rs/zerolog/log"
)

func CountTextures(cube *maps.Cube, target map[int32]int) {
	if cube.Children != nil && len(cube.Children) > 0 {
		CountChildTextures(cube.Children, target)
		return
	}

	for i := 0; i < 6; i++ {
		texture := int32(cube.Texture[i])
		target[texture] = target[texture] + 1
	}
}

func CountChildTextures(cubes []*maps.Cube, target map[int32]int) {
	for i := 0; i < 8; i++ {
		CountTextures(cubes[i], target)
	}
}

func GetChildTextures(cubes []*maps.Cube, vslots []*maps.VSlot) map[int32]int {
	vSlotRefs := make(map[int32]int)
	CountChildTextures(cubes, vSlotRefs)

	// Each VSlot can refer to two Slots:
	// * VSlot.Slot
	// * VSlot.Layer -> VSlot.Slot
	slotRefs := make(map[int32]int)
	for index := range vSlotRefs {
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

func (processor *Processor) AddSlot() *maps.Slot {
	newSlot := maps.NewSlot()
	newSlot.Index = int32(len(processor.Slots))
	processor.Slots = append(processor.Slots, newSlot)
	return newSlot
}

func (processor *Processor) ReassignVSlot(owner *maps.Slot, vslot *maps.VSlot) *maps.VSlot {
	current := vslot
	owner.Variants = current

	for current != nil {
		current.Slot = owner
		current.Linked = false
		current = current.Next
	}

	return vslot
}

func (processor *Processor) EmptyVSlot(owner *maps.Slot) *maps.VSlot {
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

	vslot := maps.NewVSlot(owner, int32(len(processor.VSlots)))
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
	texture := Find(func(x string) bool {
		return textureType == x
	})(PARAMS)

	material := Find(func(x string) bool {
		return textureType == x
	})(MATERIALS)

	isDiffuse := texture.Value == "c" || textureType == "0"

	var slot *maps.Slot
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

		//log.Printf("%s -> %d", name, vslot.Index)

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

var (
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

func (processor *Processor) SetMaterial(material string) {
	texture := maps.NewSlot()
	processor.Materials[material] = texture
	processor.LastMaterial = texture
}

var dummySlot = maps.Slot{}

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

func (processor *Processor) SaveTextureIndex(path string) error {
	p := game.Packet{}
	err := p.Put(processor.Textures)
	if err != nil {
		return err
	}

	//index := TextureIndex{}
	//err = p.Get(&index)
	//if err != nil {
		//return err
	//}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(p)
	if err != nil {
		return err
	}

	return nil
}
