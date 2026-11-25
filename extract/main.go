// Command extract OCRs Dandy's World "Buy Cards" screenshots with tesseract
// and prints one CSV row per card in the same column layout as
// dandys_world_cards.csv.
//
// Usage:
//
//	go run . [-category Toons] [-availability normal|limited] [-layout 1920x1080.json] screenshots-dir
//
// Crop regions come from a layout JSON file for a reference resolution
// (e.g. 1920x1080.json); screenshots at other resolutions are scaled to fit.
//
// Every PNG in screenshots-dir is read; -category and -availability are copied
// into each row, since neither is shown on the card.
// Requires the `tesseract` binary on PATH.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/flat20/dandys-world-cards/extract/ocr"
)

// Options are the command-line settings.
type Options struct {
	LayoutFile   string       // -layout
	Category     string       // -category
	Availability Availability // -availability
	Dir          string       // screenshots-dir argument
}

// parseFlags reads the command line into Options. On bad usage prints
// the usage and exits.
func parseFlags() Options {
	var opts Options

	flag.StringVar(&opts.LayoutFile, "layout", "1920x1080.json", "layout JSON `file`")
	flag.StringVar(&opts.Category, "category", "", "card Category for every row, e.g. Toons")
	flag.Var(&opts.Availability, "availability", "card Availability for every row: normal or limited")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: extract [flags] screenshots-dir")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	opts.Dir = flag.Arg(0)

	return opts
}

func main() {
	opts := parseFlags()

	layout, err := loadLayout(opts.LayoutFile)
	if err != nil {
		fatal(err.Error())
	}

	tessPath, err := ocr.FindTesseract()
	if err != nil {
		fatal(err.Error())
	}
	tess := ocr.NewTesseract(tessPath)

	files, err := filepath.Glob(filepath.Join(opts.Dir, "*.png"))
	if err != nil {
		fatal(err.Error())
	}
	if len(files) == 0 {
		fatal("no PNG files in " + opts.Dir)
	}

	w := csv.NewWriter(os.Stdout)
	w.UseCRLF = true
	w.Write(header)

	for _, f := range files {
		img, err := loadPNG(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", f, err)
			continue
		}

		card, err := extractCard(tess, layout, img, opts.Category, opts.Availability)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", f, err)
			continue
		}

		w.Write(card.Record())
		w.Flush() // stream each row as soon as it's parsed
	}

	w.Flush()
	if err := w.Error(); err != nil {
		fatal(err.Error())
	}
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return png.Decode(f)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
