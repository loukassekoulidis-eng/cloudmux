package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

func generateIcon(dotColor color.Color) []byte {
	const size = 22
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Transparent background
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.Transparent)
		}
	}

	// Draw simple "M" letterform
	white := color.RGBA{230, 230, 234, 255}
	for y := 4; y < 18; y++ {
		img.Set(3, y, white)
		img.Set(4, y, white)
		img.Set(18, y, white)
		img.Set(19, y, white)
	}
	for i := 0; i < 7; i++ {
		img.Set(5+i, 5+i, white)
		img.Set(17-i, 5+i, white)
	}

	// Draw notification dot if color provided
	if dotColor != nil {
		cx, cy, r := 18, 3, 3
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r {
					px, py := cx+dx, cy+dy
					if px >= 0 && px < size && py >= 0 && py < size {
						img.Set(px, py, dotColor)
					}
				}
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

var (
	iconIdle   = generateIcon(nil)
	iconBlue   = generateIcon(color.RGBA{10, 132, 255, 255})
	iconYellow = generateIcon(color.RGBA{255, 159, 10, 255})
	iconRed    = generateIcon(color.RGBA{255, 69, 58, 255})
)
