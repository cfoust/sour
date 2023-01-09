package maps

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"unsafe"
	"os"

	"github.com/cfoust/sour/pkg/game"

	"github.com/cfoust/sour/pkg/maps/worldio"
)

func saveVSlot(p *game.Buffer, vs *VSlot, prev int32) error {
	err := p.Put(
		vs.Changed,
		prev,
	)
	if err != nil {
		return err
	}

	if (vs.Changed & (1 << VSLOT_SHPARAM)) > 0 {
		err := p.Put(
			uint16(len(vs.Params)),
		)
		if err != nil {
			return err
		}

		for _, param := range vs.Params {
			err := p.Put(
				uint16(len(param.Name)),
				[]byte(param.Name),
				param.Val[0],
				param.Val[1],
				param.Val[2],
				param.Val[3],
			)
			if err != nil {
				return err
			}
		}
	}

	changed := vs.Changed
	if (changed & (1 << VSLOT_SCALE)) > 0 {
		p.Put(&vs.Scale)
	}

	if (changed & (1 << VSLOT_ROTATION)) > 0 {
		p.Put(&vs.Rotation)
	}

	if (changed & (1 << VSLOT_OFFSET)) > 0 {
		p.Put(
			&vs.Offset.X,
			&vs.Offset.Y,
		)
	}

	if (changed & (1 << VSLOT_SCROLL)) > 0 {
		p.Put(
			&vs.Scroll.X,
			&vs.Scroll.Y,
		)
	}

	if (changed & (1 << VSLOT_LAYER)) > 0 {
		p.Put(&vs.Layer)
	}

	if (changed & (1 << VSLOT_ALPHA)) > 0 {
		p.Put(
			&vs.AlphaFront,
			&vs.AlphaBack,
		)
	}

	if (changed & (1 << VSLOT_COLOR)) > 0 {
		p.Put(
			&vs.ColorScale.X,
			&vs.ColorScale.Y,
			&vs.ColorScale.Z,
		)
	}

	return nil
}

func saveVSlots(p *game.Buffer, slots []*VSlot) error {
	numVSlots := len(slots)
	if numVSlots == 0 {
		return nil
	}

	// worldio.cpp:785

	prev := make([]int32, numVSlots)
	for i := 0; i < numVSlots; i++ {
		prev[i] = -1
	}

	for _, slot := range slots {
		vs := slot
		if vs.Changed == 1 {
			continue
		}

		for {
			cur := vs
			for vs != nil && int(vs.Index) > numVSlots {
				vs = vs.Next
			}
			if vs == nil {
				break
			}
			prev[vs.Index] = cur.Index
		}
	}

	lastRoot := 0
	for i, slot := range slots {
		vs := slot
		if vs.Changed == 0 {
			continue
		}
		if lastRoot < i {
			p.Put(int32(-(i - lastRoot)))
		}
		saveVSlot(p, vs, prev[i])
		lastRoot = i + 1
	}

	if lastRoot < numVSlots {
		p.Put(int32(-(numVSlots - lastRoot)))
	}

	return nil
}

func MapToCXX(cube *Cube) worldio.Cube {
	parent := worldio.New_CubeArray(CUBE_FACTOR)
	for i := 0; i < CUBE_FACTOR; i++ {
		child := cube.Children[i]
		cxx := worldio.Getcubeindex(parent, i)

		if child.Children != nil && len(child.Children) > 0 {
			mapped := MapToCXX(child)
			cxx.SetChildren(mapped)
		}

		ext := worldio.NewCubeext()
		surfaces := worldio.New_SurfaceInfoArray(6)
		for j := 0; j < 6; j++ {
			surface := worldio.NewSurfaceinfo()
			surface.SetVerts(child.SurfaceInfo[j].Verts)
			surface.SetNumverts(child.SurfaceInfo[j].NumVerts)

			lmid := worldio.New_UcharArray(2)
			worldio.UcharArray_setitem(lmid, 0, child.SurfaceInfo[j].Lmid[0])
			worldio.UcharArray_setitem(lmid, 1, child.SurfaceInfo[j].Lmid[1])
			surface.SetLmid(lmid)
		}
		ext.SetSurfaces(surfaces)
		cxx.SetExt(ext)

		// edges
		for j := 0; j < 12; j++ {
			worldio.Cube_setedge(cxx, j, child.Edges[j])
		}

		for j := 0; j < 6; j++ {
			worldio.Cube_settexture(cxx, j, child.Texture[j])
		}

		cxx.SetMaterial(cube.Material)
		cxx.SetMaterial(cube.Material)
		cxx.SetMerged(cube.Merged)
		cxx.SetEscaped(cube.Escaped)

		worldio.CubeArray_setitem(parent, i, cxx)
	}

	return parent
}

