// azen-termux.go
// Standalone single-file versie van de AZEN engine voor Termux/Android.
// Bevat alle code in één bestand zonder externe afhankelijkheden.
//
// Compileer: go build azen-termux.go
// Starten:   go run azen-termux.go

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// CARDS
// ═══════════════════════════════════════════════════════════════

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
	RankAce   Rank = 14 // Hoogste naturelle kaart (boven Koning)
	RankTwo   Rank = 15 // Wildcard (vervangt elke kaart)
	RankJoker Rank = 16 // Reset-kaart (verslaat alles, opent nieuwe ronde)
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

func (c Card) IsWild() bool    { return c.Rank == RankTwo }    // alleen de 2 is wildcard
func (c Card) IsReset() bool   { return c.Rank == RankJoker }  // joker reset de ronde
func (c Card) IsAce() bool     { return c.Rank == RankAce }    // naturelle hoge kaart
func (c Card) IsSpecial() bool { return c.IsWild() || c.IsReset() }

func (c Card) String() string { return c.RankStr() }

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

func ParseCards(s string) ([]Card, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	result := make([]Card, 0, len(parts))
	for _, p := range parts {
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

func (h *Hand) Remove(cc []Card) error {
	rem := make([]Card, len(h.Cards))
	copy(rem, h.Cards)
	for _, c := range cc {
		found := false
		for i, hc := range rem {
			if hc.Rank == c.Rank {
				rem = append(rem[:i], rem[i+1:]...)
				found = true
				break
			}
		}
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

func (h *Hand) CountResets() int {
	n := 0
	for _, c := range h.Cards {
		if c.IsReset() {
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

func (h *Hand) Clone() *Hand { return NewHand(h.Cards) }

// ---- Deck ----

type Deck struct {
	Cards []Card
}

func NewDeck() *Deck {
	d := &Deck{}
	suits := []Suit{SuitHearts, SuitDiamonds, SuitClubs, SuitSpades}
	ranks := []Rank{
		RankThree, RankFour, RankFive, RankSix, RankSeven,
		RankEight, RankNine, RankTen, RankJack, RankQueen, RankKing,
		RankAce, RankTwo, // Aas = hoogste naturelle; Twee = wildcard
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
		RankEight, RankNine, RankTen, RankJack, RankQueen, RankKing, RankAce,
	}
}

// ═══════════════════════════════════════════════════════════════
// GAME
// ═══════════════════════════════════════════════════════════════

type Move struct {
	PlayerID int
	Cards    []Card
	IsPass   bool
}

func PassMove(playerID int) Move {
	return Move{PlayerID: playerID, IsPass: true}
}

func (m Move) String() string {
	if m.IsPass {
		return fmt.Sprintf("P%d: PASS", m.PlayerID)
	}
	return fmt.Sprintf("P%d: %s", m.PlayerID, CardsToString(m.Cards))
}

func (m Move) ContainsReset() bool {
	for _, c := range m.Cards {
		if c.IsReset() {
			return true
		}
	}
	return false
}

func (m Move) EffectiveRank(tableRank Rank) Rank {
	best := Rank(0)
	for _, c := range m.Cards {
		if !c.IsSpecial() && c.Rank > best {
			best = c.Rank
		}
	}
	if best == 0 {
		return tableRank
	}
	return best
}

type RoundState struct {
	Count        int
	TableRank    Rank
	IsOpen       bool
	LastPlayerID int
	ConsecPasses int
}

type GameState struct {
	NumPlayers  int
	Hands       []*Hand
	CurrentTurn int
	Round       RoundState
	Played      []Card
	History     []Move
	GameOver    bool
	Winner      int
	Ranking     []int
	Finished    []bool
	DeadCards   []Card
}

func NewGame(numPlayers int, rng *rand.Rand, startPlayer int) *GameState {
	numDecks := 1
	if numPlayers == 4 {
		numDecks = 2
	}
	var deck *Deck
	if numDecks == 1 {
		deck = NewDeck()
	} else {
		deck = NewMultiDeck(numDecks)
	}
	deck.Shuffle(rng)
	hands, remaining := deck.Deal(numPlayers, 18)
	return &GameState{
		NumPlayers:  numPlayers,
		Hands:       hands,
		CurrentTurn: startPlayer,
		Round:       RoundState{IsOpen: true},
		Winner:      -1,
		Finished:    make([]bool, numPlayers),
		DeadCards:   remaining,
	}
}

func NewGameWithHands(hands []*Hand, dead []Card, startPlayer int) *GameState {
	return &GameState{
		NumPlayers:  len(hands),
		Hands:       hands,
		CurrentTurn: startPlayer,
		Round:       RoundState{IsOpen: true},
		Winner:      -1,
		Finished:    make([]bool, len(hands)),
		DeadCards:   dead,
	}
}

func (gs *GameState) Clone() *GameState {
	n := &GameState{
		NumPlayers:  gs.NumPlayers,
		CurrentTurn: gs.CurrentTurn,
		Round:       gs.Round,
		GameOver:    gs.GameOver,
		Winner:      gs.Winner,
	}
	n.Hands = make([]*Hand, len(gs.Hands))
	for i, h := range gs.Hands {
		n.Hands[i] = h.Clone()
	}
	n.Played = make([]Card, len(gs.Played))
	copy(n.Played, gs.Played)
	n.History = make([]Move, len(gs.History))
	copy(n.History, gs.History)
	n.DeadCards = make([]Card, len(gs.DeadCards))
	copy(n.DeadCards, gs.DeadCards)
	n.Finished = make([]bool, len(gs.Finished))
	copy(n.Finished, gs.Finished)
	n.Ranking = make([]int, len(gs.Ranking))
	copy(n.Ranking, gs.Ranking)
	return n
}

func (gs *GameState) activePlayerCount() int {
	count := 0
	for _, f := range gs.Finished {
		if !f {
			count++
		}
	}
	return count
}

func (gs *GameState) nextActiveTurn(fromPID int) int {
	for i := 1; i <= gs.NumPlayers; i++ {
		next := (fromPID + i) % gs.NumPlayers
		if !gs.Finished[next] {
			return next
		}
	}
	return fromPID
}

func (gs *GameState) passThreshold() int {
	active := gs.activePlayerCount()
	if gs.Finished[gs.Round.LastPlayerID] {
		return active
	}
	return active - 1
}

func (gs *GameState) PlayerRank(pid int) int {
	for i, p := range gs.Ranking {
		if p == pid {
			return i
		}
	}
	return -1
}

func (gs *GameState) finishPlayer(pid int) bool {
	gs.Finished[pid] = true
	gs.Ranking = append(gs.Ranking, pid)
	if gs.Winner == -1 {
		gs.Winner = pid
	}
	if gs.activePlayerCount() <= 1 {
		for i, f := range gs.Finished {
			if !f {
				gs.Ranking = append(gs.Ranking, i)
				gs.Finished[i] = true
				break
			}
		}
		gs.GameOver = true
		return true
	}
	return false
}

func (gs *GameState) ValidateMove(m Move) error {
	if gs.GameOver {
		return fmt.Errorf("game is over")
	}
	if m.PlayerID != gs.CurrentTurn {
		return fmt.Errorf("not player %d's turn (current: %d)", m.PlayerID, gs.CurrentTurn)
	}
	if m.IsPass {
		return nil
	}
	if len(m.Cards) == 0 {
		return fmt.Errorf("must play at least one card (or pass)")
	}
	hand := gs.Hands[m.PlayerID]
	tmpHand := hand.Clone()
	if err := tmpHand.Remove(m.Cards); err != nil {
		return fmt.Errorf("cards not in hand: %v", err)
	}
	if gs.Round.IsOpen {
		return gs.validateOpenPlay(m)
	}
	return gs.validateResponsePlay(m)
}

func (gs *GameState) validateOpenPlay(m Move) error {
	hasReset, hasNormal, normalRank, err := classifyCards(m.Cards)
	if err != nil {
		return err
	}
	if hasReset && hasNormal {
		return fmt.Errorf("een joker (0) mag enkel samen met een 2 of andere joker gespeeld worden, niet met normale kaarten")
	}
	_ = normalRank
	return nil
}

func (gs *GameState) validateResponsePlay(m Move) error {
	hasReset, hasNormal, normalRank, err := classifyCards(m.Cards)
	if err != nil {
		return err
	}
	if hasReset && hasNormal {
		return fmt.Errorf("een joker (0) mag enkel samen met een 2 of andere joker gespeeld worden, niet met normale kaarten")
	}
	if len(m.Cards) != gs.Round.Count {
		return fmt.Errorf("moet exact %d kaart(en) spelen (gespeeld: %d)", gs.Round.Count, len(m.Cards))
	}
	if normalRank != 0 && normalRank <= gs.Round.TableRank {
		return fmt.Errorf("rank %d verslaat tafel-rank %d niet", normalRank, gs.Round.TableRank)
	}
	return nil
}

func classifyCards(cc []Card) (hasReset bool, hasNormal bool, normalRank Rank, err error) {
	for _, c := range cc {
		if c.IsReset() {
			hasReset = true
		} else if c.IsWild() {
			// wildcards zijn neutraal
		} else {
			// Normale kaarten: 3..K en Aas (1 is nu de hoogste naturelle rank)
			hasNormal = true
			if normalRank == 0 {
				normalRank = c.Rank
			} else if c.Rank != normalRank {
				err = fmt.Errorf("alle normale kaarten moeten dezelfde rank hebben")
				return
			}
		}
	}
	return
}

func (gs *GameState) ApplyMove(m Move) {
	gs.History = append(gs.History, m)
	pid := m.PlayerID

	if m.IsPass {
		gs.Round.ConsecPasses++
		if gs.Round.ConsecPasses >= gs.passThreshold() {
			lastPID := gs.Round.LastPlayerID
			gs.Round = RoundState{IsOpen: true, LastPlayerID: lastPID}
			if gs.Finished[lastPID] {
				gs.CurrentTurn = gs.nextActiveTurn(lastPID)
			} else {
				gs.CurrentTurn = lastPID
			}
			return
		}
		gs.CurrentTurn = gs.nextActiveTurn(pid)
		return
	}

	gs.Hands[pid].Remove(m.Cards)
	gs.Played = append(gs.Played, m.Cards...)

	if gs.Hands[pid].IsEmpty() {
		if gs.finishPlayer(pid) {
			return
		}
		if m.ContainsReset() {
			gs.Round = RoundState{IsOpen: true, LastPlayerID: pid}
			gs.CurrentTurn = gs.nextActiveTurn(pid)
		} else {
			gs.Round = RoundState{
				Count:        len(m.Cards),
				TableRank:    m.EffectiveRank(gs.Round.TableRank),
				IsOpen:       false,
				LastPlayerID: pid,
				ConsecPasses: 0,
			}
			gs.CurrentTurn = gs.nextActiveTurn(pid)
		}
		return
	}

	if m.ContainsReset() {
		gs.Round = RoundState{IsOpen: true, LastPlayerID: pid}
		gs.CurrentTurn = pid
		return
	}

	effectiveRank := m.EffectiveRank(gs.Round.TableRank)
	if gs.Round.IsOpen {
		gs.Round = RoundState{
			Count:        len(m.Cards),
			TableRank:    effectiveRank,
			IsOpen:       false,
			LastPlayerID: pid,
			ConsecPasses: 0,
		}
	} else {
		gs.Round.TableRank = effectiveRank
		gs.Round.LastPlayerID = pid
		gs.Round.ConsecPasses = 0
	}
	gs.CurrentTurn = gs.nextActiveTurn(pid)
}

func (gs *GameState) GetLegalMoves() []Move {
	if gs.GameOver {
		return nil
	}
	pid := gs.CurrentTurn
	hand := gs.Hands[pid]
	moves := []Move{PassMove(pid)}
	if gs.Round.IsOpen {
		moves = append(moves, genOpenMoves(pid, hand)...)
	} else {
		moves = append(moves, genResponseMoves(pid, hand, gs.Round)...)
	}
	return moves
}

func genOpenMoves(pid int, hand *Hand) []Move {
	var moves []Move
	byRank := map[Rank][]Card{}
	for _, c := range hand.Cards {
		byRank[c.Rank] = append(byRank[c.Rank], c)
	}
	wilds := gatherWilds(hand)
	resets := gatherResets(hand)

	for _, rank := range NormalRanks() {
		normals := byRank[rank]
		if len(normals) == 0 {
			continue
		}
		maxTotal := imin(len(normals)+len(wilds), 6)
		for total := 1; total <= maxTotal; total++ {
			for numNorm := imax(1, total-len(wilds)); numNorm <= imin(len(normals), total); numNorm++ {
				numWild := total - numNorm
				if numWild < 0 || numWild > len(wilds) {
					continue
				}
				nCombos := combos(normals, numNorm)
				if numWild == 0 {
					for _, nc := range nCombos {
						moves = append(moves, Move{PlayerID: pid, Cards: nc})
					}
				} else {
					wCombos := combos(wilds, numWild)
					for _, nc := range nCombos {
						for _, wc := range wCombos {
							merged := append(append([]Card{}, nc...), wc...)
							moves = append(moves, Move{PlayerID: pid, Cards: merged})
						}
					}
				}
			}
		}
	}

	for total := 1; total <= imin(len(wilds), 6); total++ {
		for _, wc := range combos(wilds, total) {
			moves = append(moves, Move{PlayerID: pid, Cards: wc})
		}
	}

	moves = append(moves, genResetMoves(pid, resets, wilds)...)
	return dedup(moves)
}

func genResponseMoves(pid int, hand *Hand, round RoundState) []Move {
	var moves []Move
	need := round.Count
	tableRank := round.TableRank
	wilds := gatherWilds(hand)
	resets := gatherResets(hand)

	for _, rank := range NormalRanks() {
		if rank <= tableRank {
			continue
		}
		normals := hand.GetByRank(rank)
		if len(normals) == 0 {
			continue
		}
		for numNorm := imax(1, need-len(wilds)); numNorm <= imin(len(normals), need); numNorm++ {
			numWild := need - numNorm
			if numWild < 0 || numWild > len(wilds) {
				continue
			}
			nCombos := combos(normals, numNorm)
			if numWild == 0 {
				for _, nc := range nCombos {
					moves = append(moves, Move{PlayerID: pid, Cards: nc})
				}
			} else {
				wCombos := combos(wilds, numWild)
				for _, nc := range nCombos {
					for _, wc := range wCombos {
						merged := append(append([]Card{}, nc...), wc...)
						moves = append(moves, Move{PlayerID: pid, Cards: merged})
					}
				}
			}
		}
	}

	if len(wilds) >= need {
		for _, wc := range combos(wilds, need) {
			moves = append(moves, Move{PlayerID: pid, Cards: wc})
		}
	}

	moves = append(moves, genResetResponseMoves(pid, resets, wilds, need)...)
	return dedup(moves)
}

func genResetMoves(pid int, resets, wilds []Card) []Move {
	var moves []Move
	for numReset := 1; numReset <= len(resets); numReset++ {
		maxW := imin(len(wilds), 6-numReset)
		rCombos := combos(resets, numReset)
		for numWild := 0; numWild <= maxW; numWild++ {
			if numWild == 0 {
				for _, rc := range rCombos {
					moves = append(moves, Move{PlayerID: pid, Cards: rc})
				}
			} else {
				wCombos := combos(wilds, numWild)
				for _, rc := range rCombos {
					for _, wc := range wCombos {
						merged := append(append([]Card{}, rc...), wc...)
						moves = append(moves, Move{PlayerID: pid, Cards: merged})
					}
				}
			}
		}
	}
	return moves
}

func genResetResponseMoves(pid int, resets, wilds []Card, need int) []Move {
	var moves []Move
	for numReset := 1; numReset <= imin(len(resets), need); numReset++ {
		numWild := need - numReset
		if numWild > len(wilds) {
			continue
		}
		rCombos := combos(resets, numReset)
		if numWild == 0 {
			for _, rc := range rCombos {
				moves = append(moves, Move{PlayerID: pid, Cards: rc})
			}
		} else {
			wCombos := combos(wilds, numWild)
			for _, rc := range rCombos {
				for _, wc := range wCombos {
					merged := append(append([]Card{}, rc...), wc...)
					moves = append(moves, Move{PlayerID: pid, Cards: merged})
				}
			}
		}
	}
	return moves
}

func gatherSpecials(hand *Hand) []Card {
	var sp []Card
	for _, c := range hand.Cards {
		if c.IsSpecial() {
			sp = append(sp, c)
		}
	}
	return sp
}

func gatherWilds(hand *Hand) []Card {
	var wilds []Card
	for _, c := range hand.Cards {
		if c.IsWild() {
			wilds = append(wilds, c)
		}
	}
	return wilds
}

func gatherResets(hand *Hand) []Card {
	var resets []Card
	for _, c := range hand.Cards {
		if c.IsReset() {
			resets = append(resets, c)
		}
	}
	return resets
}

func combos(arr []Card, k int) [][]Card {
	if k <= 0 || k > len(arr) {
		if k == 0 {
			return [][]Card{{}}
		}
		return nil
	}
	var result [][]Card
	var helper func(start int, curr []Card)
	helper = func(start int, curr []Card) {
		if len(curr) == k {
			c := make([]Card, k)
			copy(c, curr)
			result = append(result, c)
			return
		}
		remaining := k - len(curr)
		for i := start; i <= len(arr)-remaining; i++ {
			helper(i+1, append(curr, arr[i]))
		}
	}
	helper(0, nil)
	return result
}

func dedup(moves []Move) []Move {
	seen := map[string]bool{}
	var result []Move
	for _, m := range moves {
		key := moveKey(m)
		if !seen[key] {
			seen[key] = true
			result = append(result, m)
		}
	}
	return result
}

func moveKey(m Move) string {
	if m.IsPass {
		return "PASS"
	}
	sorted := make([]Card, len(m.Cards))
	copy(sorted, m.Cards)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Rank > sorted[j].Rank || (sorted[i].Rank == sorted[j].Rank && sorted[i].Suit > sorted[j].Suit) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	parts := make([]string, len(sorted))
	for i, c := range sorted {
		parts[i] = c.String()
	}
	return strings.Join(parts, "|")
}

func MovesEqual(a, b Move) bool {
	if a.IsPass && b.IsPass {
		return true
	}
	if a.IsPass != b.IsPass {
		return false
	}
	if len(a.Cards) != len(b.Cards) {
		return false
	}
	return moveKey(a) == moveKey(b)
}

func (gs *GameState) StatusString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== AZEN (%d players) ===\n", gs.NumPlayers))
	medals := []string{"🥇", "🥈", "🥉", "4e"}
	for i, h := range gs.Hands {
		marker := "  "
		switch {
		case gs.Finished[i]:
			rank := gs.PlayerRank(i)
			if rank >= 0 && rank < len(medals) {
				marker = medals[rank] + " "
			} else {
				marker = "✓  "
			}
		case i == gs.CurrentTurn:
			marker = "▶  "
		}
		sb.WriteString(fmt.Sprintf("%sP%d [%2d cards]: %s\n", marker, i, h.Count(), h))
	}
	if gs.Round.IsOpen {
		sb.WriteString("Round: OPEN (play anything)\n")
	} else {
		sb.WriteString(fmt.Sprintf("Round: %dx cards, beat rank %s\n", gs.Round.Count, fmtRank(gs.Round.TableRank)))
	}
	if gs.GameOver && len(gs.Ranking) > 0 {
		sb.WriteString(fmt.Sprintf("🏆 Player %d WINS!\n", gs.Ranking[0]))
		if len(gs.Ranking) == gs.NumPlayers {
			sb.WriteString(fmt.Sprintf("💀 Player %d verliest.\n", gs.Ranking[len(gs.Ranking)-1]))
		}
	}
	return sb.String()
}

func fmtRank(r Rank) string { return (Card{Rank: r}).RankStr() }

// imin/imax vermijden conflict met Go 1.21+ builtins min/max
func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ═══════════════════════════════════════════════════════════════
// KNOWLEDGE
// ═══════════════════════════════════════════════════════════════

type PassRecord struct {
	Count     int
	TableRank Rank
}

type KnowledgeTracker struct {
	NumPlayers     int
	MyPlayerID     int
	MyHand         *Hand
	CardsPlayed    []Card
	DeadCards      []Card
	HandCounts     []int
	PlayedByPlayer [][]Card
	PassRecords    [][]PassRecord
	Suspicions     map[int][]Card
	Exclusions     map[int]map[Rank]int
}

func NewKnowledgeTracker(numPlayers, myID int, myHand *Hand, deadCards []Card) *KnowledgeTracker {
	kt := &KnowledgeTracker{
		NumPlayers:     numPlayers,
		MyPlayerID:     myID,
		MyHand:         myHand.Clone(),
		DeadCards:      make([]Card, len(deadCards)),
		HandCounts:     make([]int, numPlayers),
		PlayedByPlayer: make([][]Card, numPlayers),
		PassRecords:    make([][]PassRecord, numPlayers),
		Suspicions:     map[int][]Card{},
		Exclusions:     map[int]map[Rank]int{},
	}
	copy(kt.DeadCards, deadCards)
	for i := range kt.HandCounts {
		kt.HandCounts[i] = 18
	}
	// Spelregel: elke speler krijgt bij de verdeling gegarandeerd minstens 1 wildcard (2).
	// → Voeg voor elke tegenstander 1 Twee toe als zekere suspicion.
	// updateSuspicions() verwijdert die automatisch zodra de tegenstander een Twee speelt.
	for p := 0; p < numPlayers; p++ {
		if p != myID {
			kt.Suspicions[p] = []Card{{Rank: RankTwo}}
		}
	}
	return kt
}

func (kt *KnowledgeTracker) RecordMove(m Move) {
	if m.IsPass {
		return
	}
	kt.CardsPlayed = append(kt.CardsPlayed, m.Cards...)
	kt.PlayedByPlayer[m.PlayerID] = append(kt.PlayedByPlayer[m.PlayerID], m.Cards...)
	kt.HandCounts[m.PlayerID] -= len(m.Cards)
	if m.PlayerID == kt.MyPlayerID {
		kt.MyHand.Remove(m.Cards)
	}
	kt.updateSuspicions(m.Cards)
}

func (kt *KnowledgeTracker) RecordPass(passerID int, round RoundState) {
	if passerID == kt.MyPlayerID {
		return
	}
	if round.IsOpen {
		return
	}
	if kt.HandCounts[passerID] >= 9 {
		return
	}
	if round.Count != 1 {
		return
	}
	kt.PassRecords[passerID] = append(kt.PassRecords[passerID], PassRecord{
		Count:     round.Count,
		TableRank: round.TableRank,
	})
}

func (kt *KnowledgeTracker) AddSuspicion(playerID int, cc []Card) int {
	if playerID == kt.MyPlayerID {
		return 0
	}
	pool := kt.PossibleOpponentCards()
	poolCount := map[Rank]int{}
	for _, c := range pool {
		poolCount[c.Rank]++
	}
	for pid, susp := range kt.Suspicions {
		if pid == playerID {
			continue
		}
		for _, c := range susp {
			poolCount[c.Rank]--
		}
	}
	suspCount := map[Rank]int{}
	for _, c := range kt.Suspicions[playerID] {
		suspCount[c.Rank]++
	}
	added := 0
	for _, c := range cc {
		available := poolCount[c.Rank] - suspCount[c.Rank]
		if available > 0 {
			kt.Suspicions[playerID] = append(kt.Suspicions[playerID], c)
			suspCount[c.Rank]++
			added++
		}
	}
	return added
}

func (kt *KnowledgeTracker) ClearSuspicions(playerID int) {
	kt.Suspicions[playerID] = nil
}

func (kt *KnowledgeTracker) AddExclusion(playerID int, cc []Card) int {
	if playerID == kt.MyPlayerID {
		return 0
	}
	if kt.Exclusions[playerID] == nil {
		kt.Exclusions[playerID] = map[Rank]int{}
	}
	added := 0
	for _, c := range cc {
		pool := kt.PossibleOpponentCards()
		poolCount := 0
		for _, p := range pool {
			if p.Rank == c.Rank {
				poolCount++
			}
		}
		current := kt.Exclusions[playerID][c.Rank]
		if current < poolCount {
			kt.Exclusions[playerID][c.Rank]++
			added++
		}
	}
	return added
}

func (kt *KnowledgeTracker) ClearExclusions(playerID int) {
	kt.Exclusions[playerID] = nil
}

func (kt *KnowledgeTracker) ExcludedRanks(playerID int) map[Rank]bool {
	excluded := map[Rank]bool{}
	for _, pr := range kt.PassRecords[playerID] {
		excluded[RankJoker] = true // joker reset: als je er een had, had je gereset
		excluded[RankTwo] = true   // wildcard: als je er genoeg had, had je gespeeld
		for _, r := range NormalRanks() {
			if r > pr.TableRank {
				excluded[r] = true
			}
		}
	}
	pool := kt.PossibleOpponentCards()
	poolCount := map[Rank]int{}
	for _, c := range pool {
		poolCount[c.Rank]++
	}
	for rank, exclCount := range kt.Exclusions[playerID] {
		if exclCount > 0 {
			excluded[rank] = true
		}
		_ = poolCount
	}
	return excluded
}

func (kt *KnowledgeTracker) updateSuspicions(played []Card) {
	playedCount := map[Rank]int{}
	for _, c := range played {
		playedCount[c.Rank]++
	}
	for pid, suspected := range kt.Suspicions {
		if len(suspected) == 0 {
			continue
		}
		removed := map[Rank]int{}
		var newSusp []Card
		for _, c := range suspected {
			if removed[c.Rank] < playedCount[c.Rank] {
				removed[c.Rank]++
			} else {
				newSusp = append(newSusp, c)
			}
		}
		kt.Suspicions[pid] = newSusp
	}
	for pid, exclMap := range kt.Exclusions {
		if exclMap == nil {
			continue
		}
		for rank, count := range exclMap {
			played := playedCount[rank]
			if played > 0 && count > 0 {
				newCount := count - played
				if newCount <= 0 {
					delete(exclMap, rank)
				} else {
					exclMap[rank] = newCount
				}
			}
		}
		kt.Exclusions[pid] = exclMap
	}
}

func (kt *KnowledgeTracker) PossibleOpponentCards() []Card {
	knownCount := map[Rank]int{}
	for _, c := range kt.MyHand.Cards {
		knownCount[c.Rank]++
	}
	for _, c := range kt.CardsPlayed {
		knownCount[c.Rank]++
	}
	for _, c := range kt.DeadCards {
		knownCount[c.Rank]++
	}
	numDecks := 1
	if kt.NumPlayers == 4 {
		numDecks = 2
	}
	normalRanks := []Rank{
		RankThree, RankFour, RankFive, RankSix, RankSeven,
		RankEight, RankNine, RankTen, RankJack, RankQueen, RankKing,
		RankAce, RankTwo, // beide aanwezig in 4 exemplaren per deck
	}
	totalCount := map[Rank]int{}
	for _, r := range normalRanks {
		totalCount[r] = 4 * numDecks
	}
	totalCount[RankJoker] = 2 * numDecks
	var possible []Card
	allRanks := append(normalRanks, RankJoker)
	for _, r := range allRanks {
		available := totalCount[r] - knownCount[r]
		for i := 0; i < available; i++ {
			possible = append(possible, Card{Rank: r})
		}
	}
	return possible
}

func (kt *KnowledgeTracker) TotalOpponentCards() int {
	total := 0
	for i, count := range kt.HandCounts {
		if i != kt.MyPlayerID {
			total += count
		}
	}
	return total
}

// ═══════════════════════════════════════════════════════════════
// ENGINE - WEIGHTS
// ═══════════════════════════════════════════════════════════════

type Weights struct {
	AceBonus           float64 `json:"ace_bonus"`
	WildBonus          float64 `json:"wild_bonus"`
	SynergyBonus       float64 `json:"synergy_bonus"`
	CardDiffWeight     float64 `json:"card_diff_weight"`
	KingPenalty        float64 `json:"king_penalty"`
	QueenPenalty       float64 `json:"queen_penalty"`
	IsolatedLowPenalty float64 `json:"isolated_low_penalty"`
	ClusterBonus       float64 `json:"cluster_bonus"`
	TempoBonus         float64 `json:"tempo_bonus"`
	AcePlayFactor      float64 `json:"ace_play_factor"`
	WildPlayFactor     float64 `json:"wild_play_factor"`
	SynergyPenalty     float64 `json:"synergy_penalty"`
	RankPreference     float64 `json:"rank_preference"`
	PassBase           float64 `json:"pass_base"`
	PassSpecialFactor  float64 `json:"pass_special_factor"`
	PassBehindFactor   float64 `json:"pass_behind_factor"`
	UrgencyPenalty     float64 `json:"urgency_penalty"`
	EarlyGamePassFactor float64 `json:"early_game_pass_factor"`
}

func DefaultWeights() Weights {
	return Weights{
		AceBonus:            0.33,
		WildBonus:           0.38,   // verhoogd: 0.21→0.38 (wildcard bewaren loont meer)
		SynergyBonus:        0.20,   // verhoogd: 0.12→0.20 (ace+wild synergie beter gewaardeerd)
		CardDiffWeight:      0.085,
		KingPenalty:         0.05,
		QueenPenalty:        0.032,
		IsolatedLowPenalty:  0.042,
		ClusterBonus:        0.038,
		TempoBonus:          0.085,
		AcePlayFactor:       0.52,
		WildPlayFactor:      0.33,   // verlaagd: 0.41→0.33 (simulaties spelen wilds spaarzamer)
		SynergyPenalty:      0.40,
		RankPreference:      0.11,
		PassBase:            0.085,
		PassSpecialFactor:   0.225,
		PassBehindFactor:    0.31,
		UrgencyPenalty:      0.09,
		EarlyGamePassFactor: 0.32,    // nieuwe parameter: bonus voor early-game pass bij gelijke handen
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func LoadWeights(path string) (Weights, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultWeights(), err
	}
	w := DefaultWeights()
	if err := json.Unmarshal(data, &w); err != nil {
		return DefaultWeights(), err
	}
	return w, nil
}

func SaveWeights(w Weights, path string) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ═══════════════════════════════════════════════════════════════
// ENGINE - HEURISTICS
// ═══════════════════════════════════════════════════════════════

type HandStrength struct {
	CardCount      int
	WildCount      int
	AceCount       int
	HighCardCount  int
	LonelyKings    int
	PairCount      int
	TripleCount    int
	TempoScore     float64
	OverallScore   float64
}

func EvaluateHand(hand *Hand) HandStrength {
	hs := HandStrength{CardCount: hand.Count()}
	hs.WildCount = hand.CountWilds()
	hs.AceCount = hand.CountRank(RankAce) // Aas is nu de hoogste naturelle kaart
	rankCounts := make(map[Rank]int)
	for _, c := range hand.Cards {
		if !c.IsSpecial() {
			rankCounts[c.Rank]++
		}
	}
	for rank, count := range rankCounts {
		if rank >= RankJack {
			hs.HighCardCount += count
		}
		if count >= 2 {
			hs.PairCount++
		}
		if count >= 3 {
			hs.TripleCount++
		}
		if rank == RankKing {
			hs.LonelyKings = count
		}
	}
	hs.TempoScore = float64(hs.AceCount) * 2.0
	hs.OverallScore = 100.0 - float64(hs.CardCount)*5.0
	hs.OverallScore += float64(hs.WildCount) * 8.0
	hs.OverallScore += float64(hs.AceCount) * 10.0
	hs.OverallScore += float64(hs.PairCount) * 3.0
	hs.OverallScore += float64(hs.TripleCount) * 5.0
	kingPenalty := hs.LonelyKings - hs.WildCount
	if kingPenalty > 0 {
		hs.OverallScore -= float64(kingPenalty) * 6.0
	}
	lowCards := 0
	for _, rank := range []Rank{RankThree, RankFour, RankFive} {
		lowCards += rankCounts[rank]
	}
	if hs.AceCount == 0 && lowCards > 0 {
		hs.OverallScore -= float64(lowCards) * 2.0
	}
	return hs
}

type MoveQuality struct {
	Move             Move
	Score            float64
	Reasoning        string
	WastesWilds      bool
	WastesAces       bool
	CreatesWinThreat bool
}

func QuickEvaluateMove(gs *GameState, move Move) MoveQuality {
	mq := MoveQuality{Move: move}
	hand := gs.Hands[move.PlayerID]
	if move.IsPass {
		mq.Score = 0.0
		mq.Reasoning = "Pass"
		return mq
	}
	cardsAfter := hand.Count() - len(move.Cards)
	if cardsAfter == 0 {
		mq.Score = 100.0
		mq.CreatesWinThreat = true
		mq.Reasoning = "Winning move!"
		return mq
	}
	mq.Score = 50.0
	wildsUsed := 0
	resetsUsed := 0
	for _, c := range move.Cards {
		if c.IsWild() {
			wildsUsed++
		}
		if c.IsReset() {
			resetsUsed++
		}
	}
	effectiveRank := move.EffectiveRank(gs.Round.TableRank)
	if wildsUsed > 0 && effectiveRank < RankTen {
		mq.Score -= float64(wildsUsed) * 5.0
		mq.WastesWilds = true
		mq.Reasoning = "Wastes wildcards on low play"
	}
	if resetsUsed > 0 {
		mq.Score += 5.0
		mq.WastesAces = resetsUsed > 1
		if mq.WastesAces {
			mq.Score -= float64(resetsUsed-1) * 8.0
			mq.Reasoning = "Uses multiple resets unnecessarily"
		}
	}
	if effectiveRank > 0 {
		rankValue := float64(effectiveRank-RankThree) / float64(RankAce-RankThree)
		mq.Score -= rankValue * 10.0
	}
	mq.Score += float64(len(move.Cards)) * 2.0
	if cardsAfter <= 3 {
		mq.Score += 15.0
		mq.CreatesWinThreat = true
	}
	if gs.Round.IsOpen && resetsUsed > 0 {
		mq.Score += 10.0
	}
	return mq
}

func ShouldPass(gs *GameState, playerID int) bool {
	hand := gs.Hands[playerID]
	if hand.Count() <= 3 {
		return false
	}
	if gs.Round.IsOpen {
		return false
	}
	if gs.Round.TableRank <= RankSix {
		return false
	}
	if gs.Round.TableRank >= RankKing {
		normalCardsAbove := 0
		for _, c := range hand.Cards {
			// !IsSpecial() now includes Ace (natural card)
			if !c.IsSpecial() && c.Rank > gs.Round.TableRank {
				normalCardsAbove++
			}
		}
		if normalCardsAbove == 0 {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════
// ENGINE
// ═══════════════════════════════════════════════════════════════

type Config struct {
	Iterations     int
	MaxTime        time.Duration
	ExploreConst   float64
	NumPlayers     int
	Weights        Weights
	OmniscientMode bool
	NumWorkers     int
}

func DefaultConfig(numPlayers int) Config {
	w, _ := LoadWeights("storage/shared/Documents/weights.json")
	return Config{
		Iterations:   10000,
		MaxTime:      0,
		ExploreConst: 1.4,
		NumPlayers:   numPlayers,
		Weights:      w,
		NumWorkers:   2,
	}
}

type Engine struct {
	Config Config
	rng    *rand.Rand
}

func NewEngine(cfg Config) *Engine {
	return &Engine{
		Config: cfg,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

type mctsNode struct {
	move     Move
	parent   *mctsNode
	children []*mctsNode
	visits   int
	wins     float64
	playerID int
}

func newRoot() *mctsNode { return &mctsNode{playerID: -1} }

type MoveEval struct {
	Score   float64
	Visits  int
	Details []MoveDetail
}

func (me MoveEval) String() string {
	return fmt.Sprintf("Win%%: %.1f%% (%d visits)", me.Score*100, me.Visits)
}

type MoveDetail struct {
	Move    Move
	WinRate float64
	Visits  int
}

func (md MoveDetail) String() string {
	return fmt.Sprintf("  %s -> %.1f%% (%d visits)", md.Move, md.WinRate*100, md.Visits)
}

// findImmediateWin zoekt naar een gegarandeerde winnende zet ("schaakmat")
// via minimax. Bij ≤10 totale kaarten doorzoekt het ALLE mogelijke antwoorden
// van de tegenstander. Als er een pad is waar wij ALTIJD winnen ongeacht
// tegenstander-reacties, retourneert het de eerste zet van dat pad.
func findImmediateWin(gs *GameState) *Move {
	pid := gs.CurrentTurn
	handCount := gs.Hands[pid].Count()
	moves := gs.GetLegalMoves()

	// Snelle check: 1-zet win (geen Clone nodig)
	for _, m := range moves {
		if !m.IsPass && len(m.Cards) == handCount {
			mv := m
			return &mv
		}
	}

	// Minimax forced-win zoektocht voor diepere schaakmatten
	totalCards := 0
	for _, h := range gs.Hands {
		totalCards += h.Count()
	}
	if totalCards > 10 {
		return nil // te veel kaarten, minimax te duur
	}
	maxDepth := totalCards * 4 // ruimte voor pass-sequenties
	nodes := 0

	// Probeer niet-pass zetten eerst (sneller naar winst)
	for _, m := range moves {
		if m.IsPass {
			continue
		}
		sim := gs.Clone()
		sim.ApplyMove(m)
		if sim.GameOver && sim.Winner == pid {
			mv := m
			return &mv
		}
		if forcedWinSearch(sim, pid, maxDepth-1, &nodes, 500000) {
			mv := m
			return &mv
		}
	}
	return nil
}

// forcedWinSearch bepaalt via minimax of de huidige positie een gedwongen
// winst is voor myID. Bij onze beurt: minstens 1 zet moet forceren.
// Bij tegenstander: ALLE antwoorden moeten naar onze winst leiden.
func forcedWinSearch(gs *GameState, myID int, depth int, nodes *int, maxNodes int) bool {
	*nodes++
	if *nodes > maxNodes {
		return false
	}
	if gs.GameOver {
		return gs.Winner == myID
	}
	if depth <= 0 {
		return false
	}

	moves := gs.GetLegalMoves()

	if gs.CurrentTurn == myID {
		// Onze beurt: minstens 1 zet moet naar gedwongen winst leiden
		// Probeer speel-zetten eerst (efficiënter dan PASS)
		for _, m := range moves {
			if m.IsPass {
				continue
			}
			sim := gs.Clone()
			sim.ApplyMove(m)
			if forcedWinSearch(sim, myID, depth-1, nodes, maxNodes) {
				return true
			}
		}
		// PASS alleen in response-rondes (in open ronde altijd speelbare kaarten)
		if !gs.Round.IsOpen {
			for _, m := range moves {
				if !m.IsPass {
					continue
				}
				sim := gs.Clone()
				sim.ApplyMove(m)
				if forcedWinSearch(sim, myID, depth-1, nodes, maxNodes) {
					return true
				}
			}
		}
		return false
	}

	// Tegenstander: ALLE zetten moeten naar onze winst leiden
	for _, m := range moves {
		sim := gs.Clone()
		sim.ApplyMove(m)
		if !forcedWinSearch(sim, myID, depth-1, nodes, maxNodes) {
			return false
		}
	}
	return true
}

type workerResult struct {
	visits map[string]int
	wins   map[string]float64
	moves  map[string]Move
}

// filterDominatedMoves verwijdert wild+normal combinaties die gedomineerd worden door
// naturelle zetten (geen wildcards). Een wild-zet is gedomineerd als er een naturelle zet
// bestaat die een GELIJKE OF HOGERE effectieve rank bereikt — de wildcard is dan pure verspilling.
//
// Wild-zetten die een UNIEK HOGERE rank bereiken dan alle naturelle opties blijven altijd
// beschikbaar (bijv. K+wild als QQ de hoogste naturelle zet is: rank 13 > rank 12).
// Zo is de filter veilig bij zowel lage als hoge iteratiecounts.
//
// Puur-wildcardspellen (2+joker), reset-zetten (joker) en PASS worden nooit gefilterd.
// In open rondes geen filter.
func filterDominatedMoves(moves []Move, round RoundState) []Move {
	if round.IsOpen {
		return moves
	}
	tableRank := round.TableRank

	// Bepaal de hoogste effectieve rank die bereikbaar is via naturelle zetten (geen wilds, geen aces)
	maxNaturalRank := Rank(0)
	for _, m := range moves {
		if m.IsPass {
			continue
		}
		hasWild, hasReset, hasNormal := false, false, false
		for _, c := range m.Cards {
			if c.IsWild() {
				hasWild = true
			} else if c.IsReset() {
				hasReset = true
			} else {
				hasNormal = true
			}
		}
		if !hasWild && !hasReset && hasNormal {
			if er := m.EffectiveRank(tableRank); er > maxNaturalRank {
				maxNaturalRank = er
			}
		}
	}
	if maxNaturalRank == 0 {
		return moves // Geen naturelle zetten beschikbaar — alles bewaren
	}

	filtered := make([]Move, 0, len(moves))
	for _, m := range moves {
		if m.IsPass {
			filtered = append(filtered, m)
			continue
		}
		hasWild, hasReset, hasNormal := false, false, false
		for _, c := range m.Cards {
			if c.IsWild() {
				hasWild = true
			} else if c.IsReset() {
				hasReset = true
			} else {
				hasNormal = true
			}
		}
		// Oversized combo filter ("/" zetten met meer kaarten dan de tabelgrootte).
		// Voorbeeld: tafel = XX (2 kaarten), naturelle KK beschikbaar, maar engine speelt
		// "01/55" (4 kaarten: joker+aas+5+5). De joker+aas is verspilling als KK volstaat.
		// Regel: filter oversized special combos als een naturelle zet de tafel al verslaat.
		// Uitzondering: als geen naturelle zet bestaat (maxNaturalRank == 0 of ≤ tableRank),
		// dan is de oversized combo soms de enige optie → bewaren.
		if len(m.Cards) > round.Count && (hasWild || hasReset) && maxNaturalRank > tableRank {
			continue // gefilterd: naturelle zet is efficiënter dan deze "/" combo
		}
		if !hasWild {
			filtered = append(filtered, m) // Naturelle zet (geen wildcards): altijd bewaren
			continue
		}
		// Vanaf hier: zet bevat minstens één wildcard.

		// Puur-wild (geen reset, geen normaal — bijv. 2+2):
		// een naturelle zet verslaat de tafel en spaart alle wildcards → filter.
		if !hasNormal && !hasReset {
			continue // gefilterd: puur-wild gedomineerd door naturelle zetten
		}
		// Wild+joker (bijv. "2 0") EN wild+normaal (bijv. "K 2", "A 2"):
		// beide gebruiken een wildcard. Alleen bewaren als de effectieve rank
		// voldoende hoger is dan alle naturelle opties.
		// Bij hoge naturelle ranks (≥10) is een 1-rank voordeel te klein:
		// de wildcard-kost weegt niet op tegen het minimale voordeel.
		// Eis 2+ rank voordeel zodra maxNaturalRank ≥ RankTen.
		//
		// Voorbeeld wild+aas "0 1" (rank 14) vs KK (rank 13, ≥10):
		//   threshold=14, 14>14=false → gefilterd ✓ (KK volstaat!)
		// Voorbeeld wild+aas "0 1" (rank 14) vs JJ (rank 11, ≥10):
		//   threshold=12, 14>12=true  → bewaard  ✓ (3-rank voordeel)
		// Voorbeeld wild+normaal "K 2" (rank 13) vs QQ (rank 12, ≥10):
		//   threshold=13, 13>13=false → gefilterd ✓
		// Voorbeeld wild+normaal "K 2" (rank 13) vs JJ (rank 11, ≥10):
		//   threshold=12, 13>12=true  → bewaard  ✓
		er := m.EffectiveRank(tableRank)
		threshold := maxNaturalRank
		if maxNaturalRank >= RankTen {
			threshold++ // vereist 2+ rank voordeel bij hoge naturelle ranks
		}
		if er > threshold {
			filtered = append(filtered, m)
		}
		// Gefilterd: wildcard-voordeel te klein t.o.v. beschikbare naturelle zetten
	}
	return filtered
}

func (e *Engine) runWorker(gs *GameState, kt *KnowledgeTracker, iters int, seed int64, rootFiltered []Move) workerResult {
	workerCfg := e.Config
	workerCfg.NumWorkers = 1
	worker := &Engine{Config: workerCfg, rng: rand.New(rand.NewSource(seed))}
	root := newRoot()
	myID := gs.CurrentTurn
	hasDeadline := worker.Config.MaxTime > 0
	deadline := time.Now().Add(worker.Config.MaxTime)
	for iter := 0; iter < iters; iter++ {
		if hasDeadline && time.Now().After(deadline) {
			break
		}
		detGS := worker.determinize(gs, kt)
		if detGS == nil {
			continue
		}
		node, simGS := worker.selectExpand(root, detGS, myID, rootFiltered)
		result := worker.simulate(simGS, myID)
		worker.backprop(node, result, myID)
	}
	res := workerResult{
		visits: map[string]int{},
		wins:   map[string]float64{},
		moves:  map[string]Move{},
	}
	for _, ch := range root.children {
		k := mkey(ch.move)
		res.visits[k] += ch.visits
		res.wins[k] += ch.wins
		res.moves[k] = ch.move
	}
	return res
}

func (e *Engine) BestMove(gs *GameState, kt *KnowledgeTracker) (Move, MoveEval) {
	if win := findImmediateWin(gs); win != nil {
		return *win, MoveEval{Score: 1.0, Visits: 1}
	}
	// Filter gedomineerde wild-zetten zodat MCTS iteraties efficiënter benut worden
	rootFiltered := filterDominatedMoves(gs.GetLegalMoves(), gs.Round)
	numWorkers := e.Config.NumWorkers
	if numWorkers <= 1 {
		return e.bestMoveSingle(gs, kt, rootFiltered)
	}
	itersPerWorker := e.Config.Iterations / numWorkers
	if itersPerWorker < 1 {
		itersPerWorker = 1
	}
	seeds := make([]int64, numWorkers)
	for i := range seeds {
		seeds[i] = e.rng.Int63()
	}
	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			iters := itersPerWorker
			if idx == numWorkers-1 {
				iters = e.Config.Iterations - itersPerWorker*(numWorkers-1)
			}
			results[idx] = e.runWorker(gs, kt, iters, seeds[idx], rootFiltered)
		}(w)
	}
	wg.Wait()

	totalVisits := map[string]int{}
	totalWins := map[string]float64{}
	moveMap := map[string]Move{}
	for _, r := range results {
		for k, v := range r.visits {
			totalVisits[k] += v
		}
		for k, w := range r.wins {
			totalWins[k] += w
		}
		for k, m := range r.moves {
			moveMap[k] = m
		}
	}
	if len(moveMap) == 0 {
		return PassMove(gs.CurrentTurn), MoveEval{}
	}
	bestKey := ""
	bestVisits := -1
	for k, v := range totalVisits {
		if v > bestVisits {
			bestVisits = v
			bestKey = k
		}
	}
	// OmniscientMode (analyse): selecteer op WINRATE in plaats van visits.
	// In OmniscientMode bouwen alle workers dezelfde boom, maar PASS kan door
	// boom-asymmetrie (bredere subtree) meer visits krijgen ondanks lagere winrate.
	// Selectie op winrate geeft nauwkeurigere analyse-resultaten.
	// Eis: minstens 5% van totaal bezoeken, zodat noisy low-visit zetten niet winnen.
	if e.Config.OmniscientMode {
		totalIters := 0
		for _, v := range totalVisits {
			totalIters += v
		}
		minVisits := totalIters / 20 // minstens 5% van totaal
		bestWR := -1.0
		for k, v := range totalVisits {
			if v >= minVisits {
				wr := totalWins[k] / float64(v)
				if wr > bestWR {
					bestWR = wr
					bestKey = k
					bestVisits = v
				}
			}
		}
	}
	bestMove := moveMap[bestKey]
	// Nooit PASS aanbevelen als speler 4+ kaarten achter staat of als 2-speler eindspel.
	// 2-speler: tegenstander krijgt vrije open ronde → kan alles in 1 zet wegspelen.
	// Drempel gebaseerd op tegenstander's kaarten: < 9 = dodelijk gebied.
	myID := gs.CurrentTurn
	myCards := gs.Hands[myID].Count()
	oppCards := minOppHandCount(gs, myID)
	shouldOverridePass := bestMove.IsPass && (myCards-oppCards >= 4 ||
		(activePlayerCount(gs) <= 2 && (oppCards < 9 || myCards >= oppCards)))
	if shouldOverridePass {
		bestNonPassKey := ""
		bestNonPassWR := -1.0
		for k, m := range moveMap {
			if !m.IsPass {
				v := totalVisits[k]
				if v > 0 {
					wr2 := totalWins[k] / float64(v)
					if wr2 > bestNonPassWR {
						bestNonPassWR = wr2
						bestNonPassKey = k
					}
				}
			}
		}
		if bestNonPassKey != "" {
			bestKey = bestNonPassKey
			bestVisits = totalVisits[bestKey]
			bestMove = moveMap[bestKey]
		}
	}
	wr := 0.0
	if bestVisits > 0 {
		wr = totalWins[bestKey] / float64(bestVisits)
	}
	details := make([]MoveDetail, 0, len(moveMap))
	for k, m := range moveMap {
		v := totalVisits[k]
		w := 0.0
		if v > 0 {
			w = totalWins[k] / float64(v)
		}
		details = append(details, MoveDetail{Move: m, WinRate: w, Visits: v})
	}
	for i := 0; i < len(details); i++ {
		for j := i + 1; j < len(details); j++ {
			if details[j].Visits > details[i].Visits {
				details[i], details[j] = details[j], details[i]
			}
		}
	}
	return bestMove, MoveEval{Score: wr, Visits: bestVisits, Details: details}
}

func (e *Engine) bestMoveSingle(gs *GameState, kt *KnowledgeTracker, rootFiltered []Move) (Move, MoveEval) {
	root := newRoot()
	myID := gs.CurrentTurn
	hasDeadline := e.Config.MaxTime > 0
	deadline := time.Now().Add(e.Config.MaxTime)
	for iter := 0; iter < e.Config.Iterations; iter++ {
		if hasDeadline && time.Now().After(deadline) {
			break
		}
		detGS := e.determinize(gs, kt)
		if detGS == nil {
			continue
		}
		node, simGS := e.selectExpand(root, detGS, myID, rootFiltered)
		result := e.simulate(simGS, myID)
		e.backprop(node, result, myID)
	}
	bestMove, eval := e.pickBest(root, myID)
	// Nooit PASS aanbevelen als speler 4+ kaarten achter staat of als 2-speler eindspel.
	// Drempel gebaseerd op tegenstander's kaarten: < 9 = dodelijk gebied.
	myCards2 := gs.Hands[myID].Count()
	oppCards2 := minOppHandCount(gs, myID)
	if bestMove.IsPass && (myCards2-oppCards2 >= 4 ||
		(activePlayerCount(gs) <= 2 && (oppCards2 < 9 || myCards2 >= oppCards2))) {
		if m, ok := bestNonPassFromDetails(eval.Details); ok {
			for _, d := range eval.Details {
				if MovesEqual(d.Move, m) {
					return m, MoveEval{Score: d.WinRate, Visits: d.Visits, Details: eval.Details}
				}
			}
		}
	}
	return bestMove, eval
}

func (e *Engine) determinize(gs *GameState, kt *KnowledgeTracker) *GameState {
	if e.Config.OmniscientMode {
		return gs.Clone()
	}
	det := gs.Clone()
	possible := kt.PossibleOpponentCards()
	e.rng.Shuffle(len(possible), func(i, j int) {
		possible[i], possible[j] = possible[j], possible[i]
	})
	used := make([]bool, len(possible))
	for p := 0; p < gs.NumPlayers; p++ {
		if p == kt.MyPlayerID {
			continue
		}
		need := kt.HandCounts[p]
		if need < 0 {
			need = 0
		}
		excluded := kt.ExcludedRanks(p)
		suspCount := map[Rank]int{}
		for _, c := range kt.Suspicions[p] {
			suspCount[c.Rank]++
		}
		assignedSusp := map[Rank]int{}
		var tier1, tier2, tier3 []int
		for i, c := range possible {
			if used[i] {
				continue
			}
			if assignedSusp[c.Rank] < suspCount[c.Rank] {
				tier1 = append(tier1, i)
				assignedSusp[c.Rank]++
			} else if !excluded[c.Rank] {
				tier2 = append(tier2, i)
			} else {
				tier3 = append(tier3, i)
			}
		}
		// Sterke-kaarten-bias: ga er vanuit dat de tegenstander hoge naturelle kaarten
		// en wildcards bezit (assen en tweetjes) — in alle spelfasen.
		// Joker (reset) wordt NIET meer geprioriteerd: er zijn slechts 2 jokers in het
		// hele deck. Statistisch heeft de tegenstander ~76% kans op minstens één joker
		// als jij er geen hebt — dat is te laag om stelselmatig te assumeren.
		// Aas (4 exemplaren) en Twee/wildcard (4 exemplaren) hebben wél een hoge
		// kans (~84%) → die worden wel geprioriteerd in de determinisatie.
		// Motivatie: spelers bewaren hun sterkste naturelle kaarten tot laat; het derde
		// niet-gedealde pakje bevat relatief meer zwakke kaarten, dus de echte hand
		// heeft verhoudingsgewijs meer sterke kaarten dan een willekeurige steekproef.
		if need > 0 {
			var strongFront, rest []int
			for _, idx := range tier2 {
				r := possible[idx].Rank
				// Bias: Aas (hoogste naturelle kaart) en Twee (wildcard) vooraan
				if r == RankAce || r == RankTwo {
					strongFront = append(strongFront, idx)
				} else {
					rest = append(rest, idx)
				}
			}
			tier2 = append(strongFront, rest...)
		}
		ordered := append(append(tier1, tier2...), tier3...)
		if len(ordered) < need {
			return nil
		}
		hand := make([]Card, need)
		for i := 0; i < need; i++ {
			idx := ordered[i]
			hand[i] = possible[idx]
			used[idx] = true
		}
		det.Hands[p] = NewHand(hand)
	}
	return det
}

// selectExpand voert de selectie- en expansiefase van MCTS uit.
// rootFiltered: gefilterde zetten voor het root-knooppunt (nil = gebruik alle zetten).
func (e *Engine) selectExpand(node *mctsNode, gs *GameState, myID int, rootFiltered []Move) (*mctsNode, *GameState) {
	simGS := gs.Clone()
	for !simGS.GameOver {
		// Bij het root-knooppunt (parent == nil) enkel de gefilterde zetten aanbieden;
		// dieper in de boom altijd alle legale zetten gebruiken.
		var moves []Move
		if node.parent == nil && rootFiltered != nil {
			moves = rootFiltered
		} else {
			moves = simGS.GetLegalMoves()
		}
		if len(moves) == 0 {
			break
		}
		unexplored := e.unexploredMoves(node, moves)
		if len(unexplored) > 0 {
			m := unexplored[e.rng.Intn(len(unexplored))]
			child := &mctsNode{move: m, parent: node, playerID: m.PlayerID}
			node.children = append(node.children, child)
			simGS.ApplyMove(m)
			return child, simGS
		}
		best := e.ucb1Select(node, simGS.CurrentTurn == myID)
		if best == nil {
			break
		}
		simGS.ApplyMove(best.move)
		node = best
	}
	return node, simGS
}

func (e *Engine) unexploredMoves(node *mctsNode, moves []Move) []Move {
	explored := map[string]bool{}
	for _, ch := range node.children {
		explored[mkey(ch.move)] = true
	}
	var result []Move
	for _, m := range moves {
		if !explored[mkey(m)] {
			result = append(result, m)
		}
	}
	return result
}

func (e *Engine) ucb1Select(node *mctsNode, maximizing bool) *mctsNode {
	var best *mctsNode
	bestScore := math.Inf(-1)
	for _, ch := range node.children {
		if ch.visits == 0 {
			return ch
		}
		exploit := ch.wins / float64(ch.visits)
		if !maximizing {
			exploit = 1.0 - exploit
		}
		explore := e.Config.ExploreConst * math.Sqrt(math.Log(float64(node.visits))/float64(ch.visits))
		score := exploit + explore
		if score > bestScore {
			bestScore = score
			best = ch
		}
	}
	return best
}

func (e *Engine) simulate(gs *GameState, myID int) float64 {
	sim := gs.Clone()
	for i := 0; i < 400 && !sim.GameOver; i++ {
		moves := sim.GetLegalMoves()
		if len(moves) == 0 {
			break
		}
		m := e.smartRandom(moves, sim)
		// OmniscientMode rollout-verbetering:
		// Wanneer smartRandom PASS kiest maar er speelbare zetten zijn, override:
		//   - OPEN ronde: speel de ZWAKSTE kaart (dump goedkoop, bewaar sterke kaarten)
		//   - RESPONSE ronde: speel alleen als er een NATURELLE (niet-wild, niet-joker)
		//     respons is. Dit voorkomt dat rollouts wildcards/jokers verspillen op lage beats.
		//     Kies de LAAGSTE naturelle respons (net genoeg om tafel te verslaan).
		// Zo worden rollouts realistischer: spelers dumpen zwak in open rondes en
		// verdedigen efficiënt in response rondes, zonder sterke kaarten te verspillen.
		if e.Config.OmniscientMode && m.IsPass {
			if sim.Round.IsOpen {
				// Open ronde: dump zwakste kaart (laagste rank, geen specials)
				var weakest Move
				weakestRank := Rank(99)
				found := false
				for _, alt := range moves {
					if alt.IsPass {
						continue
					}
					// Prefer single normal cards for dumping
					hasSpecial := false
					for _, c := range alt.Cards {
						if c.IsWild() || c.IsReset() {
							hasSpecial = true
							break
						}
					}
					if hasSpecial {
						continue // Skip zetten met wildcards/jokers
					}
					er := alt.EffectiveRank(sim.Round.TableRank)
					if !found || er < weakestRank || (er == weakestRank && len(alt.Cards) > len(weakest.Cards)) {
						weakestRank = er
						weakest = alt
						found = true
					}
				}
				if found {
					m = weakest
				}
			} else {
				// Response ronde: alleen naturelle respons (geen wilds/jokers verspillen)
				// Kies de laagste naturelle respons (net genoeg om tafel te verslaan)
				var cheapest Move
				cheapestRank := Rank(99)
				found := false
				for _, alt := range moves {
					if alt.IsPass {
						continue
					}
					hasSpecial := false
					for _, c := range alt.Cards {
						if c.IsWild() || c.IsReset() {
							hasSpecial = true
							break
						}
					}
					if hasSpecial {
						continue
					}
					er := alt.EffectiveRank(sim.Round.TableRank)
					if !found || er < cheapestRank {
						cheapestRank = er
						cheapest = alt
						found = true
					}
				}
				if found {
					m = cheapest
				}
				// Als alleen wild/joker beschikbaar is: laat PASS staan (bewaar specials)
			}
		}
		sim.ApplyMove(m)
	}
	if sim.GameOver {
		return positionScore(sim, myID)
	}
	return e.evalPos(sim, myID)
}

func positionScore(gs *GameState, myID int) float64 {
	numP := gs.NumPlayers
	if numP <= 1 {
		return 1.0
	}
	rank := gs.PlayerRank(myID)
	if rank < 0 {
		return 0.0
	}
	return float64(numP-1-rank) / float64(numP-1)
}

func (e *Engine) smartRandom(moves []Move, gs *GameState) Move {
	wts := e.Config.Weights
	handCount := gs.Hands[gs.CurrentTurn].Count()

	// Directe win move altijd spelen
	for _, m := range moves {
		if !m.IsPass && len(m.Cards) == handCount {
			return m
		}
	}

	var plays []Move
	var pass Move
	for _, m := range moves {
		if m.IsPass {
			pass = m
		} else {
			plays = append(plays, m)
		}
	}
	if len(plays) == 0 {
		return pass
	}

	curHand := gs.Hands[gs.CurrentTurn]
	curWilds := curHand.CountRank(RankTwo)    // alleen 2 is wildcard
	curResets := curHand.CountRank(RankJoker) // joker is reset-kaart
	specialRatio := 0.0
	if handCount > 0 {
		specialRatio = float64(curWilds+curResets) / float64(handCount)
	}

	// PASS CHANCE
	passChance := wts.PassBase + specialRatio*wts.PassSpecialFactor

	// Early-game pass bonus: alleen bij 3+ spelers.
	// In 2-speler is passen altijd gevaarlijk (tegenstander krijgt open ronde), dus nooit verhogen.
	if handCount >= 8 && gs.activePlayerCount() > 2 {
		for i, h := range gs.Hands {
			if i != gs.CurrentTurn && !gs.Finished[i] {
				if handCount <= h.Count() {
					passChance += wts.EarlyGamePassFactor
				}
				break
			}
		}
	}

	// Achterlig-penalty
	minOpp := 999
	for i, h := range gs.Hands {
		if i != gs.CurrentTurn && !gs.Finished[i] && h.Count() < minOpp {
			minOpp = h.Count()
		}
	}
	if handCount > minOpp {
		diff := handCount - minOpp
		switch {
		case diff >= 7: passChance *= 0.03
		case diff >= 6: passChance *= 0.08
		case diff >= 5: passChance *= 0.15
		case diff >= 4: passChance *= 0.25
		case diff >= 3: passChance *= 0.42
		case diff >= 2: passChance *= 0.72
		}
	}

	// Late-game threat
	if minOpp <= 5 && !gs.Round.IsOpen {
		hasBeater := false
		for _, c := range curHand.Cards {
			// Joker (reset) is altijd een "beater" - reset de ronde en open opnieuw
			if c.IsWild() || c.IsReset() || (!c.IsSpecial() && c.Rank > gs.Round.TableRank) {
				hasBeater = true
				break
			}
		}
		if hasBeater {
			passChance = 0.02
		}
	}

	// 2-speler: PASS is in ALLE fases gevaarlijk — tegenstander krijgt vrije open ronde.
	// Vroeg/midspel: 65% minder passen. Eindspel (<9 kaarten): 90% minder passen.
	{
		activePlayers2 := 0
		for i, h := range gs.Hands {
			if !gs.Finished[i] && h.Count() > 0 {
				activePlayers2++
			}
		}
		if activePlayers2 <= 2 {
			if minOpp < 9 {
				passChance *= 0.10 // eindspel: vrijwel altijd fataal
			} else {
				passChance *= 0.35 // vroeg/midspel: ook gevaarlijk in 2-speler
			}
		}
	}

	if e.rng.Float64() < passChance {
		return pass
	}

	// === SPEEL-KEUZE ===
	acePlayFactor := wts.AcePlayFactor
	wildPlayFactor := wts.WildPlayFactor
	synergyPenalty := wts.SynergyPenalty

	weights := make([]float64, len(plays))
	total := 0.0
	for i, m := range plays {
		w := 1.0
		wilds := 0
		resets := 0
		effective := m.EffectiveRank(gs.Round.TableRank)

		for _, c := range m.Cards {
			if c.IsWild() {
				wilds++
			} else if c.IsReset() {
				resets++
			}
		}

		w *= math.Pow(acePlayFactor, float64(resets))
		w *= math.Pow(wildPlayFactor, float64(wilds))

		// === WILD-VERSPIJLING STRAF ===
		if wilds > 0 && handCount >= 12 {           // nog veel kaarten
			if effective <= RankFive {              // extreem lage beat (zoals 3)
				w *= 0.09                           // bijna onmogelijk maken
			} else if effective <= RankEight {
				w *= 0.28
			} else if effective <= RankTen {
				w *= 0.40                           // nieuw: rank 9-10 ook extra straf
			} else if effective <= RankQueen {
				w *= 0.40                           // aangescherpt: 0.60→0.40 (Q0 minder aantrekkelijk)
			}
			// King+wild: geen extra straf (K is terecht moeilijk anders te spelen)
		} else if wilds > 0 {
			if effective <= RankSix {
				w *= 0.25
			} else if effective <= RankNine {
				w *= 0.45                           // iets aangescherpt: 0.50→0.45
			}
		}
		// =====================================================

		// Bonus voor dumpen van lage normale kaarten (4 4 krijgt voorkeur)
		if wilds == 0 && resets == 0 && len(m.Cards) >= 1 {
			lowest := m.Cards[0].Rank
			if lowest <= RankFive {
				w *= 1.60
			} else if lowest <= RankEight {
				w *= 1.30
			}
		}

		// Singleton dump vs cluster-breek: bij een single-antwoord op een single tafel,
		// sterk voorkeur voor kaarten waarvan je er maar 1 hebt (geïsoleerde kaarten).
		// Straf als je een paar of triple moet breken voor een enkele kaart.
		if wilds == 0 && resets == 0 && !gs.Round.IsOpen &&
			len(m.Cards) == 1 && gs.Round.Count == 1 {
			rankCount := curHand.CountRank(effective)
			if rankCount == 1 {
				w *= 2.2 // singleton: wil je kwijt, speelt niet uit cluster
			} else if rankCount >= 2 {
				w *= 0.55 // breekt paar of triple: minder wenselijk
			}
		}

		if resets > 0 {
			if gs.Round.IsOpen {
				w *= 5.5 // joker reset in open ronde: extreem sterk
			} else {
				w *= 2.9 // joker reset als antwoord: sterk maar kostbaar
			}
		}
		if resets > 0 && wilds > 0 && gs.Round.IsOpen {
			w *= 2.1
		}

		if resets > 0 && wilds > 0 {
			w *= synergyPenalty
		}

		for _, c := range m.Cards {
			// !IsSpecial() includes Ace now (natural card), but Ace has high rank
			// The formula rewards LOW ranks; Ace (14) gets slight penalty = correct
			if !c.IsSpecial() {
				w *= 1.0 + wts.RankPreference*(13.0-float64(c.Rank))
			}
		}

		weights[i] = w
		total += w
	}

	r := e.rng.Float64() * total
	cum := 0.0
	for i, w := range weights {
		cum += w
		if r <= cum {
			return plays[i]
		}
	}
	return plays[len(plays)-1]
}

func (e *Engine) evalPos(gs *GameState, myID int) float64 {
	if gs.Finished[myID] {
		return positionScore(gs, myID)
	}
	myCount := gs.Hands[myID].Count()
	if myCount == 0 {
		return 1.0
	}
	wts := e.Config.Weights
	minOpp := 999
	for i, h := range gs.Hands {
		if i != myID && !gs.Finished[i] && h.Count() < minOpp {
			minOpp = h.Count()
		}
	}
	if minOpp == 999 {
		minOpp = 0
	}
	score := 0.5 + float64(minOpp-myCount)*wts.CardDiffWeight

	// Urgentiepenalty: bij 3+ kaarten achter een extra niet-lineaire straf.
	gap := myCount - minOpp
	if gap >= 3 {
		score -= math.Pow(float64(gap), 1.2) * wts.UrgencyPenalty  // exponent voor sterker effect bij grote gap		
	}

	hand := gs.Hands[myID]
	wilds := hand.CountRank(RankTwo)    // alleen 2 is wildcard
	resets := hand.CountRank(RankJoker) // joker is reset-kaart
	score += float64(resets) * wts.AceBonus  // joker is nu de reset = voormalige aas-bonus
	score += float64(wilds) * wts.WildBonus
	if resets > 0 && wilds > 0 {
		score += float64(imin(resets, wilds)) * wts.SynergyBonus
	}
	kings := hand.CountRank(RankKing)
	if kings > 0 && wilds == 0 && resets == 0 {
		score -= float64(kings) * wts.KingPenalty
	}
	queens := hand.CountRank(RankQueen)
	if queens > 0 && wilds == 0 && resets == 0 {
		score -= float64(queens) * wts.QueenPenalty
	}
	// Geïsoleerde lage kaarten (3-7): moeilijk te dumpen als single
	for r := RankThree; r <= RankSeven; r++ {
		if hand.CountRank(r) == 1 && wilds == 0 {
			score -= wts.IsolatedLowPenalty
		}
	}
	// Geïsoleerde midden-kaarten (8-X): ook lastig, maar iets minder erg
	for r := RankEight; r <= RankTen; r++ {
		if hand.CountRank(r) == 1 && wilds == 0 {
			score -= wts.IsolatedLowPenalty * 0.5
		}
	}
	for r := RankThree; r <= RankAce; r++ {
		cnt := hand.CountRank(r)
		if cnt >= 2 {
			score += float64(cnt-1) * wts.ClusterBonus
		}
	}
	// Tempo: open ronde op je beurt is een voordeel, maar proportioneel —
	// niet zo groot dat het kaartdifferentieel overschaduwt.
	// Oude waarde (4.0x = 0.34 + 0.45 voor Joker = 0.79!) was absurd groot
	// en maakte dat PASS kunstmatig goed scoorde in MCTS rollouts,
	// omdat rolloutevaluaties met open-ronde-posities altijd ~1.0 teruggaven.
	if gs.Round.IsOpen && gs.CurrentTurn == myID {
		score += wts.TempoBonus * 1.2 // 0.085*1.2 = ~0.10 (was 0.34)
		if hand.CountResets() > 0 {
			score += 0.08 // was 0.45 — joker+tempo is sterk maar niet allesbepalend
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}

func (e *Engine) backprop(node *mctsNode, result float64, myID int) {
	for node != nil {
		node.visits++
		node.wins += result // Altijd vanuit myID-perspectief: ucb1Select inverteeert voor tegenstanders.
		node = node.parent
	}
}

func (e *Engine) pickBest(root *mctsNode, myID int) (Move, MoveEval) {
	if len(root.children) == 0 {
		return PassMove(myID), MoveEval{}
	}
	var bestNode *mctsNode
	if e.Config.OmniscientMode {
		// OmniscientMode (analyse): selecteer op winrate, niet op visits.
		// Eis: minstens 5% van root visits, zodat noisy low-visit zetten niet winnen.
		totalV := 0
		for _, ch := range root.children {
			totalV += ch.visits
		}
		minV := totalV / 20
		bestWR := -1.0
		for _, ch := range root.children {
			if ch.visits >= minV {
				wr := ch.wins / float64(ch.visits)
				if wr > bestWR {
					bestWR = wr
					bestNode = ch
				}
			}
		}
	}
	if bestNode == nil {
		// Fallback (of non-OmniscientMode): selecteer op visits
		bestV := -1
		for _, ch := range root.children {
			if ch.visits > bestV {
				bestV = ch.visits
				bestNode = ch
			}
		}
	}
	wr := 0.0
	if bestNode.visits > 0 {
		wr = bestNode.wins / float64(bestNode.visits)
	}
	details := make([]MoveDetail, len(root.children))
	for i, ch := range root.children {
		w := 0.0
		if ch.visits > 0 {
			w = ch.wins / float64(ch.visits)
		}
		details[i] = MoveDetail{Move: ch.move, WinRate: w, Visits: ch.visits}
	}
	for i := 0; i < len(details); i++ {
		for j := i + 1; j < len(details); j++ {
			if details[j].Visits > details[i].Visits {
				details[i], details[j] = details[j], details[i]
			}
		}
	}
	return bestNode.move, MoveEval{Score: wr, Visits: bestNode.visits, Details: details}
}

// minOppHandCount geeft het laagste kaartaantal van actieve tegenstanders.
func minOppHandCount(gs *GameState, myID int) int {
	min := 999
	for i, h := range gs.Hands {
		if i != myID && !gs.Finished[i] && h.Count() < min {
			min = h.Count()
		}
	}
	if min == 999 {
		return 0
	}
	return min
}

// activePlayerCount telt het aantal spelers dat nog actief is (niet gefinished, nog kaarten).
func activePlayerCount(gs *GameState) int {
	count := 0
	for i := range gs.Hands {
		if !gs.Finished[i] && gs.Hands[i].Count() > 0 {
			count++
		}
	}
	return count
}

// bestNonPassFromDetails geeft de non-pass zet met de hoogste win-rate uit MCTS-details.
func bestNonPassFromDetails(details []MoveDetail) (Move, bool) {
	bestWR := -1.0
	var bestMove Move
	found := false
	for _, d := range details {
		if !d.Move.IsPass && d.Visits > 0 && d.WinRate > bestWR {
			bestWR = d.WinRate
			bestMove = d.Move
			found = true
		}
	}
	return bestMove, found
}

func (e *Engine) AnalyzeMove(gs *GameState, kt *KnowledgeTracker, m Move) MoveDetail {
	myID := gs.CurrentTurn
	wins := 0.0
	sims := 1000
	for i := 0; i < sims; i++ {
		det := e.determinize(gs, kt)
		if det == nil {
			continue
		}
		sim := det.Clone()
		sim.ApplyMove(m)
		result := e.simulate(sim, myID)
		wins += result
	}
	return MoveDetail{Move: m, WinRate: wins / float64(sims), Visits: sims}
}

// FindMoveInEval zoekt een zet op in de MoveEval-details die door BestMove zijn berekend.
// Geeft (detail, true) terug als gevonden, anders (zero, false).
func FindMoveInEval(eval MoveEval, m Move) (MoveDetail, bool) {
	for _, d := range eval.Details {
		if MovesEqual(d.Move, m) && d.Visits > 0 {
			return d, true
		}
	}
	return MoveDetail{}, false
}

func mkey(m Move) string {
	if m.IsPass {
		return "PASS"
	}
	sorted := make([]Card, len(m.Cards))
	copy(sorted, m.Cards)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Rank > sorted[j].Rank || (sorted[i].Rank == sorted[j].Rank && sorted[i].Suit > sorted[j].Suit) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	key := ""
	for _, c := range sorted {
		key += c.String()
	}
	return key
}

// ═══════════════════════════════════════════════════════════════
// IO
// ═══════════════════════════════════════════════════════════════

type GameLog struct {
	NumPlayers int
	Hands      [][]Card
	DeadCards  []Card
	Moves      []Move
	Winner     int
}

func SaveGame(path string, log *GameLog) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "AZEN GAME LOG\n")
	fmt.Fprintf(f, "players:%d\n", log.NumPlayers)
	fmt.Fprintf(f, "winner:%d\n", log.Winner)
	for i, hand := range log.Hands {
		parts := make([]string, len(hand))
		for j, c := range hand {
			parts[j] = c.String()
		}
		fmt.Fprintf(f, "hand:%d:%s\n", i, strings.Join(parts, ","))
	}
	if len(log.DeadCards) > 0 {
		parts := make([]string, len(log.DeadCards))
		for i, c := range log.DeadCards {
			parts[i] = c.String()
		}
		fmt.Fprintf(f, "dead:%s\n", strings.Join(parts, ","))
	}
	fmt.Fprintf(f, "---\n")
	for _, m := range log.Moves {
		if m.IsPass {
			fmt.Fprintf(f, "P%d:PASS\n", m.PlayerID)
		} else {
			parts := make([]string, len(m.Cards))
			for i, c := range m.Cards {
				parts[i] = c.String()
			}
			fmt.Fprintf(f, "P%d:%s\n", m.PlayerID, strings.Join(parts, ","))
		}
	}
	return nil
}

func LoadGame(path string) (*GameLog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	log := &GameLog{Winner: -1}
	scanner := bufio.NewScanner(f)
	inMoves := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "AZEN GAME LOG" {
			continue
		}
		if line == "---" {
			inMoves = true
			continue
		}
		if inMoves {
			m, err := parseMoveLog(line)
			if err != nil {
				return nil, fmt.Errorf("parsing move %q: %v", line, err)
			}
			log.Moves = append(log.Moves, m)
			continue
		}
		if strings.HasPrefix(line, "players:") {
			n, _ := strconv.Atoi(strings.TrimPrefix(line, "players:"))
			log.NumPlayers = n
		} else if strings.HasPrefix(line, "winner:") {
			n, _ := strconv.Atoi(strings.TrimPrefix(line, "winner:"))
			log.Winner = n
		} else if strings.HasPrefix(line, "hand:") {
			parts := strings.SplitN(strings.TrimPrefix(line, "hand:"), ":", 2)
			if len(parts) == 2 {
				cc, err := ParseCards(parts[1])
				if err != nil {
					return nil, err
				}
				idx, _ := strconv.Atoi(parts[0])
				for len(log.Hands) <= idx {
					log.Hands = append(log.Hands, nil)
				}
				log.Hands[idx] = cc
			}
		} else if strings.HasPrefix(line, "dead:") {
			cc, err := ParseCards(strings.TrimPrefix(line, "dead:"))
			if err != nil {
				return nil, err
			}
			log.DeadCards = cc
		}
	}
	return log, scanner.Err()
}

func parseMoveLog(line string) (Move, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return Move{}, fmt.Errorf("invalid move format")
	}
	pid := 0
	fmt.Sscanf(parts[0], "P%d", &pid)
	if strings.TrimSpace(parts[1]) == "PASS" {
		return PassMove(pid), nil
	}
	cc, err := ParseCards(parts[1])
	if err != nil {
		return Move{}, err
	}
	return Move{PlayerID: pid, Cards: cc}, nil
}

// ---- Reader / Display ----

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

func (r *Reader) ReadCards(prompt string) ([]Card, error) {
	s := r.ReadLine(prompt)
	return ParseCards(s)
}

func (r *Reader) ReadYesNo(prompt string) bool {
	s := strings.ToLower(r.ReadLine(prompt + " (j/n): "))
	return s == "j" || s == "y" || s == "ja" || s == "yes"
}

func (r *Reader) ReadMove(playerID int, prompt string) (Move, error) {
	if prompt == "" {
		prompt = fmt.Sprintf("Speler %d zet: ", playerID+1)
	}
	s := r.ReadLine(prompt)
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "pass" || lower == "p" {
		return PassMove(playerID), nil
	}
	cc, err := ParseCards(s)
	if err != nil {
		return Move{}, err
	}
	return Move{PlayerID: playerID, Cards: cc}, nil
}

func PrintHeader(title string) {
	border := strings.Repeat("═", len(title)+4)
	fmt.Printf("\n╔%s╗\n║  %s  ║\n╚%s╝\n\n", border, title, border)
}

func PrintSubHeader(title string) {
	fmt.Printf("\n─── %s ───\n", title)
}

func PrintCards(hand *Hand) {
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

func PrintMoveOptions(moves []Move, max int) {
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

func FormatMove(m Move) string {
	if m.IsPass {
		return "PASS"
	}
	sorted := make([]Card, len(m.Cards))
	copy(sorted, m.Cards)
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

// ═══════════════════════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════════════════════

type settings struct {
	numThreads int
}

func main() {
	reader := NewReader()
	cfg := settings{numThreads: 2}
	for {
		PrintHeader("AZEN Engine v1.0")
		fmt.Println("Welkom bij de AZEN kaartspel engine!")
		fmt.Println()
		fmt.Printf("  [0] Instellingen  (threads: %d)\n", cfg.numThreads)
		fmt.Println("  [1] Spelen  - Engine suggereert zetten voor jou")
		fmt.Println("  [2] Analyse - Bekijk een gespeeld spel opnieuw")
		fmt.Println("  [3] Simuleer - Kijk hoe de engine tegen zichzelf speelt")
		fmt.Println("  [4] Snelle analyse - Plak een volledige partij in één keer")
		fmt.Println("  [5] Weight Tuner - Optimaliseer de AI gewichten (krachtige PC)")
		fmt.Println()
		modeStr := reader.ReadLine("Kies modus (0/1/2/3/4): ")
		mode, _ := strconv.Atoi(modeStr)
		switch mode {
		case 0:
			cfg = settingsMenu(reader, cfg)
		case 1:
			playMode(reader, cfg)
			return
		case 2:
			analyzeMode(reader, cfg)
			return
		case 3:
			simulateMode(reader, cfg)
			return
		case 4:
			quickAnalyzeMode(reader, cfg)
			return
		case 5:
			weightTunerMode(reader, cfg)
			return
		default:
			playMode(reader, cfg)
			return
		}
	}
}

func settingsMenu(reader *Reader, cfg settings) settings {
	PrintHeader("Instellingen")
	fmt.Printf("Huidige threads: %d\n", cfg.numThreads)
	fmt.Println()
	fmt.Println("Threads bepalen hoeveel parallelle ISMCTS-bomen tegelijk draaien.")
	fmt.Println("Meer threads = sterkere engine bij dezelfde iteraties.")
	fmt.Println("  1  = sequentieel (origineel gedrag)")
	fmt.Println("  2  = standaard (goed evenwicht, aanbevolen)")
	fmt.Println("  4+ = sterker maar meer CPU-gebruik")
	fmt.Println()
	if n, err := reader.ReadInt(fmt.Sprintf("Aantal threads (huidige: %d): ", cfg.numThreads)); err == nil && n >= 1 {
		if n > 64 {
			n = 64
		}
		cfg.numThreads = n
		fmt.Printf("✅ Threads ingesteld op %d.\n\n", n)
	} else {
		fmt.Printf("Ongewijzigd (%d threads).\n\n", cfg.numThreads)
	}
	return cfg
}

func handleGok(input string, tracker *KnowledgeTracker, myPlayer int, numPlayers int) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(input))
	if !strings.HasPrefix(lower, "gok") {
		return false, ""
	}
	rest := strings.TrimSpace(input[3:])
	if rest == "" {
		var sb strings.Builder
		sb.WriteString("🔍 Huidige vermoedens:\n")
		any := false
		for p := 0; p < numPlayers; p++ {
			if p == myPlayer {
				continue
			}
			susp := tracker.Suspicions[p]
			excl := tracker.Exclusions[p]
			if len(susp) > 0 {
				sb.WriteString(fmt.Sprintf("  Speler %d heeft:      %s\n", p+1, CardsToString(susp)))
				any = true
			}
			if len(excl) > 0 {
				var parts []string
				for r, cnt := range excl {
					for i := 0; i < cnt; i++ {
						parts = append(parts, (Card{Rank: r}).RankStr())
					}
				}
				sb.WriteString(fmt.Sprintf("  Speler %d heeft NIET: %s\n", p+1, strings.Join(parts, " ")))
				any = true
			}
		}
		if !any {
			sb.WriteString("  (geen vermoedens ingevoerd)\n")
		}
		return true, sb.String()
	}
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return true, "⚠️  Formaat: gok 2:KK  of  gok 2:clear  of  gok"
	}
	playerNum, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || playerNum < 1 || playerNum > numPlayers {
		return true, fmt.Sprintf("⚠️  Ongeldig spelernummer: %s", parts[0])
	}
	targetID := playerNum - 1
	if targetID == myPlayer {
		return true, "⚠️  Je hoeft geen vermoeden in te voeren voor jezelf."
	}
	arg := strings.TrimSpace(parts[1])
	if strings.ToLower(arg) == "clear" {
		tracker.ClearSuspicions(targetID)
		tracker.ClearExclusions(targetID)
		return true, fmt.Sprintf("🔍 Alle vermoedens voor Speler %d gewist.", playerNum)
	}
	isNegative := strings.HasPrefix(arg, "-")
	if isNegative {
		arg = arg[1:]
	}
	parsed, err := ParseCards(arg)
	if err != nil {
		return true, fmt.Sprintf("⚠️  Kaarten niet herkend: %v", err)
	}
	if isNegative {
		added := tracker.AddExclusion(targetID, parsed)
		msg := fmt.Sprintf("🚫 Speler %d heeft NIET: %s  (%d toegevoegd)",
			playerNum, CardsToString(parsed), added)
		return true, msg
	}
	added := tracker.AddSuspicion(targetID, parsed)
	susp := tracker.Suspicions[targetID]
	msg := fmt.Sprintf("🔍 Gok Speler %d heeft: %s  (%d kaart(en) toegevoegd, totaal vermoeden: %s)",
		playerNum, CardsToString(parsed), added, CardsToString(susp))
	if added < len(parsed) {
		msg += fmt.Sprintf("\n   ⚠️  %d kaart(en) niet toegevoegd: al gespeeld of niet meer in pool", len(parsed)-added)
	}
	return true, msg
}

// printGameStatus toont de spelstatus met vermoedens voor tegenstanders.
// Vervangt gs.StatusString() in speelmodus zodat gok-info zichtbaar is.
func printGameStatus(gs *GameState, tracker *KnowledgeTracker, myPlayer int) {
	fmt.Printf("=== AZEN (%d spelers) ===\n", gs.NumPlayers)
	medals := []string{"🥇", "🥈", "🥉", "4e"}
	for i := range gs.Hands {
		marker := "  "
		switch {
		case gs.Finished[i]:
			rank := gs.PlayerRank(i)
			if rank >= 0 && rank < len(medals) {
				marker = medals[rank] + " "
			} else {
				marker = "✓  "
			}
		case i == gs.CurrentTurn:
			marker = "▶  "
		}

		count := gs.Hands[i].Count()
		var handDisplay string

		if i == myPlayer {
			h := gs.Hands[i].Clone()
			h.Sort()
			handDisplay = h.String()
		} else {
			susp := tracker.Suspicions[i]
			var parts []string
			for _, c := range susp {
				parts = append(parts, c.RankStr())
			}
			remaining := count - len(susp)
			if remaining < 0 {
				remaining = 0
			}
			for j := 0; j < remaining; j++ {
				parts = append(parts, "?")
			}
			handDisplay = strings.Join(parts, " ")
		}

		fmt.Printf("%sP%d [%2d kaarten]: %s\n", marker, i+1, count, handDisplay)
	}

	if gs.Round.IsOpen {
		fmt.Println("Ronde: OPEN (speel alles)")
	} else {
		rankStr := (Card{Rank: gs.Round.TableRank}).RankStr()
		fmt.Printf("Ronde: %dx kaarten, rank %s verslaan\n", gs.Round.Count, rankStr)
	}
	if gs.GameOver && len(gs.Ranking) > 0 {
		fmt.Printf("🏆 Speler %d WINT!\n", gs.Ranking[0]+1)
	}
	fmt.Println()
}

func playMode(reader *Reader, cfg settings) {
	PrintHeader("Speel Modus")
	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}
	myPlayer := 0
	if p, err := reader.ReadInt("Jouw spelernummer (1-" + strconv.Itoa(numPlayers) + "): "); err == nil && p >= 1 && p <= numPlayers {
		myPlayer = p - 1
	}
	hands := make([]*Hand, numPlayers)
	cardCounts := make([]int, numPlayers)
	for i := 0; i < numPlayers; i++ {
		cardCounts[i] = 18
		if n, err := reader.ReadInt(fmt.Sprintf("Aantal startkaarten voor Speler %d (standaard 18): ", i+1)); err == nil && n > 0 {
			cardCounts[i] = n
		}
		if i == myPlayer {
			fmt.Println("\nVoer jouw kaarten in (komma, spatie of aaneengesloten):")
			fmt.Println("  Voorbeeld: KK3XJ19Q25  of  K,K,3,X,J  of  K K 3 X J")
			fmt.Println("  Typ 'help' voor uitleg.")
			fmt.Println()
			var myHand *Hand
			for {
				input := reader.ReadLine("Jouw kaarten: ")
				if strings.ToLower(input) == "help" {
					PrintHelp()
					continue
				}
				parsed, err := ParseCards(input)
				if err != nil {
					fmt.Printf("Fout: %v\n", err)
					continue
				}
				if len(parsed) != cardCounts[i] {
					fmt.Printf("Verwacht %d kaarten, kreeg %d. Probeer opnieuw.\n", cardCounts[i], len(parsed))
					continue
				}
				myHand = NewHand(parsed)
				break
			}
			hands[i] = myHand
			fmt.Println("\n\nJouw hand:")
			PrintCards(myHand)
		} else {
			ph := make([]Card, cardCounts[i])
			hands[i] = NewHand(ph)
		}
	}
	var deadCards []Card
	if numPlayers == 2 {
		fmt.Println("\nMet 2 spelers zijn 18 kaarten niet in spel (engine houdt hiermee rekening).")
	}
	tracker := NewKnowledgeTracker(numPlayers, myPlayer, hands[myPlayer], deadCards)
	gs := NewGameWithHands(hands, deadCards, 0)
	iters := 10000
	if n, err := reader.ReadInt("Engine-iteraties per zet (standaard 10000, meer = nauwkeuriger maar trager): "); err == nil && n > 0 {
		iters = n
	}
	engConfig := DefaultConfig(numPlayers)
	engConfig.Iterations = iters
	engConfig.MaxTime = 0
	engConfig.NumWorkers = cfg.numThreads
	eng := NewEngine(engConfig)
	startStr := reader.ReadLine("Wie begint? (spelernummer of 'ik'): ")
	if strings.ToLower(startStr) == "ik" || strings.ToLower(startStr) == "me" {
		gs.CurrentTurn = myPlayer
	} else if p, err := strconv.Atoi(startStr); err == nil && p >= 1 && p <= numPlayers {
		gs.CurrentTurn = p - 1
	}
	fmt.Printf("\n🎮 Spel gestart! Typ 'help' voor commando's, 'gok 2:KK' voor vermoedens, 'rethink' om opnieuw te berekenen.\n\n")
	for !gs.GameOver {
		printGameStatus(gs, tracker, myPlayer)
		if gs.CurrentTurn == myPlayer {
			PrintSubHeader("Jouw beurt")
			PrintCards(gs.Hands[myPlayer])
			fmt.Println("\n🤔 Engine denkt na...")
			bestMove, eval := eng.BestMove(gs, tracker)
			if eval.Score >= 1.0 && eval.Visits <= 1 {
				fmt.Printf("\n♟️  Gedwongen winst gevonden: %s\n\n", FormatMove(bestMove))
			} else {
				fmt.Printf("\n💡 Engine suggereert: %s (winst: %s)\n\n",
					FormatMove(bestMove), FormatScore(eval.Score))
			}
			for {
				input := reader.ReadLine("Jouw zet (of 'hint'/'rethink'/'help'/'hand'/'status'/'moves'/'gok'): ")
				lower := strings.ToLower(input)
				switch lower {
				case "help":
					PrintHelp()
					continue
				case "hand":
					PrintCards(gs.Hands[myPlayer])
					continue
				case "status":
					printGameStatus(gs, tracker, myPlayer)
					continue
				case "rethink":
					fmt.Println("\n🤔 Engine herdenkt de situatie...")
					bestMove, eval = eng.BestMove(gs, tracker)
					if eval.Score >= 1.0 && eval.Visits <= 1 {
						fmt.Printf("\n♟️  Gedwongen winst gevonden: %s\n\n", FormatMove(bestMove))
					} else {
						fmt.Printf("\n💡 Nieuwe suggestie: %s (winst: %s)\n\n",
							FormatMove(bestMove), FormatScore(eval.Score))
					}
					continue
				case "hint":
					fmt.Printf("💡 Suggestie: %s (winst: %s)\n",
						FormatMove(bestMove), FormatScore(eval.Score))
					continue
				case "moves":
					PrintMoveOptions(gs.GetLegalMoves(), 20)
					continue
				case "quit", "exit":
					fmt.Println("Tot ziens!")
					os.Exit(0)
				}
				if handled, msg := handleGok(input, tracker, myPlayer, numPlayers); handled {
					fmt.Println(msg)
					continue
				}
				mainInput, followInput, hasFollow := strings.Cut(input, "/")
				mainInput = strings.TrimSpace(mainInput)
				mainLower := strings.ToLower(mainInput)
				var move Move
				if mainLower == "pass" || mainLower == "p" || mainLower == "-" {
					move = PassMove(myPlayer)
				} else {
					parsed, err := ParseCards(mainInput)
					if err != nil {
						fmt.Printf("Fout: %v\n", err)
						continue
					}
					move = Move{PlayerID: myPlayer, Cards: parsed}
				}
				if err := gs.ValidateMove(move); err != nil {
					fmt.Printf("Ongeldige zet: %v\n", err)
					continue
				}
				if move.IsPass {
					tracker.RecordPass(move.PlayerID, gs.Round)
				}
				gs.ApplyMove(move)
				tracker.RecordMove(move)
				if hasFollow && !gs.GameOver && gs.CurrentTurn == myPlayer {
					followInput = strings.TrimSpace(followInput)
					parsed, err := ParseCards(followInput)
					if err != nil {
						fmt.Printf("✅ Gespeeld: %s\n⚠️  Fout in vervolg-zet: %v\n\n", FormatMove(move), err)
						break
					}
					followMove := Move{PlayerID: myPlayer, Cards: parsed}
					if err := gs.ValidateMove(followMove); err != nil {
						fmt.Printf("✅ Gespeeld: %s\n⚠️  Ongeldige vervolg-zet: %v\n\n", FormatMove(move), err)
						break
					}
					gs.ApplyMove(followMove)
					tracker.RecordMove(followMove)
					fmt.Printf("✅ Gespeeld: %s / %s\n\n", FormatMove(move), FormatMove(followMove))
				} else {
					fmt.Printf("✅ Gespeeld: %s\n\n", FormatMove(move))
				}
				break
			}
		} else {
			playerNum := gs.CurrentTurn + 1
			oppID := gs.CurrentTurn
			PrintSubHeader(fmt.Sprintf("Beurt van Speler %d", playerNum))
			for {
				input := reader.ReadLine(fmt.Sprintf("Zet van Speler %d (of '-' voor pas, 'gok' voor vermoeden): ", playerNum))
				lower := strings.ToLower(strings.TrimSpace(input))
				if lower == "help" {
					PrintHelp()
					continue
				}
				if lower == "quit" || lower == "exit" {
					fmt.Println("Tot ziens!")
					os.Exit(0)
				}
				if handled, msg := handleGok(input, tracker, myPlayer, numPlayers); handled {
					fmt.Println(msg)
					continue
				}
				mainInput, followInput, hasFollow := strings.Cut(input, "/")
				mainInput = strings.TrimSpace(mainInput)
				mainLower := strings.ToLower(mainInput)
				var move Move
				if mainLower == "pass" || mainLower == "p" || mainLower == "-" {
					move = PassMove(oppID)
				} else {
					parsed, err := ParseCards(mainInput)
					if err != nil {
						fmt.Printf("Fout: %v\n", err)
						continue
					}
					move = Move{PlayerID: oppID, Cards: parsed}
				}
				if move.IsPass {
					tracker.RecordPass(move.PlayerID, gs.Round)
				}
				gs.ApplyMove(move)
				tracker.RecordMove(move)
				if hasFollow && !gs.GameOver && gs.CurrentTurn == oppID {
					followInput = strings.TrimSpace(followInput)
					if parsed, err := ParseCards(followInput); err == nil {
						followMove := Move{PlayerID: oppID, Cards: parsed}
						gs.ApplyMove(followMove)
						tracker.RecordMove(followMove)
						fmt.Printf("📝 Speler %d speelde: %s / %s\n\n", playerNum, FormatMove(move), FormatMove(followMove))
						break
					}
				}
				fmt.Printf("📝 Speler %d speelde: %s\n\n", playerNum, FormatMove(move))
				break
			}
		}
	}
	PrintHeader("Spel Voorbij!")
	printRanking(gs)
}

func analyzeMode(reader *Reader, cfg settings) {
	PrintHeader("Analyse Modus")
	fmt.Println("Voer het volledige spel in voor analyse.")
	fmt.Println()
	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}
	hands := make([]*Hand, numPlayers)
	for i := 0; i < numPlayers; i++ {
		cardCount := 18
		if n, err := reader.ReadInt(fmt.Sprintf("Aantal startkaarten voor Speler %d (standaard 18): ", i+1)); err == nil && n > 0 {
			cardCount = n
		}
		fmt.Printf("\nVoer de starthand van Speler %d in (%d kaarten):\n", i+1, cardCount)
		for {
			parsed, err := reader.ReadCards(fmt.Sprintf("Speler %d kaarten: ", i+1))
			if err != nil {
				fmt.Printf("Fout: %v\n", err)
				continue
			}
			if len(parsed) != cardCount {
				fmt.Printf("Verwacht %d, kreeg %d\n", cardCount, len(parsed))
				continue
			}
			hands[i] = NewHand(parsed)
			break
		}
	}
	var deadCards []Card
	if numPlayers == 2 {
		if reader.ReadYesNo("Dode kaarten invoeren?") {
			for {
				parsed, err := reader.ReadCards("Dode kaarten: ")
				if err != nil {
					fmt.Printf("Fout: %v\n", err)
					continue
				}
				deadCards = parsed
				break
			}
		}
	}
	gs := NewGameWithHands(hands, deadCards, 0)
	engConfig := DefaultConfig(numPlayers)
	engConfig.OmniscientMode = true
	iters := 3000
	if n, err := reader.ReadInt("Iteraties per zet (standaard 3000, meer = nauwkeuriger maar trager): "); err == nil && n > 0 {
		iters = n
	}
	engConfig.Iterations = iters
	engConfig.NumWorkers = cfg.numThreads
	analyzeStr := reader.ReadLine(fmt.Sprintf("Welke speler(s) analyseren? (bv. '1' of '1,3', leeg = alle %d spelers): ", numPlayers))
	analyzeAll := strings.TrimSpace(analyzeStr) == "" || strings.ToLower(strings.TrimSpace(analyzeStr)) == "alle"
	analyzePlayers := map[int]bool{}
	if !analyzeAll {
		for _, part := range strings.Split(analyzeStr, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && n >= 1 && n <= numPlayers {
				analyzePlayers[n-1] = true
			}
		}
		if len(analyzePlayers) == 0 {
			analyzeAll = true
		}
	}
	trackers := make([]*KnowledgeTracker, numPlayers)
	for p := 0; p < numPlayers; p++ {
		if analyzeAll || analyzePlayers[p] {
			trackers[p] = NewKnowledgeTracker(numPlayers, p, gs.Hands[p], gs.DeadCards)
		}
	}
	fmt.Println("\nVoer nu elke zet van het spel in.")
	fmt.Println("Formaat: 'speler:kaarten'  bv. '1:KK' of '2:-' (pas) of '1:11/444' (aas+vervolg)")
	fmt.Println("Zonder spelernummer gebruikt de engine de speler aan de beurt.")
	fmt.Println("Typ 'klaar' om te stoppen.")
	fmt.Println()
	moveNum := 0
	for !gs.GameOver {
		moveNum++
		fmt.Printf("--- Zet %d (Speler %d aan de beurt) ---\n", moveNum, gs.CurrentTurn+1)
		input := reader.ReadLine("Zet: ")
		if strings.ToLower(input) == "klaar" || strings.ToLower(input) == "done" {
			break
		}
		parts := strings.SplitN(input, ":", 2)
		playerStr := strings.TrimSpace(parts[0])
		cardsStr := ""
		if len(parts) > 1 {
			cardsStr = strings.TrimSpace(parts[1])
		} else {
			cardsStr = playerStr
			playerStr = strconv.Itoa(gs.CurrentTurn + 1)
		}
		playerNum, _ := strconv.Atoi(playerStr)
		playerID := playerNum - 1
		if playerID < 0 {
			playerID = gs.CurrentTurn
		}
		mainCardsStr, followCardsStr, hasFollowCards := strings.Cut(cardsStr, "/")
		mainCardsStr = strings.TrimSpace(mainCardsStr)
		mainCardsLower := strings.ToLower(mainCardsStr)
		var move Move
		if mainCardsLower == "pass" || mainCardsLower == "p" || mainCardsStr == "-" {
			move = Move{PlayerID: playerID, IsPass: true}
		} else {
			parsed, err := ParseCards(mainCardsStr)
			if err != nil {
				fmt.Printf("Fout: %v\n", err)
				moveNum--
				continue
			}
			move = Move{PlayerID: playerID, Cards: parsed}
		}
		doAnalysis := analyzeAll || analyzePlayers[playerID]
		var bestMove Move
		var bestEval MoveEval
		var actualDetail MoveDetail
		var bestLabel string
		if doAnalysis {
			tracker := trackers[playerID]
			eng := NewEngine(engConfig)
			bestMove, bestEval = eng.BestMove(gs, tracker)
			bestLabel = FormatMove(bestMove)
			if bestMove.ContainsReset() {
				gsClone := gs.Clone()
				gsClone.ApplyMove(bestMove)
				if !gsClone.GameOver && gsClone.CurrentTurn == playerID {
					bestFollow, _ := eng.BestMove(gsClone, tracker)
					bestLabel = fmt.Sprintf("%s / %s", FormatMove(bestMove), FormatMove(bestFollow))
				}
			}
			if d, ok := FindMoveInEval(bestEval, move); ok {
				actualDetail = d
			} else {
				actualDetail = eng.AnalyzeMove(gs, tracker, move)
			}
		}
		if err := gs.ValidateMove(move); err != nil {
			fmt.Printf("Ongeldige zet: %v\n", err)
			moveNum--
			continue
		}
		if move.IsPass {
			for p := 0; p < numPlayers; p++ {
				if trackers[p] != nil {
					trackers[p].RecordPass(move.PlayerID, gs.Round)
				}
			}
		}
		gs.ApplyMove(move)
		for p := 0; p < numPlayers; p++ {
			if trackers[p] != nil {
				trackers[p].RecordMove(move)
			}
		}
		moveLabel := FormatMove(move)
		if hasFollowCards && !gs.GameOver && gs.CurrentTurn == playerID {
			followCardsStr = strings.TrimSpace(followCardsStr)
			parsed, err := ParseCards(followCardsStr)
			if err != nil {
				fmt.Printf("⚠️  Fout in vervolg-zet: %v\n", err)
			} else {
				followMove := Move{PlayerID: playerID, Cards: parsed}
				if err2 := gs.ValidateMove(followMove); err2 != nil {
					fmt.Printf("⚠️  Ongeldige vervolg-zet: %v\n", err2)
				} else {
					gs.ApplyMove(followMove)
					for p := 0; p < numPlayers; p++ {
						if trackers[p] != nil {
							trackers[p].RecordMove(followMove)
						}
					}
					moveLabel = fmt.Sprintf("%s / %s", FormatMove(move), FormatMove(followMove))
				}
			}
		}
		if doAnalysis {
			forcedWin := bestEval.Score >= 1.0 && bestEval.Visits <= 1
			playedIsBest := MovesEqual(bestMove, move)
			var diff float64
			emoji := "✅"
			if !playedIsBest {
				diff = bestEval.Score - actualDetail.WinRate
				if forcedWin {
					emoji = "❌"
				} else if diff > 0.15 {
					emoji = "❌"
				} else if diff > 0.02 {
					emoji = "⚠️ "
				}
			}
			fmt.Printf("%s Gespeeld: %s (score: %.1f%%)\n", emoji, moveLabel, actualDetail.WinRate*100)
			if forcedWin && !playedIsBest {
				fmt.Printf("   ♟️  Gedwongen winst gemist! Beste was: %s\n", bestLabel)
			} else if forcedWin && playedIsBest {
				fmt.Printf("   ♟️  Gedwongen winst gevonden!\n")
			} else {
				showBest := !playedIsBest &&
					(diff > 0.02 || (bestEval.Score > 0.90 && diff > 0.005))
				if showBest {
					fmt.Printf("   Beste was: %s (score: %.1f%%, verschil: %.1f%%)\n",
						bestLabel, bestEval.Score*100, diff*100)
				}
			}
			// Diagnostiek: toon top alternatieven (gesorteerd op score, max 5)
			if len(bestEval.Details) > 1 {
				sorted := make([]MoveDetail, len(bestEval.Details))
				copy(sorted, bestEval.Details)
				for i := 0; i < len(sorted); i++ {
					for j := i + 1; j < len(sorted); j++ {
						if sorted[j].WinRate > sorted[i].WinRate {
							sorted[i], sorted[j] = sorted[j], sorted[i]
						}
					}
				}
				fmt.Printf("   Top: ")
				limit := len(sorted)
				if limit > 5 {
					limit = 5
				}
				for k := 0; k < limit; k++ {
					d := sorted[k]
					label := FormatMove(d.Move)
					marker := ""
					if MovesEqual(d.Move, move) {
						marker = "←"
					}
					if k > 0 {
						fmt.Printf(" | ")
					}
					fmt.Printf("%s %.1f%%%s", label, d.WinRate*100, marker)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("⏭️  Speler %d: %s\n", playerID+1, moveLabel)
		}
		if !gs.GameOver && gs.Finished[playerID] && gs.Hands[playerID].IsEmpty() {
			rank := gs.PlayerRank(playerID)
			medals := []string{"🥇", "🥈", "🥉"}
			m := ""
			if rank >= 0 && rank < len(medals) {
				m = medals[rank]
			}
			fmt.Printf("%s Speler %d eindigt op plaats %d!\n", m, playerID+1, rank+1)
		}
		fmt.Println()
	}
	if gs.GameOver {
		fmt.Println()
		printRanking(gs)
	}
	fmt.Println("\nAnalyse klaar.")
}

func quickAnalyzeMode(reader *Reader, cfg settings) {
	PrintHeader("Snelle Analyse")
	fmt.Println("Voer de partij in één keer in.")
	fmt.Println("Zetten: spatie-gescheiden tokens die alterneren tussen spelers.")
	fmt.Println("Aas+vervolg: schrijf als '1/5' (aas, dan 5 in dezelfde beurt).")
	fmt.Println("Pas: p of - of pass")
	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}
	analyzePlayer := 0
	if p, err := reader.ReadInt(fmt.Sprintf("Welke speler analyseren (1-%d): ", numPlayers)); err == nil && p >= 1 && p <= numPlayers {
		analyzePlayer = p - 1
	}
	hands := make([]*Hand, numPlayers)
	for i := 0; i < numPlayers; i++ {
		cardCount := 18
		if n, err := reader.ReadInt(fmt.Sprintf("Aantal startkaarten voor Speler %d (standaard 18): ", i+1)); err == nil && n > 0 {
			cardCount = n
		}
		for {
			parsed, err := reader.ReadCards(fmt.Sprintf("Speler %d kaarten (%d): ", i+1, cardCount))
			if err != nil {
				fmt.Printf("Fout: %v\n", err)
				continue
			}
			if len(parsed) != cardCount {
				fmt.Printf("Verwacht %d, kreeg %d\n", cardCount, len(parsed))
				continue
			}
			hands[i] = NewHand(parsed)
			break
		}
	}
	var deadCards []Card
	startPlayer := 0
	if p, err := reader.ReadInt(fmt.Sprintf("Wie begint (spelernummer 1-%d): ", numPlayers)); err == nil && p >= 1 && p <= numPlayers {
		startPlayer = p - 1
	}
	iters := 3000
	if n, err := reader.ReadInt("Iteraties per zet (standaard 3000): "); err == nil && n > 0 {
		iters = n
	}
	fmt.Println()
	fmt.Printf("Voer alle zetten in als spatie-gescheiden tokens (bv: 8888 p 33 44 66 jj p 4 5 9 1/5 ...)\n")
	movesLine := reader.ReadLine("Zetten: ")
	tokens := strings.Fields(movesLine)
	if len(tokens) == 0 {
		fmt.Println("Geen zetten ingevoerd.")
		return
	}
	gs := NewGameWithHands(hands, deadCards, startPlayer)
	engConfig := DefaultConfig(numPlayers)
	engConfig.OmniscientMode = true
	engConfig.Iterations = iters
	engConfig.NumWorkers = cfg.numThreads
	trackers := make([]*KnowledgeTracker, numPlayers)
	for p := 0; p < numPlayers; p++ {
		trackers[p] = NewKnowledgeTracker(numPlayers, p, gs.Hands[p], gs.DeadCards)
	}
	fmt.Println()
	moveNum := 0
	ti := 0
	for ti < len(tokens) && !gs.GameOver {
		token := tokens[ti]
		ti++
		moveNum++
		playerID := gs.CurrentTurn
		mainStr, followStr, hasFollow := strings.Cut(token, "/")
		mainLower := strings.ToLower(strings.TrimSpace(mainStr))
		var move Move
		if mainLower == "p" || mainLower == "pass" || mainLower == "-" {
			move = PassMove(playerID)
		} else {
			parsed, err := ParseCards(mainStr)
			if err != nil {
				fmt.Printf("⚠️  Token %d (%q): %v — overgeslagen\n", moveNum, token, err)
				continue
			}
			move = Move{PlayerID: playerID, Cards: parsed}
		}
		doAnalysis := playerID == analyzePlayer
		var bestMove Move
		var bestEval MoveEval
		var actualDetail MoveDetail
		var bestLabel string
		if doAnalysis {
			tracker := trackers[playerID]
			eng := NewEngine(engConfig)
			bestMove, bestEval = eng.BestMove(gs, tracker)
			bestLabel = FormatMove(bestMove)
			if bestMove.ContainsReset() {
				gsClone := gs.Clone()
				gsClone.ApplyMove(bestMove)
				if !gsClone.GameOver && gsClone.CurrentTurn == playerID {
					bestFollow, _ := eng.BestMove(gsClone, tracker)
					bestLabel = fmt.Sprintf("%s / %s", FormatMove(bestMove), FormatMove(bestFollow))
				}
			}
			if d, ok := FindMoveInEval(bestEval, move); ok {
				actualDetail = d
			} else {
				actualDetail = eng.AnalyzeMove(gs, tracker, move)
			}
		}
		if err := gs.ValidateMove(move); err != nil {
			fmt.Printf("⚠️  Token %d (%q): ongeldige zet: %v — overgeslagen\n", moveNum, token, err)
			continue
		}
		if move.IsPass {
			for p := 0; p < numPlayers; p++ {
				if trackers[p] != nil {
					trackers[p].RecordPass(move.PlayerID, gs.Round)
				}
			}
		}
		gs.ApplyMove(move)
		for p := 0; p < numPlayers; p++ {
			if trackers[p] != nil {
				trackers[p].RecordMove(move)
			}
		}
		moveLabel := FormatMove(move)
		if hasFollow && !gs.GameOver && gs.CurrentTurn == playerID {
			followStr = strings.TrimSpace(followStr)
			parsed, err := ParseCards(followStr)
			if err == nil {
				followMove := Move{PlayerID: playerID, Cards: parsed}
				if err2 := gs.ValidateMove(followMove); err2 == nil {
					gs.ApplyMove(followMove)
					for p := 0; p < numPlayers; p++ {
						if trackers[p] != nil {
							trackers[p].RecordMove(followMove)
						}
					}
					moveLabel = fmt.Sprintf("%s / %s", FormatMove(move), FormatMove(followMove))
				} else {
					fmt.Printf("⚠️  Vervolg-zet %q ongeldig: %v\n", followStr, err2)
				}
			} else {
				fmt.Printf("⚠️  Vervolg-zet %q fout: %v\n", followStr, err)
			}
		}
		if doAnalysis {
			forcedWin := bestEval.Score >= 1.0 && bestEval.Visits <= 1
			playedIsBest := MovesEqual(bestMove, move)
			var diff float64
			emoji := "✅"
			if !playedIsBest {
				diff = bestEval.Score - actualDetail.WinRate
				if forcedWin {
					emoji = "❌"
				} else if diff > 0.15 {
					emoji = "❌"
				} else if diff > 0.02 {
					emoji = "⚠️ "
				}
			}
			fmt.Printf("%s Z%d P%d: %s (score: %.1f%%)\n", emoji, moveNum, playerID+1, moveLabel, actualDetail.WinRate*100)
			if forcedWin && !playedIsBest {
				fmt.Printf("   ♟️  Gedwongen winst gemist! Beste was: %s\n", bestLabel)
			} else if forcedWin && playedIsBest {
				fmt.Printf("   ♟️  Gedwongen winst gevonden!\n")
			} else {
				showBest := !playedIsBest && (diff > 0.02 || (bestEval.Score > 0.90 && diff > 0.005))
				if showBest {
					fmt.Printf("   Beste was: %s (score: %.1f%%, verschil: %.1f%%)\n",
						bestLabel, bestEval.Score*100, diff*100)
				}
			}
			// Diagnostiek: toon top alternatieven (gesorteerd op score, max 5)
			if len(bestEval.Details) > 1 {
				sorted := make([]MoveDetail, len(bestEval.Details))
				copy(sorted, bestEval.Details)
				for i := 0; i < len(sorted); i++ {
					for j := i + 1; j < len(sorted); j++ {
						if sorted[j].WinRate > sorted[i].WinRate {
							sorted[i], sorted[j] = sorted[j], sorted[i]
						}
					}
				}
				fmt.Printf("   Top: ")
				limit := len(sorted)
				if limit > 5 {
					limit = 5
				}
				for k := 0; k < limit; k++ {
					d := sorted[k]
					label := FormatMove(d.Move)
					marker := ""
					if MovesEqual(d.Move, move) {
						marker = "←"
					}
					if k > 0 {
						fmt.Printf(" | ")
					}
					fmt.Printf("%s %.1f%%%s", label, d.WinRate*100, marker)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("⏭️  Z%d P%d: %s\n", moveNum, playerID+1, moveLabel)
		}
	}
	fmt.Println()
	if gs.GameOver {
		printRanking(gs)
	} else {
		fmt.Printf("Partij gestopt na %d zetten (spel nog niet voorbij).\n", moveNum)
	}
	fmt.Println("\nSnelle analyse klaar.")
}

func simulateMode(reader *Reader, cfg settings) {
	PrintHeader("Simulatie Modus")
	fmt.Println("Kijk hoe de engine tegen zichzelf speelt!")
	fmt.Println()
	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}
	sims := 1000
	if s, err := reader.ReadInt("Engine-simulaties per zet (standaard 1000): "); err == nil && s > 0 {
		sims = s
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gs := NewGame(numPlayers, rng, 0)
	fmt.Println("\nStarthanden:")
	for i := 0; i < numPlayers; i++ {
		fmt.Printf("Speler %d: %s\n", i+1, gs.Hands[i])
	}
	fmt.Println()
	trackers := make([]*KnowledgeTracker, numPlayers)
	engines := make([]*Engine, numPlayers)
	for i := 0; i < numPlayers; i++ {
		engConfig := DefaultConfig(numPlayers)
		engConfig.Iterations = sims
		engConfig.NumWorkers = cfg.numThreads
		trackers[i] = NewKnowledgeTracker(numPlayers, i, gs.Hands[i], gs.DeadCards)
		engines[i] = NewEngine(engConfig)
	}
	prevFinished := 0
	moveNum := 0
	for !gs.GameOver {
		moveNum++
		playerID := gs.CurrentTurn
		eng := engines[playerID]
		bestMove, eval := eng.BestMove(gs, trackers[playerID])
		fmt.Printf("Zet %d | Speler %d: %s (score: %.1f%%) | Kaarten:",
			moveNum, playerID+1, FormatMove(bestMove), eval.Score*100)
		for i := 0; i < numPlayers; i++ {
			if gs.Finished[i] {
				fmt.Printf(" P%d:✓", i+1)
			} else {
				fmt.Printf(" P%d:%d", i+1, gs.Hands[i].Count())
			}
		}
		fmt.Println()
		if bestMove.IsPass {
			for i := 0; i < numPlayers; i++ {
				trackers[i].RecordPass(bestMove.PlayerID, gs.Round)
			}
		}
		gs.ApplyMove(bestMove)
		for i := 0; i < numPlayers; i++ {
			trackers[i].RecordMove(bestMove)
		}
		nowFinished := len(gs.Ranking)
		if nowFinished > prevFinished {
			medals := []string{"🥇", "🥈", "🥉", "4e"}
			for pos := prevFinished; pos < nowFinished && !gs.GameOver; pos++ {
				m := ""
				if pos < len(medals) {
					m = medals[pos]
				}
				fmt.Printf("  %s Speler %d eindigt op plaats %d!\n",
					m, gs.Ranking[pos]+1, pos+1)
			}
			prevFinished = nowFinished
		}
		if moveNum > 600 {
			fmt.Println("Spel overschreed 600 zetten, gestopt.")
			break
		}
	}
	if gs.GameOver {
		PrintHeader("Spel Voorbij!")
		printRanking(gs)
	}
}

func printRanking(gs *GameState) {
	medals := []string{"🥇", "🥈", "🥉", "4️⃣ "}
	labels := []string{"wint!", "wordt 2e", "wordt 3e", "wordt 4e (verliezer)"}
	for i, pid := range gs.Ranking {
		m := ""
		if i < len(medals) {
			m = medals[i]
		}
		lbl := ""
		if i < len(labels) {
			lbl = labels[i]
		}
		if i == len(gs.Ranking)-1 && gs.NumPlayers > 2 {
			lbl = "verliest 💀"
		}
		fmt.Printf("%s Speler %d %s\n", m, pid+1, lbl)
	}
}

// ═══════════════════════════════════════════════════════════════
// WEIGHT TUNER v2.1 — met Elitism + Adaptive Mutation (krachtige PC)
// ═══════════════════════════════════════════════════════════════

func weightTunerMode(reader *Reader, cfg settings) {
	PrintHeader("Weight Tuner v2.1 — Elitism Edition")
	fmt.Println("Zeer sterke optimalisatie met elitism en adaptieve mutatie.")
	fmt.Println()

	games, _ := reader.ReadInt("Games per matchup (aanbevolen 600-1200): ")
	if games < 100 { games = 800 }
	generations, _ := reader.ReadInt("Aantal generaties (aanbevolen 25-60): ")
	if generations < 10 { generations = 35 }
	iters, _ := reader.ReadInt("Iteraties per zet (aanbevolen 8000-15000): ")
	if iters < 2000 { iters = 10000 }

	fmt.Printf("\n🚀 Start TUNER v2.1\n")
	fmt.Printf("Games: %d | Generaties: %d | Iters: %d | Threads: %d\n\n", 
		games, generations, iters, cfg.numThreads)

	current, _ := LoadWeights("weights.json")
	best := current
	bestScore := 0.0

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for gen := 1; gen <= generations; gen++ {
		fmt.Printf("Generatie %2d/%d  ─  Beste score tot nu: %.2f%%\n", gen, generations, bestScore*100)

		// Elitism: beste altijd behouden
		candidates := []Weights{best}

		// 15 mutants
		for m := 0; m < 15; m++ {
			mutStrength := 0.22
			if float64(gen) > float64(generations)*0.6 {
				mutStrength = 0.09 // later fijner tunen
			}
			mutant := perturbWeights(best, rng, mutStrength)
			score := evaluateWeights(mutant, games, iters, cfg.numThreads, rng)

			candidates = append(candidates, mutant)

			if score > bestScore {
				best = mutant
				bestScore = score
				fmt.Printf("   🔥 NIEUWE BESTE! %.2f%% (mutant %d)\n", score*100, m+1)
			}
		}

		// Random restart elke 6 generaties
		if gen%6 == 0 && gen < generations {
			fmt.Println("   🔄 Random restart (ontsnapt aan lokaal maximum)")
			best = perturbWeights(best, rng, 0.45)
		}
	}

	SaveWeights(best, "weights.json")
	fmt.Printf("\n🏆 TUNING AFGEROND!\n")
	fmt.Printf("Beste score: %.2f%%\n", bestScore*100)
	fmt.Println("Gewichten opgeslagen in weights.json")
	fmt.Println("Je kunt nu direct met de verbeterde AI spelen.")
}

func perturbWeights(base Weights, rng *rand.Rand, strength float64) Weights {
	w := base
	w.AceBonus           = clamp(w.AceBonus          *(1 + strength*(rng.Float64()*2-1)), 0.08, 0.85)
	w.WildBonus          = clamp(w.WildBonus         *(1 + strength*(rng.Float64()*2-1)), 0.08, 0.65)
	w.SynergyBonus       = clamp(w.SynergyBonus      *(1 + strength*(rng.Float64()*2-1)), 0.02, 0.45)
	w.CardDiffWeight     = clamp(w.CardDiffWeight    *(1 + strength*(rng.Float64()*2-1)), 0.02, 0.30)
	w.KingPenalty        = clamp(w.KingPenalty       *(1 + strength*(rng.Float64()*2-1)), 0.01, 0.18)
	w.QueenPenalty       = clamp(w.QueenPenalty      *(1 + strength*(rng.Float64()*2-1)), 0.01, 0.15)
	w.IsolatedLowPenalty = clamp(w.IsolatedLowPenalty*(1 + strength*(rng.Float64()*2-1)), 0.01, 0.18)
	w.ClusterBonus       = clamp(w.ClusterBonus      *(1 + strength*(rng.Float64()*2-1)), 0.01, 0.20)
	w.TempoBonus         = clamp(w.TempoBonus        *(1 + strength*(rng.Float64()*2-1)), 0.02, 0.30)
	w.AcePlayFactor      = clamp(w.AcePlayFactor     *(1 + strength*(rng.Float64()*2-1)), 0.15, 1.3)
	w.WildPlayFactor     = clamp(w.WildPlayFactor    *(1 + strength*(rng.Float64()*2-1)), 0.15, 1.1)
	w.SynergyPenalty     = clamp(w.SynergyPenalty    *(1 + strength*(rng.Float64()*2-1)), 0.15, 1.1)
	w.RankPreference     = clamp(w.RankPreference    *(1 + strength*(rng.Float64()*2-1)), 0.02, 0.45)
	w.PassBase           = clamp(w.PassBase          *(1 + strength*(rng.Float64()*2-1)), 0.02, 0.35)
	w.PassSpecialFactor  = clamp(w.PassSpecialFactor *(1 + strength*(rng.Float64()*2-1)), 0.05, 0.65)
	w.PassBehindFactor   = clamp(w.PassBehindFactor  *(1 + strength*(rng.Float64()*2-1)), 0.10, 0.85)
	w.UrgencyPenalty     = clamp(w.UrgencyPenalty    *(1 + strength*(rng.Float64()*2-1)), 0.02, 0.25)
	return w
}

func evaluateWeights(w Weights, games int, iters int, threads int, rng *rand.Rand) float64 {
	config := DefaultConfig(2)
	config.Iterations = iters
	config.NumWorkers = threads
	config.Weights = w
	config.OmniscientMode = true

	wins := 0
	for g := 0; g < games; g++ {
		gs := NewGame(2, rng, rng.Intn(2)) // random startspeler
		t1 := NewKnowledgeTracker(2, 0, gs.Hands[0], gs.DeadCards)
		t2 := NewKnowledgeTracker(2, 1, gs.Hands[1], gs.DeadCards)
		e1 := NewEngine(config)
		e2 := NewEngine(config)

		for !gs.GameOver {
			pid := gs.CurrentTurn
			var eng *Engine
			var tr *KnowledgeTracker
			if pid == 0 {
				eng, tr = e1, t1
			} else {
				eng, tr = e2, t2
			}
			move, _ := eng.BestMove(gs, tr)
			gs.ApplyMove(move)
			t1.RecordMove(move)
			t2.RecordMove(move)
		}

		if gs.Ranking[0] == 1 { // speler 2 wint
			wins++
		}
	}
	return float64(wins) / float64(games)
}

// Onderdruk "declared but not used" voor hulpfuncties die enkel door de tuner gebruikt worden
var _ = clamp
var _ = SaveWeights
var _ = SaveGame
var _ = LoadGame
var _ = EvaluateHand
var _ = QuickEvaluateMove
var _ = ShouldPass
var _ = gatherSpecials
