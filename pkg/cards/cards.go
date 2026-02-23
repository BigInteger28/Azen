package cards

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// Rank represents a card rank. Internal ordering for game logic.
type Rank int

const (
	RankThree Rank = 3
	RankFour  Rank = 4
	RankFive  Rank = 5
	RankSix   Rank = 6
	RankSeven Rank = 7
	RankEight Rank = 8
	RankNine  Rank = 9
	RankTen   Rank = 10
	RankJack  Rank = 11
	RankQueen Rank = 12
	RankKing  Rank = 13
	RankTwo   Rank = 14 // Wildcard
	RankJoker Rank = 15 // Wildcard (same as 2)
	RankAce   Rank = 16 // Played on anything, resets round
)

type Suit int

const (
	SuitHearts   Suit = 0
	SuitDiamonds Suit = 1
	SuitClubs    Suit = 2
	SuitSpades   Suit = 3
	SuitJoker1   Suit = 4
	SuitJoker2   Suit = 5
)

type Card struct {
	Rank Rank
	Suit Suit
}

func (c Card) IsWild() bool    { return c.Rank == RankTwo || c.Rank == RankJoker }
func (c Card) IsAce() bool     { return c.Rank == RankAce }
func (c Card) IsSpecial() bool { return c.IsWild() || c.IsAce() }

// String returns the single-character representation of the card.
// Suits are not shown because they don't affect gameplay.
// 0=Joker 1=Aas 2-9 X=10 J Q K
func (c Card) String() string {
	return c.RankStr()
}

func (c Card) RankStr() string {
	switch c.Rank {
	case RankAce:
		return "1"
	case RankTwo:
		return "2"
	case RankThree:
		return "3"
	case RankFour:
		return "4"
	case RankFive:
		return "5"
	case RankSix:
		return "6"
	case RankSeven:
		return "7"
	case RankEight:
		return "8"
	case RankNine:
		return "9"
	case RankTen:
		return "X"
	case RankJack:
		return "J"
	case RankQueen:
		return "Q"
	case RankKing:
		return "K"
	case RankJoker:
		return "0"
	}
	return "?"
}

// ParseCard parses een enkel teken naar een kaart.
// Geldig: 0 (joker) 1 (aas) 2 3 4 5 6 7 8 9 X (10) J Q K
// Suit wordt intern toegewezen maar doet er niet toe voor de spellogica.
func ParseCard(s string) (Card, error) {
	s = strings.TrimSpace(s)
	if len(s) != 1 {
		return Card{}, fmt.Errorf("ongeldige kaart: %q (verwacht één teken: 0 1 2..9 X J Q K)", s)
	}
	switch strings.ToUpper(s) {
	case "0":
		return Card{RankJoker, SuitJoker1}, nil
	case "1":
		return Card{RankAce, SuitHearts}, nil
	case "2":
		return Card{RankTwo, SuitHearts}, nil
	case "3":
		return Card{RankThree, SuitHearts}, nil
	case "4":
		return Card{RankFour, SuitHearts}, nil
	case "5":
		return Card{RankFive, SuitHearts}, nil
	case "6":
		return Card{RankSix, SuitHearts}, nil
	case "7":
		return Card{RankSeven, SuitHearts}, nil
	case "8":
		return Card{RankEight, SuitHearts}, nil
	case "9":
		return Card{RankNine, SuitHearts}, nil
	case "X":
		return Card{RankTen, SuitHearts}, nil
	case "J":
		return Card{RankJack, SuitHearts}, nil
	case "Q":
		return Card{RankQueen, SuitHearts}, nil
	case "K":
		return Card{RankKing, SuitHearts}, nil
	}
	return Card{}, fmt.Errorf("ongeldige kaart: %q (gebruik: 0 1 2..9 X J Q K)", s)
}

