package min

import (
	"fmt"
	"path/filepath"

	"github.com/cfoust/sour/pkg/maps"

	"github.com/repeale/fp-go"
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
	err := p.ProcessModel(modelFile)

	if err != nil {
		log.Printf("Failed to process model %s", name)
		p.AddModel(make([]*Reference, 0))
		return
	}

	if p.ModelFiles != nil {
		p.AddModel(p.ModelFiles)
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
	"mdlalphablend",
	"mdlalphadepth",
	"mdlalphatest",
	"mdlambient",
	"mdlbb",
	"mdlcollide",
	"mdlcullface",
	"mdldepthoffset",
	"mdlellipsecollide",
	"mdlextendbb",
	"mdlfullbright",
	"mdlglare",
	"mdlglow",
	"mdlpitch",
	"mdlscale",
	"mdlshader",
	"mdlshadow",
	"mdlspec",
	"mdlspin",
	"mdltrans",
	"mdlyaw",
	"minimapclip",
	"minimapcolour",
	"minimapheight",
	"noclip",
	"panelset",
	"rdeye",
	"rdjoint",
	"rdlimitdist",
	"rdlimitrot",
	"rdtri",
	"rdvert",
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

var EMPTY_MODEL_COMMANDS = []string{
	"adjust",
	"alphablend",
	"alphatest",
	"ambient",
	"animpart",
	"cullface",
	"dir",
	"envmap",
	"fullbright",
	"glare",
	"glow",
	"link",
	"noclip",
	"pitch",
	"pitchcorrect",
	"pitchtarget",
	"scroll",
	"shader",
	"spec",
	"tag",
}

func expandTexture(texture string) []string {
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

func (p *Processor) AddModelTexture(name string) {
	for _, file := range expandTexture(name) {
		ref := p.SearchFile(file)
		if ref != nil {
			p.ModelFiles = append(p.ModelFiles, ref)
		}
	}
}

func (p *Processor) SetSkin(meshname string, tex string, masks string, envMapMax float32, envMapMin float32) {
	p.AddModelTexture(tex)
}

func (p *Processor) SetBumpMap(meshname string, normalMapFile string) {
	p.AddModelTexture(normalMapFile)
}

func (p *Processor) VertLoadPart(model string) {
	p.AddModelTexture(model)
}

func (p *Processor) VertSetAnim(anim string) {
	p.AddModelTexture(anim)
}

func (p *Processor) SkelLoadPart(model string, other string) {
	p.AddModelTexture(model)
}

func (p *Processor) SkelSetAnim(anim string, animFile string) {
	p.AddModelTexture(anim)
	p.AddModelTexture(animFile)
}

var (
	SKEL_MODEL_TYPES = []string{"md5", "iqm", "smd"}
	VERT_MODEL_TYPES = []string{"md3", "md2", "obj"}
)

func (p *Processor) MdlEnvMap(envMapMax float32, envMapMin float32, envMap string) {
	for _, texture := range p.FindCubemap(NormalizeTexture(envMap)) {
		p.ModelFiles = append(p.ModelFiles, texture)
	}
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
	vm.AddCommand("mdlenvmap", p.MdlEnvMap)

	addModelCommand := func(type_ string, name string, callback interface{}) {
		vm.AddCommand(
			type_+name,
			callback,
		)
	}

	for _, type_ := range VERT_MODEL_TYPES {
		addModelCommand(type_, "load", p.VertLoadPart)
		addModelCommand(type_, "anim", p.VertSetAnim)
	}

	for _, type_ := range SKEL_MODEL_TYPES {
		addModelCommand(type_, "load", p.SkelLoadPart)
		addModelCommand(type_, "anim", p.SkelSetAnim)
	}

	for _, type_ := range MODELTYPES {
		addModelCommand(type_, "skin", p.SetSkin)
		addModelCommand(type_, "bumpmap", p.SetBumpMap)

		for _, command := range EMPTY_MODEL_COMMANDS {
			addModelCommand(type_, command, p.DoNothing)
		}
	}

	for _, command := range EMPTY_COMMANDS {
		vm.AddCommand(command, p.DoNothing)
	}
}
