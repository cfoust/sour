package min

import (
	"github.com/cfoust/sour/pkg/maps"

	"github.com/repeale/fp-go/option"
)

func (processor *Processor) Texture(type_ string, name string, rot int, xOffset int, yOffset int, scale float32) {
	texture := Texture{
		Type: type_,
		Name: name,
		Rotation: rot,
		Xoffset: xOffset,
		Yoffset: yOffset,
		Scale: scale,
	}

	processor.Textures = append(processor.Textures, texture)

	textureType := Find(func(x string) bool {
		return type_ == x
	})(PARAMS)

	material := Find(func(x string) bool {
		return type_ == x
	})(MATERIALS)

	isDiffuse := textureType.Value == "c" || type_ == "0"

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

func (p *Processor) setupVM() {
	vm := p.cfgVM
	vm.AddCommand("texture", p.Texture)
}
