package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"os"
)

// Rect is a region in pixels: [x0, y0, x1, y1].
type Rect [4]int

// Layout describes where the card's text is.
type Layout struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Name         Rect   `json:"name"`
	Tags         []Rect `json:"tags"`
	Effect       Rect   `json:"effect"`
	Ambrosia     Rect   `json:"ambrosia"`
	Requirements Rect   `json:"requirements"`
}

// Rectangle converts r to an image.Rectangle.
func (r Rect) Rectangle() image.Rectangle {
	return image.Rect(r[0], r[1], r[2], r[3])
}

// ScaledTo returns the layout mapped onto a screenshot bounds, so its
// regions are in that image's pixels.
func (l Layout) ScaledTo(b image.Rectangle) Layout {
	s := float64(b.Dy()) / float64(l.Height)
	cx, lcx := float64(b.Dx())/2, float64(l.Width)/2
	x := func(v int) int { return b.Min.X + int(cx+(float64(v)-lcx)*s) }
	y := func(v int) int { return b.Min.Y + int(float64(v)*s) }
	scale := func(r Rect) Rect {
		return Rect{x(r[0]), y(r[1]), x(r[2]), y(r[3])}
	}

	out := Layout{
		Width:        b.Dx(),
		Height:       b.Dy(),
		Name:         scale(l.Name),
		Effect:       scale(l.Effect),
		Ambrosia:     scale(l.Ambrosia),
		Requirements: scale(l.Requirements),
	}
	for _, r := range l.Tags {
		out.Tags = append(out.Tags, scale(r))
	}

	return out
}

// loadLayout reads a JSON layout file.
func loadLayout(path string) (Layout, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Layout{}, err
	}

	var l Layout
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&l); err != nil {
		return Layout{}, fmt.Errorf("%s: %w", path, err)
	}

	return l, nil
}
