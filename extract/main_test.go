package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/flat20/dandys-world-cards/extract/ocr"
)

// TestExtractCard runs the real OCR pipeline (requires tesseract) on the
// screenshots in test-screenshots and checks the resulting CSV rows.
func TestExtractCard(t *testing.T) {
	tessPath, err := ocr.FindTesseract()
	if err != nil {
		t.Skip(err)
	}
	tess := ocr.NewTesseract(tessPath)
	layout, err := loadLayout("1920x1080.json")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		file    string
		want    Card
		wantErr error
	}{
		{
			file: "treasure-hunter.png",
			want: Card{
				Name:         "Treasure Hunter",
				Rarity:       "Rare",
				Effect:       "Rarer Items are more likely to appear on floors and on Dessa's counter.",
				Ambrosia:     1000,
				Applies:      "Rest of Match",
				Category:     "Utility",
				Availability: AvailabilityNormal,
			},
		},
		{
			file: "window-shopping.png",
			want: Card{
				Name:         "Window Shopping",
				Rarity:       "Common",
				Effect:       "Common Items are less likely to appear on Dessa's counter.",
				Ambrosia:     500,
				Applies:      "Rest of Match",
				Category:     "Utility",
				Availability: AvailabilityNormal,
			},
		},
		{
			// Locked card: shows requirements instead of a price.
			file: "chain-reaction-locked.png",
			want: Card{
				Name:         "Chain Reaction",
				Rarity:       "Legendary",
				Effect:       "Every machine the team finishes makes all Toons extract 6% faster, up to 60%. Resets each floor.",
				Ambrosia:     2500, // not shown; assumed from rarity
				Applies:      "Rest of Match",
				Category:     "Toons",
				Availability: AvailabilityLimited,
				Requirements: "Unlock Card Slot 4",
			},
		},
		{
			file:    "card-list-not-a-card.png",
			wantErr: ErrNotCardDialog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			img, err := loadPNG(filepath.Join("test-screenshots", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			card, err := extractCard(tess, layout, img, tt.want.Category, tt.want.Availability)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, card %+v; want %v", err, card, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if card != tt.want {
				t.Errorf("got  %+v\nwant %+v", card, tt.want)
			}
		})
	}
}
