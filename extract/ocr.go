package main

import (
	"errors"
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/flat20/dandys-world-cards/extract/ocr"
)

// ErrNotCardDialog is returned by extractCard for screenshots that don't show
// a card's buy dialog (no rarity tag or Ambrosia price was found).
var ErrNotCardDialog = errors.New("not a card dialog")

var rarities = map[string]bool{
	"Common":    true,
	"Uncommon":  true,
	"Rare":      true,
	"Epic":      true,
	"Legendary": true,
	"Mythic":    true,
}

// defaultAmbrosia is the assumed price by rarity, used when a card's price
// isn't shown (e.g. a locked card) or can't be read.
var defaultAmbrosia = map[string]int{
	"Common":    500,
	"Rare":      1000,
	"Legendary": 2500,
}

// textPrep returns the filters that turn region r of a screenshot into text.
// scale is how much to enlarge it, and level the threshold (0-255) that separates the text from the
// background.
//
// The game draws white/light-grey text with a dark outline on a dark,
// textured background, which tesseract reads poorly as-is. Each region is
// turned into clean black-on-white text first: enlarged (tesseract wants
// glyphs ~30px+ tall; the name font is ~12px at 1080p), thresholded so only
// the bright text survives, and given a white margin.
// Crops with (almost) no ink are blank (e.g. an unused tag slot); measured
// on the screenshots, empty crops have 0 ink and real text at least ~1%.
func textPrep(r image.Rectangle, scale float64, level uint8) ocr.Filter {
	return ocr.Chain(
		ocr.Crop(r),
		ocr.Grayscale(),
		ocr.UpscaleBilinear(max(int(scale+0.5), 1)),
		ocr.Threshold(level),
		ocr.Pad(10),
		ocr.RequireInk(0.005),
	)
}

// extractCard OCRs a screenshot, using the crop regions in l, and returns
// the card it shows.
func extractCard(tess *ocr.Tesseract, l Layout, img image.Image, category string, availability Availability) (Card, error) {
	l = l.ScaledTo(img.Bounds())
	// The upscale factors below are tuned for 1080p; zoom adjusts them for
	// other screen heights.
	zoom := 1080 / float64(l.Height)

	name, err := tess.Process(img, ocr.PSMSingleLine, textPrep(l.Name.Rectangle(), 4*zoom, 150))
	if err != nil {
		return Card{}, err
	}

	effect, err := tess.Process(img, ocr.PSMSingleBlock, textPrep(l.Effect.Rectangle(), 2*zoom, 170))
	if err != nil {
		return Card{}, err
	}

	amb, err := tess.Process(img, ocr.PSMSingleLine, textPrep(l.Ambrosia.Rectangle(), 2*zoom, 170))
	if err != nil {
		return Card{}, err
	}

	var tags []string
	for _, r := range l.Tags {
		t, err := tess.Process(img, ocr.PSMSingleLine, textPrep(r.Rectangle(), 2*zoom, 150))
		if err != nil {
			return Card{}, err
		}
		if t = cleanTag(t); t != "" {
			tags = append(tags, t)
		}
	}

	var rarity, applies []string
	for _, t := range tags {
		if rarities[t] {
			rarity = append(rarity, t)
		} else {
			applies = append(applies, t)
		}
	}

	// A card the player can't buy yet shows its requirements (e.g. "Unlock
	// Card Slot 4 (0/1)") where the price would be, so only read them when
	// there's no valid price.
	price, hasPrice := parsePrice(amb)
	var requirement string
	if !hasPrice {
		req, err := tess.Process(img, ocr.PSMSingleLine, textPrep(l.Requirements.Rectangle(), 2*zoom, 170))
		if err != nil {
			return Card{}, err
		}
		requirement = cleanRequirement(req)
	}

	// If we haven't been able to read required details, it's invalid.
	if len(rarity) == 0 || (!hasPrice && requirement == "") {
		return Card{}, fmt.Errorf("%w (rarity %q, ambrosia %q), skipped", ErrNotCardDialog, rarity, cleanLine(amb))
	}

	if !hasPrice && len(rarity) == 1 {
		price = defaultAmbrosia[rarity[0]]
	}

	return Card{
		Name:         cleanLine(name),
		Rarity:       strings.Join(rarity, ", "),
		Effect:       cleanLine(effect),
		Ambrosia:     price,
		Applies:      strings.Join(applies, ", "),
		Category:     category,
		Availability: availability,
		Requirements: requirement,
	}, nil
}

// priceRe matches "1,000 Ambrosia".
var priceRe = regexp.MustCompile(`(?i)(\d[\d,.]*)\s*Ambrosia`)

// parsePrice returns the Ambrosia price in s, e.g. "1,000 Ambrosia" -> 1000.
func parsePrice(s string) (int, bool) {
	m := priceRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(digitsOnly(m[1]))
	return n, err == nil && n > 0
}

// progressRe matches the progress counter for requirements,
// e.g. " (0/1)" or " (12%/75%)".
var progressRe = regexp.MustCompile(`\s*\(\s*[\d.]+%?\s*/\s*[\d.]+%?\s*\)?\s*$`)

// cleanRequirement converts "Unlock Card Slot 4 (0/1)" -> "Unlock Card Slot 4".
func cleanRequirement(s string) string {
	return strings.TrimSpace(progressRe.ReplaceAllString(cleanLine(s), ""))
}

var spaces = regexp.MustCompile(`\s+`)

// cleanLine normalises OCR output to a single line. Removing typical noise.
func cleanLine(s string) string {
	s = strings.NewReplacer("’", "'", "‘", "'", "“", `"`, "”", `"`, "|", "I").Replace(s)
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

// cleanTag drops OCR noise from empty tag slots and trims punctuation.
func cleanTag(s string) string {
	s = cleanLine(s)
	s = strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters++
		}
	}
	if letters < 2 {
		return ""
	}
	return s
}

// digitsOnly keeps only 0-9, e.g. "1,000 Ambrosia" -> "1000".
func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
