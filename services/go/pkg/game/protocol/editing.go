package protocol

import (
	"fmt"

	"github.com/cfoust/sour/pkg/game/io"
	"github.com/cfoust/sour/pkg/game/variables"
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

// If a server is not in "open edit" mode and we receive one of these messages
// from a user who is not the owner, we disconnect and reconnect them.
func IsOwnerOnly(code MessageCode) bool {
	return IsEditMessage(code) || code == N_EDITMODE
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
	Value variables.Variable
}

func (m EditVar) Type() MessageCode { return N_EDITVAR }

func (e *EditVar) Unmarshal(p *io.Packet) error {
	var type_ int32
	err := p.Get(
		&type_,
		&e.Key,
	)
	if err != nil {
		return err
	}

	switch variables.VariableType(type_) {
	case variables.VariableTypeInt:
		value, ok := p.GetInt()
		if !ok {
			return FAILED_EDIT
		}
		e.Value = variables.IntVariable(value)
	case variables.VariableTypeFloat:
		value, ok := p.GetFloat()
		if !ok {
			return FAILED_EDIT
		}
		e.Value = variables.FloatVariable(value)
	case variables.VariableTypeString:
		value, ok := p.GetString()
		if !ok {
			return FAILED_EDIT
		}
		e.Value = variables.StringVariable(value)
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

func (m EditVSlot) Type() MessageCode { return N_EDITVSLOT }

func (e *EditVSlot) Unmarshal(p *io.Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Delta,
		&e.AllFaces,
	)
	if err != nil {
		return err
	}

	q := io.Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED_EDIT
	}
	e.Extra, ok = q.GetBytes(int(numBytes))
	if !ok {
		return FAILED_EDIT
	}

	*p = io.Packet(q)

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

type Redo PackData

func (m Redo) Type() MessageCode { return N_REDO }

type Undo PackData

func (m Undo) Type() MessageCode { return N_UNDO }

type Clipboard PackData

func (m Clipboard) Type() MessageCode { return N_CLIPBOARD }

func (e *PackData) Unmarshal(p *io.Packet) error {
	err := p.Get(
		&e.Client,
		&e.UnpackLength,
		&e.PackLength,
	)
	if err != nil {
		return err
	}

	q := io.Buffer(*p)
	data, ok := q.GetBytes(int(e.PackLength))
	if !ok {
		return FAILED_EDIT
	}
	e.Data = data

	*p = io.Packet(q)

	return nil
}

// N_EDITF
type EditFace struct {
	Sel  Selection
	Dir  int
	Mode int
}

func (m EditFace) Type() MessageCode { return N_EDITF }

// N_EDITT
type EditTexture struct {
	Sel      Selection
	Tex      int
	AllFaces int
	Extra    []byte
}

func (m EditTexture) Type() MessageCode { return N_EDITT }

var FAILED_EDIT = fmt.Errorf("failed to unmarshal edit message")

func (e *EditTexture) Unmarshal(p *io.Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Tex,
		&e.AllFaces,
	)
	if err != nil {
		return err
	}

	q := io.Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED_EDIT
	}
	e.Extra, ok = q.GetBytes(int(numBytes))
	if !ok {
		return FAILED_EDIT
	}

	*p = io.Packet(q)

	return nil
}

// N_EDITM
type EditMaterial struct {
	Sel    Selection
	Mat    int
	Filter int
}

func (m EditMaterial) Type() MessageCode { return N_EDITM }

// N_EDITENT
type EditEntity struct {
	Index      int
	Position   Vec
	EntityType int
	Attr1      int
	Attr2      int
	Attr3      int
	Attr4      int
	Attr5      int
}

func (m EditEntity) Type() MessageCode { return N_EDITENT }

// N_COPY
type Copy struct {
	Sel Selection
}

func (m Copy) Type() MessageCode { return N_COPY }

// N_PASTE
type Paste struct {
	Sel Selection
}

func (m Paste) Type() MessageCode { return N_PASTE }

// N_FLIP
type Flip struct {
	Sel Selection
}

func (m Flip) Type() MessageCode { return N_FLIP }

// N_ROTATE
type Rotate struct {
	Sel Selection
	Dir int
}

func (m Rotate) Type() MessageCode { return N_ROTATE }

// N_REPLACE
type Replace struct {
	Sel    Selection
	Tex    int
	NewTex int
	Insel  int
	Extra  []byte
}

func (m Replace) Type() MessageCode { return N_REPLACE }

func (e *Replace) Unmarshal(p *io.Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Tex,
		&e.NewTex,
		&e.Insel,
	)
	if err != nil {
		return err
	}

	q := io.Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED_EDIT
	}
	e.Extra, ok = q.GetBytes(int(numBytes))
	if !ok {
		return FAILED_EDIT
	}

	*p = io.Packet(q)

	return nil
}

// N_DELCUBE
type DeleteCube struct {
	Sel Selection
}

func (m DeleteCube) Type() MessageCode { return N_DELCUBE }

// N_NEWMAP
type NewMap struct {
	Size int
}

func (m NewMap) Type() MessageCode { return N_NEWMAP }

// N_REMIP
type Remip struct {
}

func (m Remip) Type() MessageCode { return N_REMIP }
