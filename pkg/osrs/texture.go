package osrs

import (
	"fmt"
	"image"
	"image/color"
)

const TextureCount = 61

// LoadTextures extracts all textures from the texture archive.
// Returns up to 61 RGBA images indexed by texture ID.
func LoadTextures(cache *Cache) ([]*image.RGBA, error) {
	data, err := cache.ReadFile(0, 6) // TEXTURES_CRC = 6
	if err != nil {
		return nil, fmt.Errorf("reading texture archive: %w", err)
	}

	archive, err := DecodeArchive(data)
	if err != nil {
		return nil, fmt.Errorf("decoding texture archive: %w", err)
	}

	// Read the shared index.dat
	indexData, err := archive.ReadFile("index.dat")
	if err != nil {
		return nil, fmt.Errorf("reading index.dat: %w", err)
	}

	textures := make([]*image.RGBA, TextureCount)

	for id := 0; id < TextureCount; id++ {
		name := fmt.Sprintf("%d.dat", id)
		pixelData, err := archive.ReadFile(name)
		if err != nil {
			continue
		}

		img, err := decodeIndexedImage(pixelData, indexData, 0)
		if err != nil {
			continue
		}
		textures[id] = img
	}

	return textures, nil
}

// decodeIndexedImage decodes an OSRS IndexedImage from pixel data and shared index data.
func decodeIndexedImage(pixelData, indexData []byte, frameID int) (*image.RGBA, error) {
	pixBuf := NewBuffer(pixelData)
	idxBuf := NewBuffer(indexData)

	// Pixel data starts with a uint16 offset into index.dat
	offset, err := pixBuf.ReadUShort()
	if err != nil {
		return nil, err
	}
	idxBuf.SetPosition(offset)

	resizeWidth, _ := idxBuf.ReadUShort()
	resizeHeight, _ := idxBuf.ReadUShort()

	// Read palette
	colorLength, _ := idxBuf.ReadUnsignedByte()
	palette := make([]int, colorLength)
	for i := 0; i < colorLength-1; i++ {
		rgb, _ := idxBuf.ReadTriByte()
		palette[i+1] = rgb // palette[0] is transparent (0)
	}

	// Skip frames before the one we want
	for i := 0; i < frameID; i++ {
		idxBuf.Skip(2) // drawOffset
		w, _ := idxBuf.ReadUShort()
		h, _ := idxBuf.ReadUShort()
		pixBuf.Skip(w * h)
		idxBuf.Skip(1) // type
	}

	drawOffsetX, _ := idxBuf.ReadUnsignedByte()
	drawOffsetY, _ := idxBuf.ReadUnsignedByte()
	width, _ := idxBuf.ReadUShort()
	height, _ := idxBuf.ReadUShort()
	pixelType, _ := idxBuf.ReadUnsignedByte()

	if width == 0 || height == 0 {
		return nil, fmt.Errorf("zero-size image")
	}

	// Read palette indices
	palettePixels := make([]byte, width*height)
	if pixelType == 0 {
		// Row-major
		for i := 0; i < width*height; i++ {
			b, _ := pixBuf.ReadUnsignedByte()
			palettePixels[i] = byte(b)
		}
	} else {
		// Column-major
		for x := 0; x < width; x++ {
			for y := 0; y < height; y++ {
				b, _ := pixBuf.ReadUnsignedByte()
				palettePixels[x+y*width] = byte(b)
			}
		}
	}

	// Build the final image at resizeWidth x resizeHeight
	finalW := resizeWidth
	finalH := resizeHeight
	if finalW == 0 {
		finalW = width
	}
	if finalH == 0 {
		finalH = height
	}

	img := image.NewRGBA(image.Rect(0, 0, finalW, finalH))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := int(palettePixels[x+y*width])
			if idx == 0 {
				continue // transparent
			}
			if idx >= len(palette) {
				continue
			}
			rgb := palette[idx]
			px := x + drawOffsetX
			py := y + drawOffsetY
			if px >= 0 && px < finalW && py >= 0 && py < finalH {
				img.Set(px, py, color.RGBA{
					R: uint8((rgb >> 16) & 0xff),
					G: uint8((rgb >> 8) & 0xff),
					B: uint8(rgb & 0xff),
					A: 255,
				})
			}
		}
	}

	return img, nil
}
