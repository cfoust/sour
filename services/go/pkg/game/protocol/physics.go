package protocol

import (
	"math"

	"github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/game/io"
)

type Vec struct {
	X float64
	Y float64
	Z float64
}

func (v *Vec) SquaredLen() float64 {
	return v.Z*v.Z + v.Z*v.Z + v.Z*v.Z
}

func (v *Vec) Magnitude() float64 {
	return math.Sqrt(v.SquaredLen())
}

func (v Vec) Scale(factor float64) Vec {
	return Vec{
		X: v.X * factor,
		Y: v.Y * factor,
		Z: v.Z * factor,
	}
}

func (v Vec) IsZero() bool {
	return v.X == 0 && v.Y == 0 && v.Z == 0
}

type IVec struct {
	X int32
	Y int32
	Z int32
}

func IVecFromVec(v Vec) *IVec {
	return &IVec{
		X: int32(v.X),
		Y: int32(v.Y),
		Z: int32(v.Z),
	}
}

func minInt32(a int32, b int32) int32 {
	if a < b {
		return a
	}

	return b
}

type PhysicsState struct {
	State        byte
	LifeSequence int32
	Yaw          float64
	Roll         float64
	Pitch        float64
	Move         int8
	Strafe       int8
	O            Vec
	Falling      Vec
	Velocity     Vec
}

func getComponent(p *io.Packet, flags uint32, k uint32) float64 {
	r, _ := p.GetByte()
	n := int(r)
	r, _ = p.GetByte()
	n |= int(r) << 8
	if flags&(1<<k) > 0 {
		r, _ = p.GetByte()
		n |= int(r) << 16
		if n&0x800000 > 0 {
			n |= -1 << 24
		}
	}

	return float64(n)
}

func clamp(a int, b int, c int) int {
	if a < b {
		return b
	}
	if a > c {
		return c
	}

	return a
}

const RAD = math.Pi / 180.0

func vecFromYawPitch(yaw float64, pitch float64, move int8, strafe int8) Vec {
	m := Vec{}
	if move > 0 {
		m.X = float64(move) * -math.Sin(RAD*yaw)
		m.Y = float64(move) * math.Cos(RAD*yaw)
	} else {
		m.X = 0
		m.Y = 0
	}

	if pitch > 0 {
		m.X *= math.Cos(RAD * pitch)
		m.Y *= math.Cos(RAD * pitch)
		m.Z = float64(move) * math.Sin(RAD*pitch)
	} else {
		m.Z = 0
	}

	if strafe > 0 {
		m.X += float64(strafe) * math.Cos(RAD*yaw)
		m.Y += float64(strafe) * math.Sin(RAD*yaw)
	}
	return m
}

func vecToYawPitch(v Vec) (yaw float64, pitch float64) {
	if v.IsZero() {
		yaw = 0
		pitch = 0
	} else {
		yaw = -math.Atan2(v.X, v.Y) / RAD
		pitch = math.Asin(v.Z/v.Magnitude()) / RAD
	}

	return yaw, pitch
}

func (d *PhysicsState) Unmarshal(p *io.Packet) error {
	r, _ := p.GetByte()
	state := r
	flags, _ := p.GetUint()

	d.O.X = getComponent(p, flags, 0)
	d.O.Y = getComponent(p, flags, 1)
	d.O.Z = getComponent(p, flags, 2)

	r, _ = p.GetByte()
	dir := int(r)
	r, _ = p.GetByte()
	dir |= int(r) << 8
	var yaw float64 = float64(dir % 360)
	var pitch float64 = float64(clamp(dir/360, 0, 180) - 90)
	r, _ = p.GetByte()
	var roll float64 = float64(clamp(int(r), 0, 180) - 90)
	r, _ = p.GetByte()
	mag := int(r)
	if flags&(1<<3) > 0 {
		r, _ = p.GetByte()
		mag |= int(r) << 8
	}
	r, _ = p.GetByte()
	dir = int(r)
	r, _ = p.GetByte()
	dir |= int(r) << 8

	d.Velocity = vecFromYawPitch(float64(dir%360), float64(clamp(dir/360, 0, 180)-90), 1, 0)

	falling := Vec{}
	if flags&(1<<4) > 0 {
		r, _ = p.GetByte()
		mag := int(r)
		if flags&(1<<5) > 0 {
			r, _ = p.GetByte()
			mag |= int(r) << 8
		}

		if flags&(1<<6) > 0 {
			r, _ = p.GetByte()
			dir = int(r)
			r, _ = p.GetByte()
			dir |= int(r) << 8
			falling = vecFromYawPitch(float64(dir%360), float64(clamp(dir/360, 0, 180)-90), 1, 0)
		} else {
			falling = Vec{
				X: 0,
				Y: 0,
				Z: -1,
			}
		}
	}

	d.Falling = falling

	d.Yaw = yaw
	d.Pitch = pitch
	d.Roll = roll

	if (state>>4)&2 > 0 {
		d.Move = -1
	} else {
		d.Move = (int8(state) >> 4) & 1
	}

	if (state>>6)&2 > 0 {
		d.Strafe = -1
	} else {
		d.Strafe = (int8(state) >> 6) & 1
	}

	d.State = state & 7
	return nil
}

