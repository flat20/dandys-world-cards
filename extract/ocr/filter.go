package ocr

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
)

// Filter transforms an image.
//
//	text, err := tess.Process(img, PSMSingleLine, Crop(r), Grayscale())
//
// A filter can stop the chain by returning an error.
type Filter func(image.Image) (image.Image, error)

// Chain returns a Filter that applies fs in order, stopping at the first error.
func Chain(fs ...Filter) Filter {
	return func(img image.Image) (image.Image, error) {
		for _, f := range fs {
			var err error
			if img, err = f(img); err != nil {
				return nil, err
			}
		}

		return img, nil
	}
}

// ErrBlank is returned by RequireInk when an image has (almost) no text.
var ErrBlank = errors.New("blank image")

// RequireInk stops the chain with ErrBlank if less than minRatio (0..1) of
// the pixels are dark. Tesseract tends to invent characters from a few stray
// specks, so blank crops (e.g. an unused tag slot) shouldn't reach it.
func RequireInk(minRatio float64) Filter {
	return func(img image.Image) (image.Image, error) {
		if inkRatio(img) < minRatio {
			return nil, ErrBlank
		}

		return img, nil
	}
}

// Crop keeps only region r of the image (in the image's own coordinates).
// It fails if r doesn't overlap the image, e.g. a layout for another resolution.
func Crop(r image.Rectangle) Filter {
	return func(img image.Image) (image.Image, error) {
		r := r.Intersect(img.Bounds())
		if r.Empty() {
			return nil, fmt.Errorf("crop region outside image %v", img.Bounds())
		}

		// Decoded PNGs support SubImage, which shares pixels instead of copying.
		if s, ok := img.(interface {
			SubImage(image.Rectangle) image.Image
		}); ok {
			return s.SubImage(r), nil
		}
		out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)

		return out, nil
	}
}

// Grayscale converts the image to 8-bit greyscale (luminance).
func Grayscale() Filter {
	return func(img image.Image) (image.Image, error) { return toGray(img), nil }
}

// UpscaleBilinear enlarges the image by an integer factor using bilinear
// interpolation: each output pixel is a distance-weighted blend of the 4
// nearest source pixels. Compared with repeating pixels (nearest-neighbour)
// this keeps curves and diagonals smooth, which matters for small text.
// The result is greyscale.
func UpscaleBilinear(scale int) Filter {
	return func(img image.Image) (image.Image, error) {
		src := toGray(img)
		b := src.Bounds()
		w, h := b.Dx(), b.Dy()
		dst := image.NewGray(image.Rect(0, 0, w*scale, h*scale))

		// at returns a source pixel, clamping coordinates to the image edge.
		at := func(x, y int) float64 {
			x = min(max(x, 0), w-1)
			y = min(max(y, 0), h-1)

			return float64(src.GrayAt(b.Min.X+x, b.Min.Y+y).Y)
		}
		// toSrc maps an output pixel index to its source coordinate, split into
		// the integer pixel to its top/left and the fraction (0..1) towards the next.
		// The +0.5/-0.5 aligns pixel centres rather than pixel corners.
		toSrc := func(o int) (int, float64) {
			f := (float64(o)+0.5)/float64(scale) - 0.5
			i := math.Floor(f)

			return int(i), f - i
		}

		for oy := range h * scale {
			y0, ty := toSrc(oy)
			for ox := range w * scale {
				x0, tx := toSrc(ox)
				top := at(x0, y0)*(1-tx) + at(x0+1, y0)*tx
				bottom := at(x0, y0+1)*(1-tx) + at(x0+1, y0+1)*tx
				v := top*(1-ty) + bottom*ty
				dst.SetGray(ox, oy, color.Gray{Y: uint8(v + 0.5)})
			}
		}

		return dst, nil
	}
}

// Threshold turns bright text into black ink on a white background: pixels
// with luminance >= level become black, all others white. Lower levels keep
// more of the anti-aliased edges; higher ones reject more background noise.
func Threshold(level uint8) Filter {
	return func(img image.Image) (image.Image, error) {
		src := toGray(img)
		dst := image.NewGray(src.Bounds())
		b := src.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				ink := uint8(255)
				if src.GrayAt(x, y).Y >= level {
					ink = 0
				}
				dst.SetGray(x, y, color.Gray{Y: ink})
			}
		}

		return dst, nil
	}
}

// Pad adds a white border of n pixels around the image. Tesseract misreads
// text that touches the image edge.
func Pad(n int) Filter {
	return func(img image.Image) (image.Image, error) {
		b := img.Bounds()
		dst := image.NewGray(image.Rect(0, 0, b.Dx()+2*n, b.Dy()+2*n))
		draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
		draw.Draw(dst, image.Rect(n, n, n+b.Dx(), n+b.Dy()), img, b.Min, draw.Src)

		return dst, nil
	}
}

// inkRatio returns the fraction (0..1) of dark pixels (luminance < 128) in img.
func inkRatio(img image.Image) float64 {
	g := toGray(img)
	b := g.Bounds()
	if b.Empty() {
		return 0
	}

	ink := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if g.GrayAt(x, y).Y < 128 {
				ink++
			}
		}
	}

	return float64(ink) / float64(b.Dx()*b.Dy())
}

// toGray returns img as *image.Gray, converting only if it isn't one already.
func toGray(img image.Image) *image.Gray {
	if g, ok := img.(*image.Gray); ok {
		return g
	}
	b := img.Bounds()
	g := image.NewGray(b)
	draw.Draw(g, b, img, b.Min, draw.Src)

	return g
}
