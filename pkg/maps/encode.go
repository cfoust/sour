package maps

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"unsafe"

	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/game/io"
	V "github.com/cfoust/sour/pkg/game/variables"

	"github.com/cfoust/sour/pkg/maps/worldio"
)

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

		worldio.CubeArray_setitem(parent, int64(i), cxx)
	}

	return parent
}

func SaveChildren(p *io.Buffer, cube *Cube, size int32) error {
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

func SavePartial(p *io.Buffer, header Header, state worldio.MapState) error {
	buf := make([]byte, 20000000) // 20 MiB
	numBytes := worldio.Partial_save_world(
		uintptr(unsafe.Pointer(&(buf)[0])),
		int64(len(buf)),
		state,
		int(header.WorldSize),
	)
	if numBytes == 0 {
		return fmt.Errorf("failed to write cubes")
	}
	(*p) = append(*p, buf[:numBytes]...)
	return nil
}

func (m *GameMap) Encode() ([]byte, error) {
	p := io.Buffer{}

	err := p.Put(
		FileHeader{
			Magic:      [4]byte{byte('O'), byte('C'), byte('T'), byte('A')},
			Version:    C.MAP_VERSION,
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
			NumVSlots: int32(worldio.Getnumvslots(m.C)),
		},
	)
	if err != nil {
		return p, err
	}

	defaults := V.DEFAULT_VARIABLES

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
		case V.VariableTypeInt:
			err = p.Put(variable.(V.IntVariable))
		case V.VariableTypeFloat:
			err = p.Put(variable.(V.FloatVariable))
		case V.VariableTypeString:
			value := variable.(V.StringVariable)
			if len(value) >= C.MAXSTRLEN {
				return p, fmt.Errorf(
					"svar value %s is too long (%d > %d)",
					key,
					len(value),
					C.MAXSTRLEN,
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

	err = SavePartial(&p, m.Header, m.C)
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
