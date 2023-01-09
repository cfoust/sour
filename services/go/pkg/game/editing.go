package game

import (
	"fmt"
)

var EDIT_MESSAGES = []MessageCode{
	N_COPY,
	N_DELCUBE,
	N_EDITENT,
	N_EDITF,
	N_EDITM,
	N_EDITT,
	N_EDITVAR,
	N_EDITVSLOT,
	N_FLIP,
	N_NEWMAP,
	N_PASTE,
	N_REMIP,
	N_REPLACE,
	N_ROTATE,
}

func IsEditMessage(code MessageCode) bool {
	for _, editType := range EDIT_MESSAGES {
		if editType == code {
			return true
		}
	}

	return false
}

type Selection struct {
	O      IVec
	S      IVec
	Grid   int
	Orient int
	Cx     int
	Cxs    int
	Cy     int
	Cys    int
	Corner int
}

// N_EDITVAR
type EditVar struct {
	Key   string
	Value Variable
}

func (e *EditVar) Unmarshal(p *Packet) error {
	var type_ int32
	err := p.Get(
		&type_,
		&e.Key,
	)
	if err != nil {
		return err
	}

	switch VariableType(type_) {
	case VariableTypeInt:
		value, ok := p.GetInt()
		if !ok {
			return FAILED
		}
		e.Value = IntVariable(value)
	case VariableTypeFloat:
		value, ok := p.GetFloat()
		if !ok {
			return FAILED
		}
		e.Value = FloatVariable(value)
	case VariableTypeString:
		value, ok := p.GetString()
		if !ok {
			return FAILED
		}
		e.Value = StringVariable(value)
	}

	return nil
}

// N_EDITVSLOT
type EditVSlot struct {
	Sel      Selection
	Delta    int
	AllFaces int
	Extra    []byte
}

func (e *EditVSlot) Unmarshal(p *Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Delta,
		&e.AllFaces,
	)
	if err != nil {
		return err
	}

	q := Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED
	}
	e.Extra, ok = q.GetBytes(int(numBytes))
	if !ok {
		return FAILED
	}

	*p = Packet(q)

	return nil
}

// These are the same
// N_REDO
// N_UNDO
// N_CLIPBOARD
type PackData struct {
	Client       int
	UnpackLength int
	PackLength   int
	Data         []byte
}

func (e *PackData) Unmarshal(p *Packet) error {
	err := p.Get(
		&e.Client,
		&e.UnpackLength,
		&e.PackLength,
	)
	if err != nil {
		return err
	}

	q := Buffer(*p)
	data, ok := q.GetBytes(int(e.PackLength))
	if !ok {
		return FAILED
	}
	e.Data = data

	*p = Packet(q)

	return nil
}

// N_EDITF
type Editf struct {
	Sel  Selection
	Dir  int
	Mode int
}

// N_EDITT
type Editt struct {
	Sel      Selection
	Tex      int
	AllFaces int
	Extra    []byte
}

var FAILED = fmt.Errorf("failed to unmarshal")

func (e *Editt) Unmarshal(p *Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Tex,
		&e.AllFaces,
	)
	if err != nil {
		return err
	}

	q := Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED
	}
	e.Extra, ok = q.GetBytes(int(numBytes))
	if !ok {
		return FAILED
	}

	*p = Packet(q)

	return nil
}

// N_EDITM
type Editm struct {
	Sel    Selection
	Mat    int
	Filter int
}

// N_EDITENT
type EditEnt struct {
	Entid int
	X     int
	Y     int
	Z     int
	Type  int
	Attr1 int
	Attr2 int
	Attr3 int
	Attr4 int
	Attr5 int
}

// N_COPY
type Copy struct {
	Sel Selection
}

// N_PASTE
type Paste struct {
	Sel Selection
}

// N_FLIP
type Flip struct {
	Sel Selection
}

// N_ROTATE
type Rotate struct {
	Sel Selection
	Dir int
}

// N_REPLACE
type Replace struct {
	Sel    Selection
	Tex    int
	NewTex int
	Insel  int
	Extra  []byte
}

func (e *Replace) Unmarshal(p *Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Tex,
		&e.NewTex,
		&e.Insel,
	)
	if err != nil {
		return err
	}

	q := Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED
	}
	e.Extra, ok = q.GetBytes(int(numBytes))
	if !ok {
		return FAILED
	}

	*p = Packet(q)

	return nil
}

// N_DELCUBE
type Delcube struct {
	Sel Selection
}

// N_NEWMAP
type NewMap struct {
	Size int
}

// N_REMIP
type Remip struct {
}
