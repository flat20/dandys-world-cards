// Package ocr reads text from images with the tesseract command-line tool.
package ocr

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os/exec"
)

// PageSegMode is tesseract's --psm option: how it should split the image
// into text before recognising it.
type PageSegMode int

const (
	// PSMSingleBlock treats the image as one block of text that may span
	// several lines (used for the card effect).
	PSMSingleBlock PageSegMode = 6
	// PSMSingleLine treats the image as exactly one line of text (names,
	// tags, the price).
	PSMSingleLine PageSegMode = 7
)

// FindTesseract looks up the tesseract executable on PATH and returns its path.
func FindTesseract() (string, error) {
	path, err := exec.LookPath("tesseract")
	if err != nil {
		return "", fmt.Errorf("tesseract not found on PATH (brew install tesseract): %w", err)
	}

	return path, nil
}

// Tesseract OCRs images by running the tesseract command-line tool.
// Create one with NewTesseract.
type Tesseract struct {
	path string // tesseract executable
	lang string // tesseract -l language, e.g. "eng"
}

// NewTesseract returns a Tesseract that runs the executable at path and
// reads English text.
func NewTesseract(path string) *Tesseract {
	return &Tesseract{path: path, lang: "eng"}
}

// Process prepares img with filters, applied in order, and OCRs the result.
// psm tells tesseract whether to expect one line or a block of text.
//
//	text, err := tess.Process(img, PSMSingleLine, Crop(r), Grayscale(), Threshold(150))
//
// A filter returning ErrBlank means there's no text to read, and Process
// returns "" without running tesseract. Any other filter error is returned.
func (t *Tesseract) Process(img image.Image, psm PageSegMode, filters ...Filter) (string, error) {
	img, err := Chain(filters...)(img)
	if errors.Is(err, ErrBlank) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return t.run(img, psm)
}

// run runs tesseract on img and returns the recognised text.
func (t *Tesseract) run(img image.Image, psm PageSegMode) (string, error) {
	var in bytes.Buffer
	if err := png.Encode(&in, img); err != nil {
		return "", err
	}
	cmd := exec.Command(t.path, "stdin", "stdout", "--psm", fmt.Sprint(int(psm)), "-l", t.lang)
	cmd.Stdin = &in
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract: %v: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}

	return string(out), nil
}
