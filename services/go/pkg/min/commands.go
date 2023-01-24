package min

import (
	"fmt"

	"github.com/cfoust/sour/pkg/maps"

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

func (p *Processor) TextureReset(limit int) {
	reset := TextureReset{
		Limit: limit,
	}

	p.Textures = append(p.Textures, reset)
	p.ResetTextures(int32(limit))
}

func (p *Processor) Texture(type_ string, name string, rot int, xOffset int, yOffset int, scale float32) {
	texture := Texture{
		Type:     type_,
		Name:     name,
		Rotation: rot,
		Xoffset:  xOffset,
		Yoffset:  yOffset,
		Scale:    scale,
	}

	p.Textures = append(p.Textures, texture)

	textureType := Find(func(x string) bool {
		return type_ == x
	})(PARAMS)

	material := Find(func(x string) bool {
		return type_ == x
	})(MATERIALS)

	isDiffuse := textureType.Value == "c" || type_ == "0"

	var slot *maps.Slot
	if isDiffuse {
		p.LastMaterial = nil
	} else if p.LastMaterial != nil {
		slot = p.LastMaterial
	}

	if slot == nil {
		if opt.IsSome(material) {
			slot = p.Materials[material.Value]
			p.LastMaterial = slot
		} else {
			if isDiffuse {
				p.AddSlot()
			}

			slot = p.Slots[len(p.Slots)-1]
		}
	}

	slot.Loaded = false

	slot.AddSts(name)

	if isDiffuse && opt.IsNone(material) {
		vslot := p.EmptyVSlot(slot)
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

func (p *Processor) MModel(name string) {
	modelFile := name
	textures, err := p.ProcessModel(modelFile)

	if err != nil {
		log.Printf("Failed to process model %s", name)
		p.AddModel(make([]*Reference, 0))
		return
	}

	if textures != nil {
		p.AddModel(textures)
	}
}

func (p *Processor) MapModelCompat(rad int, h int, tex int, name string, shadow string) {
	p.MModel(name)
}

func (p *Processor) AutoGrass(name string) {
	texture := p.SearchFile(NormalizeTexture(name))

	if texture != nil {
		p.AddFile(texture)
	}
}

func (p *Processor) RegisterSound(name string, vol int) {
	for _, _type := range []string{"", ".wav", ".ogg"} {
		path := fmt.Sprintf(
			"packages/sounds/%s%s",
			name,
			_type,
		)

		resolved := p.SearchFile(path)
		if resolved != nil {
			p.AddSound(resolved)
			break
		}
	}
}

func (p *Processor) MapSound(name string, vol int, maxUses int) {
	p.RegisterSound(name, vol)
}

func (p *Processor) LoadSky(name string) {
	for _, texture := range p.FindCubemap(NormalizeTexture(name)) {
		p.AddFile(texture)
	}
}

func (p *Processor) Exec(name string) {
	ref := p.SearchFile(name)
	if ref == nil {
		log.Printf("Could not find %s", name)
		return
	}

	p.ProcessFile(ref)
	p.AddFile(ref)
}

func (p *Processor) LoadSkyOverlay(name string) {
	resolved := p.FindTexture(name)

	if resolved != nil {
		p.AddFile(resolved)
	}
}

func (p *Processor) DoNothing() {
}

var EMPTY_COMMANDS = []string{
	"adaptivesample",
	"alias",
	"ambient",
	"blurlms",
	"blurskylight",
	"causticmillis",
	"causticscale",
	"cloudalpha",
	"cloudboxalpha",
	"cloudboxcolour",
	"cloudcolour",
	"cloudfade",
	"cloudheight",
	"cloudscale",
	"cloudscrollx",
	"cloudscrolly",
	"edgetolerance",
	"elevcontag",
	"fog",
	"fogcolour",
	"fogdomecap",
	"fogdomeclip",
	"fogdomeclouds",
	"fogdomecolour",
	"fogdomeheight",
	"fogdomemax",
	"fogdomemin",
	"grassalpha",
	"grasscolour",
	"lightlod",
	"lightprecision",
	"lmshadows",
	"mapmsg",
	"maptitle",
	"maxmerge",
	"minimapclip",
	"minimapcolour",
	"minimapheight",
	"panelset",
	"setshader",
	"setshaderparam",
	"shadowmapambient",
	"shadowmapangle",
	"skill",
	"skyboxcolour",
	"skylight",
	"skytexture",
	"skytexturelight",
	"smoothangle",
	"spinclouds",
	"spinsky",
	"sunlight",
	"sunlightpitch",
	"sunlightscale",
	"sunlightyaw",
	"texalpha",
	"texcolor",
	"texlayer",
	"texoffset",
	"texrotate",
	"texscale",
	"texscroll",
	"texsmooth",
	"water2colour",
	"water2fog",
	"watercolour",
	"waterfallcolour",
	"waterfog",
	"waterspec",
	"yawsky",
}

func (p *Processor) setupVM() {
	vm := p.cfgVM
	vm.AddCommand("autograss", p.AutoGrass)
	vm.AddCommand("cloudbox", p.LoadSky)
	vm.AddCommand("exec", p.Exec)
	vm.AddCommand("loadsky", p.LoadSky)
	vm.AddCommand("mapmodel", p.MapModelCompat)
	vm.AddCommand("mapmodelreset", p.ResetModels)
	vm.AddCommand("mapsound", p.MapSound)
	vm.AddCommand("mapsoundreset", p.ResetSounds)
	vm.AddCommand("materialreset", p.ResetMaterials)
	vm.AddCommand("mmodel", p.MModel)
	vm.AddCommand("registersound", p.RegisterSound)
	vm.AddCommand("skybox", p.LoadSky)
	vm.AddCommand("texture", p.Texture)
	vm.AddCommand("texturereset", p.TextureReset)

	for _, command := range EMPTY_COMMANDS {
		vm.AddCommand(command, p.DoNothing)
	}
}
