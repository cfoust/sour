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
	Type int
	Text string
	// TODO impl
	//switch(type)
	//{
	//case ID_VAR: getint(p); break;
	//case ID_FVAR: getfloat(p); break;
	//case ID_SVAR: getstring(text, p);
	//}
}

// N_EDITVSLOT
type EditVSlot struct {
	Sel      Selection
	Delta    int
	AllFaces int
	// TODO impl
	Extra1 byte
	Extra2 byte
}

// N_REDO
type Redo struct {
	// TODO impl
}

// N_UNDO
type Undo struct {
	// TODO impl
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
	Allfaces int
	Extra    []byte
}

var FAILED = fmt.Errorf("failed to unmarshal")

func (e *Editt) Unmarshal(p *Packet) error {
	err := p.Get(
		&e.Sel,
		&e.Tex,
		&e.Allfaces,
	)
	if err != nil {
		return err
	}

	q := Buffer(*p)
	numBytes, ok := q.GetShort()
	if !ok {
		return FAILED
	}
	e.Extra = q[:numBytes]
	q = q[numBytes:]

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
	Newtex int
	Insel  int
}

// N_CLIPBOARD
type Clipboard struct {
	Client    int
	UnpackLen int
	Data      []byte `type:"count"`
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
