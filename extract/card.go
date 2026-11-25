package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Card is one card as read from a screenshot.
type Card struct {
	Name         string
	Rarity       string // "Rare", "Common", "Legendary"
	Effect       string
	Ambrosia     int          // price; assumed from rarity if not shown, 0 if unknown
	Applies      string       // duration tag, e.g. "Rest of Match"; may be empty
	Category     string       // not shown on screenshots; set with -category, e.g. "Toons"
	Availability Availability // not shown on screenshots; set with -availability
	Requirements string       // only shown on locked cards, e.g. "Unlock Card Slot 4"
}

// header is the CSV header; it must stay in the same order as Card.Record.
var header = []string{"Name", "Rarity", "Effect", "Ambrosia", "Applies", "Category", "Availability", "Requirements"}

// Record returns the card as a CSV row in header order.
func (c Card) Record() []string {
	ambrosia := ""
	if c.Ambrosia > 0 {
		ambrosia = strconv.Itoa(c.Ambrosia)
	}
	return []string{
		c.Name,
		c.Rarity,
		c.Effect,
		ambrosia,
		c.Applies,
		c.Category,
		c.Availability.String(),
		c.Requirements,
	}
}

// Availability is whether a card is always in the shop or only for a limited
// time. The zero value means unknown and is written as an empty CSV field.
type Availability int

const (
	AvailabilityUnknown Availability = iota
	AvailabilityNormal
	AvailabilityLimited // only purchasable for a short time
)

func (a Availability) String() string {
	switch a {
	case AvailabilityNormal:
		return "Normal"
	case AvailabilityLimited:
		return "Limited"
	}
	return ""
}

// Set parses "normal" or "limited" (any case), so *Availability can be used
// as a flag.Value.
func (a *Availability) Set(s string) error {
	switch strings.ToLower(s) {
	case "normal":
		*a = AvailabilityNormal
	case "limited":
		*a = AvailabilityLimited
	default:
		return fmt.Errorf("must be normal or limited, not %q", s)
	}
	return nil
}