func SaveChildren(p *game.Buffer, cube *Cube, size int32) error {
	buf := make([]byte, 20000000) // 20 MiB
	root := MapToCXX(cube)
	numBytes := worldio.Savec_buf(
		uintptr(unsafe.Pointer(&(buf)[0])),
		uint(len(buf)),
		root,
		int(size),
	)
	if numBytes == 0 {
		return fmt.Errorf("failed to write cubes")
	}
	(*p) = append(*p, buf[:numBytes]...)
	worldio.Freeocta(root)
	return nil
}

func (m *GameMap) Encode() ([]byte, error) {
	p := game.Buffer{}

	err := p.Put(
		FileHeader{
			Magic:      [4]byte{byte('O'), byte('C'), byte('T'), byte('A')},
			Version:    game.MAP_VERSION,
			HeaderSize: 40,
			WorldSize:  m.Header.WorldSize,
			NumEnts:    int32(len(m.Entities)),
			// TODO
			NumPVs:    0,
			LightMaps: 0,
		},
		NewFooter{
			BlendMap: 0,
			NumVars:  int32(len(m.Vars)),
			// TODO
			NumVSlots: int32(len(m.VSlots)),
		},
	)
	if err != nil {
		return p, err
	}

	defaults := game.DEFAULT_VARIABLES

	for key, variable := range m.Vars {
		defaultValue, defaultExists := defaults[key]
		if !defaultExists || defaultValue.Type() != variable.Type() {
			return p, fmt.Errorf("variable %s is not a valid map variable or invalid type", key)
		}
		err = p.Put(
			byte(variable.Type()),
			uint16(len(key)),
			[]byte(key),
		)
		if err != nil {
			return p, err
		}

		switch variable.Type() {
		case game.VariableTypeInt:
			err = p.Put(variable.(game.IntVariable))
		case game.VariableTypeFloat:
			err = p.Put(variable.(game.FloatVariable))
		case game.VariableTypeString:
			value := variable.(game.StringVariable)
			if len(value) >= game.MAXSTRLEN {
				return p, fmt.Errorf(
					"svar value %s is too long (%d > %d)",
					key,
					len(value),
					game.MAXSTRLEN,
				)
			}
			err = p.Put(
				uint16(len(value)),
				[]byte(value),
			)
		}
		if err != nil {
			return p, err
		}
	}

	err = p.Put(
		// game type (almost always FPS)
		byte(len(m.Header.GameType)),
		[]byte(m.Header.GameType),
		byte(0), // null terminated

		uint16(0), // eif
		uint16(0), // extras

		uint16(0), // texture MRU
	)
	if err != nil {
		return p, err
	}

	for _, entity := range m.Entities {
		err = p.Put(entity)
		if err != nil {
			return p, err
		}
	}

	err = saveVSlots(&p, m.VSlots)
	if err != nil {
		return p, err
	}

	err = SaveChildren(&p, m.WorldRoot, m.Header.WorldSize)
	if err != nil {
		return p, err
	}

	return p, nil
}

func (m *GameMap) EncodeOGZ() ([]byte, error) {
	data, err := m.Encode()
	if err != nil {
		return data, err
	}

	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	_, err = gz.Write(data)
	if err != nil {
		return nil, err
	}
	gz.Close()

	return buffer.Bytes(), nil
}

func (m *GameMap) ToFile(path string) error {
	data, err := m.EncodeOGZ()
	if err != nil {
		return err
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(data)
	if err != nil {
		return err
	}

	return nil
}