// ParseCards parst kaarten gescheiden door komma's of spaties: "K,K,Q" of "K K Q" of "KKQ"
func ParseCards(s string) ([]Card, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	// Ondersteun komma's en spaties als scheidingsteken; ook aaneengesloten tekens
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	result := make([]Card, 0, len(parts))
	for _, p := range parts {
		// Elke "token" kan meerdere aaneengesloten tekens bevatten, bv. "KKQ"
		for _, ch := range p {
			c, err := ParseCard(string(ch))
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
	}
	return result, nil
}

// CardsToString formats a slice of cards
func CardsToString(cc []Card) string {
	parts := make([]string, len(cc))
	for i, c := range cc {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}

// ---- Hand ----

type Hand struct {
	Cards []Card
}

func NewHand(cc []Card) *Hand {
	h := &Hand{Cards: make([]Card, len(cc))}
	copy(h.Cards, cc)
	return h
}

func (h *Hand) Count() int    { return len(h.Cards) }
func (h *Hand) IsEmpty() bool { return len(h.Cards) == 0 }

// Remove verwijdert kaarten op basis van rank (suit wordt genegeerd).
// Als een kaart niet gevonden wordt, wordt een rank-0 placeholder gebruikt
// (voor onbekende tegenstander-handen in speel-modus).
func (h *Hand) Remove(cc []Card) error {
	rem := make([]Card, len(h.Cards))
	copy(rem, h.Cards)
	for _, c := range cc {
		found := false
		// Eerst zoeken op echte rank
		for i, hc := range rem {
			if hc.Rank == c.Rank {
				rem = append(rem[:i], rem[i+1:]...)
				found = true
				break
			}
		}
		// Fallback: rank-0 placeholder (onbekende hand)
		if !found {
			for i, hc := range rem {
				if hc.Rank == 0 {
					rem = append(rem[:i], rem[i+1:]...)
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("kaart %s niet in hand", c)
		}
	}
	h.Cards = rem
	return nil
}

// Has controleert op rank (suit wordt genegeerd).
func (h *Hand) Has(c Card) bool {
	for _, hc := range h.Cards {
		if hc.Rank == c.Rank {
			return true
		}
	}
	return false
}

func (h *Hand) CountWilds() int {
	n := 0
	for _, c := range h.Cards {
		if c.IsWild() {
			n++
		}
	}
	return n
}

func (h *Hand) CountAces() int {
	n := 0
	for _, c := range h.Cards {
		if c.IsAce() {
			n++
		}
	}
	return n
}

func (h *Hand) CountRank(r Rank) int {
	n := 0
	for _, c := range h.Cards {
		if c.Rank == r {
			n++
		}
	}
	return n
}

func (h *Hand) GetByRank(r Rank) []Card {
	var res []Card
	for _, c := range h.Cards {
		if c.Rank == r {
			res = append(res, c)
		}
	}
	return res
}

func (h *Hand) Sort() {
	sort.Slice(h.Cards, func(i, j int) bool {
		if h.Cards[i].Rank != h.Cards[j].Rank {
			return h.Cards[i].Rank < h.Cards[j].Rank
		}
		return h.Cards[i].Suit < h.Cards[j].Suit
	})
}

func (h *Hand) String() string {
	h.Sort()
	return CardsToString(h.Cards)
}

func (h *Hand) Clone() *Hand {
	return NewHand(h.Cards)
}

// ---- Deck ----

type Deck struct {
	Cards []Card
}

// NewDeck creates a 54-card deck (52 + 2 jokers)
func NewDeck() *Deck {
	d := &Deck{}
	suits := []Suit{SuitHearts, SuitDiamonds, SuitClubs, SuitSpades}
	ranks := []Rank{
		RankAce, RankTwo, RankThree, RankFour, RankFive, RankSix, RankSeven,
		RankEight, RankNine, RankTen, RankJack, RankQueen, RankKing,
	}
	for _, s := range suits {
		for _, r := range ranks {
			d.Cards = append(d.Cards, Card{r, s})
		}
	}
	d.Cards = append(d.Cards, Card{RankJoker, SuitJoker1})
	d.Cards = append(d.Cards, Card{RankJoker, SuitJoker2})
	return d
}

func NewMultiDeck(n int) *Deck {
	d := &Deck{}
	for i := 0; i < n; i++ {
		single := NewDeck()
		d.Cards = append(d.Cards, single.Cards...)
	}
	return d
}

func (d *Deck) Shuffle(rng *rand.Rand) {
	rng.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

func (d *Deck) Deal(numPlayers, cardsPerPlayer int) ([]*Hand, []Card) {
	hands := make([]*Hand, numPlayers)
	for i := range hands {
		hands[i] = &Hand{}
	}
	idx := 0
	for c := 0; c < cardsPerPlayer; c++ {
		for p := 0; p < numPlayers; p++ {
			if idx < len(d.Cards) {
				hands[p].Cards = append(hands[p].Cards, d.Cards[idx])
				idx++
			}
		}
	}
	return hands, d.Cards[idx:]
}

func NormalRanks() []Rank {
	return []Rank{
		RankThree, RankFour, RankFive, RankSix, RankSeven,
		RankEight, RankNine, RankTen, RankJack, RankQueen, RankKing,
	}
}
