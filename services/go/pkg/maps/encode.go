package maps

import (
	"bytes"
	"compress/gzip"

	"github.com/cfoust/sour/pkg/game"
	"github.com/rs/zerolog/log"
)

func saveVSlot(p *game.Packet, vs *VSlot, prev int32) error {
	err := p.PutRaw(
		vs.Changed,
		prev,
	)
	if err != nil {
		return err
	}

	if (vs.Changed & (1 << VSLOT_SHPARAM)) > 0 {
		err := p.PutRaw(
			uint16(len(vs.Params)),
		)
		if err != nil {
			return err
		}

		for _, param := range vs.Params {
			err := p.PutRaw(
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
		p.PutRaw(&vs.Scale)
	}

	if (changed & (1 << VSLOT_ROTATION)) > 0 {
		p.PutRaw(&vs.Rotation)
	}

	if (changed & (1 << VSLOT_OFFSET)) > 0 {
		p.PutRaw(
			&vs.Offset.X,
			&vs.Offset.Y,
		)
	}

	if (changed & (1 << VSLOT_SCROLL)) > 0 {
		p.PutRaw(
			&vs.Scroll.X,
			&vs.Scroll.Y,
		)
	}

	if (changed & (1 << VSLOT_LAYER)) > 0 {
		p.PutRaw(&vs.Layer)
	}

	if (changed & (1 << VSLOT_ALPHA)) > 0 {
		p.PutRaw(
			&vs.AlphaFront,
			&vs.AlphaBack,
		)
	}

	if (changed & (1 << VSLOT_COLOR)) > 0 {
		p.PutRaw(
			&vs.ColorScale.X,
			&vs.ColorScale.Y,
			&vs.ColorScale.Z,
		)
	}

	return nil
}

func saveVSlots(p *game.Packet, slots []*VSlot) error {
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
			p.PutRaw(int32(-(i - lastRoot)))
		}
		saveVSlot(p, vs, prev[i])
		lastRoot = i + 1
	}

	if lastRoot < numVSlots {
		p.PutRaw(int32(-(numVSlots - lastRoot)))
	}

	return nil
}

func (m *GameMap) Encode() ([]byte, error) {
	p := game.Packet{}

	err := p.PutRaw(
		FileHeader{
			Magic:      [4]byte{byte('O'), byte('C'), byte('T'), byte('A')},
			Version:    MAP_VERSION,
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

	defaults := GetDefaultVariables()

	for key, variable := range m.Vars {
		defaultValue, defaultExists := defaults[key]
		if !defaultExists || defaultValue.Type() != variable.Type() {
			log.
				Warn().
				Msgf("variable %s is not a valid map variable or invalid type", key)
		}
		err = p.PutRaw(
			byte(variable.Type()),
			uint16(len(key)),
			[]byte(key),
		)
		if err != nil {
			return p, err
		}

		switch variable.Type() {
		case VariableTypeInt:
			err = p.PutRaw(variable.(IntVariable))
		case VariableTypeFloat:
			err = p.PutRaw(variable.(FloatVariable))
		case VariableTypeString:
			value := variable.(StringVariable)
			err = p.PutRaw(
				uint16(len(value)),
				[]byte(value),
			)
		}
		if err != nil {
			return p, err
		}
	}

	err = p.PutRaw(
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
		err = p.PutRaw(entity)
		if err != nil {
			return p, err
		}
	}

	err = saveVSlots(&p, m.VSlots)
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
