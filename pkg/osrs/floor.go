package osrs

import (
	"fmt"
	"image/color"
	"math"
)

// Floor describes an OSRS floor texture (underlay or overlay).
type Floor struct {
	RGB        int // 24-bit RGB color
	Texture    int // texture ID, -1 if none
	Occlude    bool
	AnotherRGB int // secondary RGB, -1 if none

	// Derived HSL values
	Hue, Saturation, Luminance             int
	AnotherHue, AnotherSaturation, AnotherLuminance int
	BlendHue, BlendHueMultiplier           int
	HSL16                                  int
}

// Color returns the floor's RGB as a Go color.
func (f *Floor) Color() color.RGBA {
	return color.RGBA{
		R: uint8((f.RGB >> 16) & 0xff),
		G: uint8((f.RGB >> 8) & 0xff),
		B: uint8(f.RGB & 0xff),
		A: 255,
	}
}

// FloorDefs holds the underlay and overlay floor definitions.
type FloorDefs struct {
	Underlays []Floor
	Overlays  []Floor
}

// LoadFloorDefs parses flo.dat from a config archive.
func LoadFloorDefs(data []byte) (*FloorDefs, error) {
	buf := NewBuffer(data)

	underlayCount, err := buf.ReadUShort()
	if err != nil {
		return nil, fmt.Errorf("reading underlay count: %w", err)
	}

	underlays := make([]Floor, underlayCount)
	for i := 0; i < underlayCount; i++ {
		underlays[i].Texture = -1
		underlays[i].Occlude = true
		underlays[i].AnotherRGB = -1
		if err := decodeUnderlay(buf, &underlays[i]); err != nil {
			return nil, fmt.Errorf("underlay %d: %w", i, err)
		}
		generateHSL(&underlays[i])
	}

	overlayCount, err := buf.ReadUShort()
	if err != nil {
		return nil, fmt.Errorf("reading overlay count: %w", err)
	}

	overlays := make([]Floor, overlayCount)
	for i := 0; i < overlayCount; i++ {
		overlays[i].Texture = -1
		overlays[i].Occlude = true
		overlays[i].AnotherRGB = -1
		if err := decodeOverlay(buf, &overlays[i]); err != nil {
			return nil, fmt.Errorf("overlay %d: %w", i, err)
		}
		generateHSL(&overlays[i])
	}

	return &FloorDefs{
		Underlays: underlays,
		Overlays:  overlays,
	}, nil
}

func decodeUnderlay(buf *Buffer, f *Floor) error {
	for {
		opcode, err := buf.ReadUnsignedByte()
		if err != nil {
			return err
		}
		if opcode == 0 {
			return nil
		}
		if opcode == 1 {
			r, _ := buf.ReadUnsignedByte()
			g, _ := buf.ReadUnsignedByte()
			b, _ := buf.ReadUnsignedByte()
			f.RGB = (r << 16) + (g << 8) + b
		}
	}
}

func decodeOverlay(buf *Buffer, f *Floor) error {
	for {
		opcode, err := buf.ReadUnsignedByte()
		if err != nil {
			return err
		}
		switch opcode {
		case 0:
			return nil
		case 1:
			r, _ := buf.ReadUnsignedByte()
			g, _ := buf.ReadUnsignedByte()
			b, _ := buf.ReadUnsignedByte()
			f.RGB = (r << 16) + (g << 8) + b
		case 2:
			t, _ := buf.ReadUnsignedByte()
			f.Texture = t
		case 5:
			f.Occlude = false
		case 7:
			r, _ := buf.ReadUnsignedByte()
			g, _ := buf.ReadUnsignedByte()
			b, _ := buf.ReadUnsignedByte()
			f.AnotherRGB = (r << 16) + (g << 8) + b
		}
	}
}

func generateHSL(f *Floor) {
	if f.AnotherRGB != -1 {
		rgbToHSL(f.AnotherRGB, &f.AnotherHue, &f.AnotherSaturation, &f.AnotherLuminance, nil, nil)
	}
	rgbToHSL(f.RGB, &f.Hue, &f.Saturation, &f.Luminance, &f.BlendHue, &f.BlendHueMultiplier)
	f.HSL16 = hsl24to16(f.Hue, f.Saturation, f.Luminance)
}

func rgbToHSL(rgb int, hue, saturation, luminance *int, blendHue, blendHueMultiplier *int) {
	r := float64((rgb>>16)&0xff) / 256.0
	g := float64((rgb>>8)&0xff) / 256.0
	b := float64(rgb&0xff) / 256.0

	mn := math.Min(r, math.Min(g, b))
	mx := math.Max(r, math.Max(g, b))

	h := 0.0
	s := 0.0
	l := (mn + mx) / 2.0

	if mn != mx {
		if l < 0.5 {
			s = (mx - mn) / (mx + mn)
		} else {
			s = (mx - mn) / (2.0 - mx - mn)
		}
		if r == mx {
			h = (g - b) / (mx - mn)
		} else if g == mx {
			h = 2.0 + (b-r)/(mx-mn)
		} else if b == mx {
			h = 4.0 + (r-g)/(mx-mn)
		}
	}
	h /= 6.0

	*hue = int(h * 256.0)
	*saturation = int(s * 256.0)
	if *saturation < 0 {
		*saturation = 0
	} else if *saturation > 255 {
		*saturation = 255
	}
	*luminance = int(l * 256.0)
	if *luminance < 0 {
		*luminance = 0
	} else if *luminance > 255 {
		*luminance = 255
	}

	if blendHueMultiplier != nil {
		if l > 0.5 {
			*blendHueMultiplier = int((1.0 - l) * s * 512.0)
		} else {
			*blendHueMultiplier = int(l * s * 512.0)
		}
		if *blendHueMultiplier < 1 {
			*blendHueMultiplier = 1
		}
		*blendHue = int(h * float64(*blendHueMultiplier))
	}
}

func hsl24to16(h, s, l int) int {
	if l > 179 {
		s /= 2
	}
	if l > 192 {
		s /= 2
	}
	if l > 217 {
		s /= 2
	}
	if l > 243 {
		s /= 2
	}
	return (h/4)<<10 + (s/32)<<7 + l/2
}