func writeDirection(p *io.Packet, pitch float64, yaw float64) error {
	var dir uint32 = uint32(clamp(
		int(pitch+90),
		0,
		180,
	)) * 360
	if yaw < 0 {
		dir += 360 + uint32(yaw)%360
	} else {
		dir += uint32(yaw) % 360
	}

	return p.Put(
		byte(dir&0xFF),
		byte((dir>>8)&0xFF),
	)
}

func (state PhysicsState) Marshal(p *io.Packet) error {
	var physState byte = state.State |
		byte((state.LifeSequence&1)<<3) |
		byte((state.Move&3)<<4) |
		byte((state.Strafe&3)<<6)
	err := p.Put(physState)
	if err != nil {
		return err
	}

	o := IVecFromVec(
		Vec{
			X: state.O.X,
			Y: state.O.Y,
			Z: state.O.Z - constants.DEFAULT_EYE_HEIGHT,
		}.Scale(constants.DMF),
	)

	var vel uint32 = uint32(minInt32(int32(state.Velocity.Magnitude()*constants.DVELF), 0xFFFF))
	var fall uint32 = uint32(minInt32(int32(state.Falling.Magnitude()*constants.DVELF), 0xFFFF))
	var flags uint32 = 0
	if o.X < 0 || o.X > 0xFFFF {
		flags |= 1 << 0
	}
	if o.Y < 0 || o.Y > 0xFFFF {
		flags |= 1 << 1
	}
	if o.Z < 0 || o.Z > 0xFFFF {
		flags |= 1 << 2
	}
	if vel > 0xFF {
		flags |= 1 << 3
	}
	if fall > 0 {
		flags |= 1 << 4
		if fall > 0xFF {
			flags |= 1 << 5
		}
		if state.Falling.X == 1 || state.Falling.Y == 1 || state.Falling.Z > 0 {
			flags |= 1 << 6
		}
	}

	// TODO
	//if((lookupmaterial(d->feetpos())&MATF_CLIP) == MAT_GAMECLIP) flags |= 1<<7;
	err = p.Put(flags)
	if err != nil {
		return err
	}

	for _, val := range []int32{o.X, o.Y, o.Z} {
		p.Put(
			byte(val&0xFF),
			byte((val>>8)&0xFF),
		)
		if val < 0 || val > 0xFFFF {
			p.Put(byte((val >> 16) & 0xFF))
		}
	}

	//uint dir = (d->yaw < 0 ? 360 + int(d->yaw)%360 : int(d->yaw)%360) + clamp(int(d->pitch+90), 0, 180)*360;
	err = writeDirection(p, state.Pitch, state.Yaw)
	if err != nil {
		return err
	}

	err = p.Put(
		clamp(int(state.Roll+90), 0, 180),
		vel&0xFF,
	)
	if err != nil {
		return err
	}

	if vel > 0xFF {
		p.Put((vel >> 8) & 0xFF)
	}

	velyaw, velpitch := vecToYawPitch(state.Velocity)
	err = writeDirection(p, velpitch, velyaw)
	if err != nil {
		return err
	}

	if fall > 0 {
		p.Put(fall & 0xFF)
		if fall > 0xFF {
			p.Put((fall >> 8) & 0xFF)
		}

		if state.Falling.X == 1 || state.Falling.Y == 1 || state.Falling.Z > 0 {
			fallyaw, fallpitch := vecToYawPitch(state.Falling)
			writeDirection(p, fallpitch, fallyaw)
		}
	}

	return nil
}

var _ io.Marshalable = (*PhysicsState)(nil)
var _ io.Unmarshalable = (*PhysicsState)(nil)
