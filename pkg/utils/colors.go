package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Color struct {
	R byte
	G byte
	B byte
}

func (c Color) ToUint() (color uint32) {
	color = color | (uint32(c.R) << 16)
	color = color | (uint32(c.G) << 8)
	color = color | uint32(c.B)
	return color
}

func (c Color) ToHex() string {
	return fmt.Sprintf("#%06x", c.ToUint())
}

func (c Color) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ToHex())
}

func ColorFromUint(value uint32) Color {
	var c Color
	c.R = byte((value >> 16) & 0xFF)
	c.G = byte((value >> 8) & 0xFF)
	c.B = byte(value & 0xFF)
	return c
}

func (c *Color) UnmarshalJSON(data []byte) error {
	var hex string
	err := json.Unmarshal(data, &hex)
	if err == nil {
		value, err := strconv.ParseUint(hex[1:], 16, 32)
		if err != nil {
			return err
		}

		*c = ColorFromUint(uint32(value))
		return nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return err
	}

	elements := [3]byte{}
	err = json.Unmarshal(data, &elements)
	if err == nil {
		c.R = elements[0]
		c.G = elements[1]
		c.B = elements[2]
		return nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return err
	}

	full := struct {
		R byte
		G byte
		B byte
	}{}
	err = json.Unmarshal(data, &full)
	if err == nil {
		c.R = full.R
		c.G = full.G
		c.B = full.B
		return nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return err
	}

	return fmt.Errorf("could not deserialize color")
}
