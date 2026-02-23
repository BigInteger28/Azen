package io

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/azen-engine/pkg/cards"
	"github.com/azen-engine/pkg/game"
)

// Reader leest interactieve invoer van stdin.
type Reader struct {
	scanner *bufio.Scanner
}

func NewReader() *Reader {
	return &Reader{scanner: bufio.NewScanner(os.Stdin)}
}

func (r *Reader) ReadLine(prompt string) string {
	fmt.Print(prompt)
	if r.scanner.Scan() {
		return strings.TrimSpace(r.scanner.Text())
	}
	return ""
}

func (r *Reader) ReadInt(prompt string) (int, error) {
	s := r.ReadLine(prompt)
	return strconv.Atoi(strings.TrimSpace(s))
}

func (r *Reader) ReadCards(prompt string) ([]cards.Card, error) {
	s := r.ReadLine(prompt)
	return cards.ParseCards(s)
}

func (r *Reader) ReadYesNo(prompt string) bool {
	s := strings.ToLower(r.ReadLine(prompt + " (j/n): "))
	return s == "j" || s == "y" || s == "ja" || s == "yes"
}

func (r *Reader) ReadMove(playerID int, prompt string) (game.Move, error) {
	if prompt == "" {
		prompt = fmt.Sprintf("Speler %d zet: ", playerID+1)
	}
	s := r.ReadLine(prompt)
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "pass" || lower == "p" {
		return game.PassMove(playerID), nil
	}
	cc, err := cards.ParseCards(s)
	if err != nil {
		return game.Move{}, err
	}
	return game.Move{PlayerID: playerID, Cards: cc}, nil
}

// ---- Display-functies ----

func PrintHeader(title string) {
	border := strings.Repeat("═", len(title)+4)
	fmt.Printf("\n╔%s╗\n║  %s  ║\n╚%s╝\n\n", border, title, border)
}

func PrintSubHeader(title string) {
	fmt.Printf("\n─── %s ───\n", title)
}

func PrintCards(hand *cards.Hand) {
	hand.Sort()
	fmt.Printf("  Hand: %s\n", hand.String())
}

func PrintHelp() {
	fmt.Print(`
Kaartnotatie (één teken per kaart):
  0=Joker  1=Aas  2-9=cijfers  X=10  J=Boer  Q=Dame  K=Heer

Invoerformaten (alle drie werken):
  KK3XJ       aaneengesloten
  K,K,3,X,J   komma-gescheiden
  K K 3 X J   spatie-gescheiden

Commando's tijdens jouw beurt:
  pass / p   pas
  hint       laat motorsuggestie opnieuw zien
  hand       laat jouw hand opnieuw zien
  status     laat spelstatus zien
  moves      laat alle legale zetten zien
  quit       stop het spel

`)
}

func PrintMoveOptions(moves []game.Move, max int) {
	if max > len(moves) {
		max = len(moves)
	}
	fmt.Printf("Mogelijke zetten (%d totaal):\n", len(moves))
	for i := 0; i < max; i++ {
		fmt.Printf("  %2d. %s\n", i+1, FormatMove(moves[i]))
	}
	if len(moves) > max {
		fmt.Printf("  ... en nog %d meer\n", len(moves)-max)
	}
}

func FormatMove(m game.Move) string {
	if m.IsPass {
		return "PASS"
	}
	sorted := make([]cards.Card, len(m.Cards))
	copy(sorted, m.Cards)
	// Sorteer op rank zodat "25" en "52" altijd hetzelfde tonen
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rank < sorted[j].Rank
	})
	parts := make([]string, len(sorted))
	for i, c := range sorted {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}

func FormatScore(score float64) string {
	return fmt.Sprintf("%.1f%%", score*100)
}
