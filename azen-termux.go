// azen-termux.go
// Standalone single-file versie van de AZEN engine voor Termux/Android.
// Bevat alle code in één bestand zonder externe afhankelijkheden.
//
// Compileer: go build azen-termux.go
// Starten:   go run azen-termux.go

package main

import (
	"bufio"
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
		if gs.Round.IsOpen {
			return fmt.Errorf("cannot pass in an open round — must play")
		}
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
	_, _, _, err := classifyCards(m.Cards)
	return err
}

func (gs *GameState) validateResponsePlay(m Move) error {
	_, hasNormal, normalRank, err := classifyCards(m.Cards)
	if err != nil {
		return err
	}
	if len(m.Cards) != gs.Round.Count {
		return fmt.Errorf("moet exact %d kaart(en) spelen (gespeeld: %d)", gs.Round.Count, len(m.Cards))
	}
	// Rank-check geldt altijd als er normale kaarten aanwezig zijn — ook met joker.
	// Puur joker+wildcards hoeft de rank niet te verslaan (joker reset altijd).
	if hasNormal && normalRank != 0 && normalRank <= gs.Round.TableRank {
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
	if gs.Round.IsOpen {
		return genOpenMoves(pid, hand)
	}
	moves := []Move{PassMove(pid)}
	moves = append(moves, genResponseMoves(pid, hand, gs.Round)...)
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
		maxTotal := len(normals) + len(wilds)
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

	for total := 1; total <= len(wilds); total++ {
		for _, wc := range combos(wilds, total) {
			moves = append(moves, Move{PlayerID: pid, Cards: wc})
		}
	}

	moves = append(moves, genResetMoves(pid, resets, wilds)...)

	// Joker + normale kaarten (± wildcards) van dezelfde rank (bv. "70", "044", "444422200")
	for _, rank := range NormalRanks() {
		normals := byRank[rank]
		if len(normals) == 0 {
			continue
		}
		for numReset := 1; numReset <= len(resets); numReset++ {
			for numNorm := 1; numNorm <= len(normals); numNorm++ {
				for _, rc := range combos(resets, numReset) {
					for _, nc := range combos(normals, numNorm) {
						// Zonder wilds
						merged := append(append([]Card{}, rc...), nc...)
						moves = append(moves, Move{PlayerID: pid, Cards: merged})
						// Met wildcards erbij
						for numWild := 1; numWild <= len(wilds); numWild++ {
							for _, wc := range combos(wilds, numWild) {
								full := append(append(append([]Card{}, rc...), nc...), wc...)
								moves = append(moves, Move{PlayerID: pid, Cards: full})
							}
						}
					}
				}
			}
		}
	}

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

	// Joker + normale kaarten (± wildcards) als antwoord — normale rank moet tafel verslaan
	for _, rank := range NormalRanks() {
		if rank <= tableRank {
			continue
		}
		normals := hand.GetByRank(rank)
		if len(normals) == 0 {
			continue
		}
		for numReset := 1; numReset <= imin(len(resets), need); numReset++ {
			for numNorm := 1; numNorm <= imin(len(normals), need-numReset); numNorm++ {
				numWild := need - numReset - numNorm
				if numWild < 0 || numWild > len(wilds) {
					continue
				}
				for _, rc := range combos(resets, numReset) {
					for _, nc := range combos(normals, numNorm) {
						if numWild == 0 {
							merged := append(append([]Card{}, rc...), nc...)
							moves = append(moves, Move{PlayerID: pid, Cards: merged})
						} else {
							for _, wc := range combos(wilds, numWild) {
								merged := append(append(append([]Card{}, rc...), nc...), wc...)
								moves = append(moves, Move{PlayerID: pid, Cards: merged})
							}
						}
					}
				}
			}
		}
	}

	return dedup(moves)
}

func genResetMoves(pid int, resets, wilds []Card) []Move {
	var moves []Move
	for numReset := 1; numReset <= len(resets); numReset++ {
		maxW := len(wilds)
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
	kt.updateSuspicions(m.Cards, m.PlayerID)
}

func (kt *KnowledgeTracker) RecordPass(passerID int, round RoundState) {
	if passerID == kt.MyPlayerID {
		return
	}
	if round.IsOpen {
		return
	}
	// Alleen loggen bij ≤12 kaarten; bij heel veel kaarten is passen nog strategisch
	if kt.HandCounts[passerID] > 12 {
		return
	}
	// Accepteer zowel singles als pairs — wie past op een pair heeft dat pair niet
	if round.Count < 1 || round.Count > 2 {
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
		// Joker: bij een pass op single altijd reset-mogelijkheid gemist → exclude.
		// Bij pair: joker helpt niet direct → niet excluden.
		if pr.Count == 1 {
			excluded[RankJoker] = true
		}
		// Wildcard (2): alleen excluden bij hoge tafel (≥ Queen).
		// Bij lage tafel is het slim om wild te bewaren — geen bewijs dat je er geen hebt.
		if pr.Count == 1 && pr.TableRank >= RankQueen {
			excluded[RankTwo] = true
		}
		// Bij single-pass: sluit hogere ranks uit (ze hadden die kunnen spelen)
		// Bij pair-pass: NIET excluden — ze kunnen singles van hoge ranks hebben,
		// alleen geen PAIRS. De exclusie-map is binair en kan dat onderscheid niet maken.
		if pr.Count == 1 {
			for _, r := range NormalRanks() {
				if r > pr.TableRank {
					excluded[r] = true
				}
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

func (kt *KnowledgeTracker) updateSuspicions(played []Card, playerID int) {
	newlyPlayed := map[Rank]int{}
	for _, c := range played {
		newlyPlayed[c.Rank]++
	}

	// Step 1: The player who played these cards definitely used them — reduce their suspicion
	// by exactly the cards played (they may still have more copies of the same rank).
	if playerID != kt.MyPlayerID {
		for rank, n := range newlyPlayed {
			toRemove := n
			var newSusp []Card
			for _, c := range kt.Suspicions[playerID] {
				if c.Rank == rank && toRemove > 0 {
					toRemove--
				} else {
					newSusp = append(newSusp, c)
				}
			}
			kt.Suspicions[playerID] = newSusp
		}
	}

	// Step 2: Pool check — trim any remaining suspicion (across all opponents) only when the
	// available pool for a rank is truly exhausted. CardsPlayed was already updated in
	// RecordMove before this call, so PossibleOpponentCards() reflects the current state.
	pool := kt.PossibleOpponentCards()
	poolCount := map[Rank]int{}
	for _, c := range pool {
		poolCount[c.Rank]++
	}

	for rank := range newlyPlayed {
		totalSusp := 0
		for _, suspected := range kt.Suspicions {
			for _, c := range suspected {
				if c.Rank == rank {
					totalSusp++
				}
			}
		}
		for totalSusp > poolCount[rank] {
			removed := false
			for pid := range kt.Suspicions {
				for i, c := range kt.Suspicions[pid] {
					if c.Rank == rank {
						kt.Suspicions[pid] = append(kt.Suspicions[pid][:i], kt.Suspicions[pid][i+1:]...)
						totalSusp--
						removed = true
						break
					}
				}
				if removed {
					break
				}
			}
			if !removed {
				break
			}
		}
	}

	// Step 3: Exclusions — reduce counts for ranks that were played.
	for pid, exclMap := range kt.Exclusions {
		if exclMap == nil {
			continue
		}
		for rank, count := range exclMap {
			n := newlyPlayed[rank]
			if n > 0 && count > 0 {
				newCount := count - n
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

// KnownOpponentCards geeft de kaarten terug die de tracker zeker weet dat speler playerID heeft.
// Drie gevallen:
//  1. Gok volledig: de gebruiker heeft alle kaarten van playerID ingevoerd via gok.
//  2. Eliminatie (1 actieve tegenstander): pool = precies hun hand.
//  3. Deductie: alle andere actieve tegenstanders hebben volledig bekende handen (gok) →
//     pool minus die handen = hand van playerID.
// Geeft nil terug als deductie niet mogelijk is.
func (kt *KnowledgeTracker) KnownOpponentCards(playerID int) []Card {
	if playerID == kt.MyPlayerID || kt.HandCounts[playerID] == 0 {
		return nil
	}

	// Geval 1: alle kaarten van playerID zijn ingevoerd via gok.
	if len(kt.Suspicions[playerID]) == kt.HandCounts[playerID] {
		return kt.Suspicions[playerID]
	}

	// Geval 2 & 3: trek bekende handen van alle andere actieve tegenstanders af van de pool.
	// Als er geen andere actieve tegenstanders zijn (geval 2), geldt pool = hand van playerID.
	pool := kt.PossibleOpponentCards()
	remaining := map[Rank]int{}
	for _, c := range pool {
		remaining[c.Rank]++
	}
	for p, count := range kt.HandCounts {
		if p == kt.MyPlayerID || p == playerID || count == 0 {
			continue
		}
		// p moet volledig bekend zijn via gok, anders kan je playerID niet afleiden.
		if len(kt.Suspicions[p]) != count {
			return nil
		}
		for _, c := range kt.Suspicions[p] {
			remaining[c.Rank]--
			if remaining[c.Rank] < 0 {
				return nil // inconsistentie in gok-invoer
			}
		}
	}
	var deduced []Card
	for _, r := range append(NormalRanks(), RankTwo, RankJoker) {
		for i := 0; i < remaining[r]; i++ {
			deduced = append(deduced, Card{Rank: r})
		}
	}
	if len(deduced) != kt.HandCounts[playerID] {
		return nil
	}
	return deduced
}

// AllOpponentHandsKnown geeft true als alle actieve tegenstanders volledig bekend zijn
// (via gok of door deductie). Dit maakt exacte schaakmat-detectie mogelijk in speelmodus.
func (kt *KnowledgeTracker) AllOpponentHandsKnown() bool {
	for p, count := range kt.HandCounts {
		if p == kt.MyPlayerID || count == 0 {
			continue
		}
		if kt.KnownOpponentCards(p) == nil {
			return false
		}
	}
	return true
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
	normalsUsed := 0
	for _, c := range move.Cards {
		if c.IsWild() {
			wildsUsed++
		} else if c.IsReset() {
			resetsUsed++
		} else {
			normalsUsed++
		}
	}
	effectiveRank := move.EffectiveRank(gs.Round.TableRank)

	// Wild-verspilling: scherpere straf naarmate de rank lager is.
	// Wild op rank 3-5 is catastrofaal; op rank 6-9 is slecht; op 10+ is acceptabel.
	// UITZONDERING: als de combo ook een reset bevat (joker+wildcard kill-shot) is de straf mild.
	if wildsUsed > 0 {
		if resetsUsed > 0 {
			// joker + wildcards als response = kill-shot (bijv. x220): nauwelijks straf
			mq.Score -= float64(wildsUsed) * 1.0
		} else if effectiveRank <= RankFive {
			mq.Score -= float64(wildsUsed) * 12.0
			mq.WastesWilds = true
			mq.Reasoning = "Wastes wildcards on very low play"
		} else if effectiveRank < RankTen {
			mq.Score -= float64(wildsUsed) * 7.0
			mq.WastesWilds = true
			mq.Reasoning = "Wastes wildcards on low play"
		} else {
			mq.Score -= float64(wildsUsed) * 2.0 // mild; wild op K is prima
		}
	}

	if resetsUsed > 0 {
		mq.Score += 5.0
		mq.WastesAces = resetsUsed > 1
		if mq.WastesAces {
			mq.Score -= float64(resetsUsed-1) * 8.0
			mq.Reasoning = "Uses multiple resets unnecessarily"
		}
	}

	// Response-context: bij een response-ronde is de LAAGSTE winnende zet het best.
	// Je wilt sterke kaarten bewaren voor later. Straf proportioneel aan "overshoot".
	// Uitzondering 1: als je na de zet nog maar 1 kaart overhoudt (eindspel 2-kaarten).
	// Uitzondering 2: tegenstander heeft nog maar 1 kaart — die is sowieso zijn sterkste.
	//   Speel dan je hoogste kaart om hem te dwingen tot passen of een speciale kaart.
	if !gs.Round.IsOpen && effectiveRank > 0 {
		overshoot := float64(effectiveRank-gs.Round.TableRank) - 1.0
		if overshoot < 0 {
			overshoot = 0
		}
		opponentHasOne := false
		if gs.NumPlayers == 2 {
			opponentID := 1 - gs.CurrentTurn
			opponentHasOne = gs.Hands[opponentID].Count() == 1
		}
		if cardsAfter == 1 || opponentHasOne {
			mq.Score += overshoot * 2.0 // eindspel: hogere kaart = beter
		} else {
			mq.Score -= overshoot * 3.0 // bijv. Queen op een 6 tafel = -15 (overshoot 5)
		}
	}

	// Open ronde: lage kaarten dumpen is goed — MAAR alleen als je daarna
	// nog ≥3 kaarten overhoudt. Als je na de zet ≤2 kaarten overhoudt, is
	// het spelen van hoge kaarten (bijv. QQQ) juist de WIN-strategie en moet
	// de straf onderdrukt worden (anders wint "5" onterecht van "QQQ").
	if gs.Round.IsOpen && effectiveRank > 0 && cardsAfter >= 3 {
		rankValue := float64(effectiveRank-RankThree) / float64(RankAce-RankThree)
		mq.Score -= rankValue * 8.0 // hoge kaarten in open ronde = verspilling
	}

	// Extra straf: wildcards toevoegen aan lage combo in open ronde (bijv. 33322).
	// Lage rank + wildcards = je benut de wildcard niet strategisch: de tegenstander
	// beantwoordt even makkelijk met een grotere combo. De wildcards zijn meer waard
	// als kill-shot of als respons op sterke tafels.
	if gs.Round.IsOpen && wildsUsed > 0 && resetsUsed == 0 && effectiveRank <= RankSeven {
		mq.Score -= float64(wildsUsed) * 9.0
	}

	// Meerdere kaarten tegelijk kwijtraken is goed.
	mq.Score += float64(len(move.Cards)) * 3.0

	// Paar-breek penalty: als je een paar breekt om een single te spelen, is dat slecht.
	if normalsUsed == 1 && wildsUsed == 0 && resetsUsed == 0 {
		for _, c := range move.Cards {
			if hand.CountRank(c.Rank) >= 2 {
				mq.Score -= 5.0 // breekt een cluster
				break
			}
		}
	}

	// Win-threat bonus: schaalbaar naargelang hoeveel kaarten er overblijven.
	// Hoe dichter bij winst, hoe groter de bonus — zodat QQQ (1 kaart over)
	// duidelijk wint van 5 (3 kaarten over) in de heuristische evaluatie.
	switch {
	case cardsAfter == 1:
		mq.Score += 25.0
		mq.CreatesWinThreat = true
	case cardsAfter == 2:
		mq.Score += 18.0
		mq.CreatesWinThreat = true
	case cardsAfter == 3:
		mq.Score += 10.0
		mq.CreatesWinThreat = true
	case cardsAfter == 4:
		mq.Score += 4.0
	}
	if gs.Round.IsOpen && resetsUsed > 0 {
		mq.Score += 10.0
	}

	// Drain strategie: geïsoleerde single in open ronde bij 2 spelers.
	// Door een single te spelen dwing je de tegenstander één kaart uit zijn combo te breken.
	// Dit is sterker dan je eigen quad spelen als je nog 2+ combos in hand hebt.
	if gs.Round.IsOpen && gs.NumPlayers == 2 &&
		len(move.Cards) == 1 && normalsUsed == 1 && wildsUsed == 0 && resetsUsed == 0 &&
		cardsAfter >= 6 {
		rankCount := hand.CountRank(effectiveRank)
		if rankCount == 1 {
			// Kaart is geïsoleerd (geen partner). Tel combos die overblijven.
			comboCount := 0
			for r := RankThree; r <= RankAce; r++ {
				if r == effectiveRank {
					continue
				}
				if hand.CountRank(r) >= 2 {
					comboCount++
				}
			}
			remWilds := hand.CountRank(RankTwo)
			remResets := hand.CountRank(RankJoker)
			if comboCount >= 2 || (comboCount >= 1 && (remWilds+remResets) >= 1) {
				mq.Score += 11.0 // drain bonus: forceer tegenstander zijn combo te breken
			}
		}
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
	Iterations     int           // maximum iteraties (hard stop)
	MinIterations  int           // minimum iteraties (altijd uitvoeren, ook na deadline)
	MaxTime        time.Duration
	ExploreConst   float64
	NumPlayers     int
	OmniscientMode bool
	NumWorkers     int
	PressureTarget int // -1 = geen druk; >= 0 = speler-ID die moet verliezen
}

func DefaultConfig(numPlayers int) Config {
	return Config{
		Iterations:     50000,
		MinIterations:  500,
		MaxTime:        0,
		ExploreConst:   1.4,
		NumPlayers:     numPlayers,
		NumWorkers:     2,
		PressureTarget: -1,
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
	Score          float64
	Visits         int
	Details        []MoveDetail
	ForcedWinDepth int     // >0 als gedwongen winst: aantal eigen beurten tot winst
	TotalIters     int     // werkelijk aantal MCTS-iteraties uitgevoerd
	ElapsedMs      float64 // rekentijd in milliseconden
}

func (me MoveEval) String() string {
	return fmt.Sprintf("Win%%: %.1f%% (%d visits)", me.Score*100, me.Visits)
}

// StatsString geeft een korte string met iteraties en snelheid, bv. "8.4k iter | 4.2k/s".
func (me MoveEval) StatsString() string {
	if me.TotalIters == 0 {
		return "instant"
	}
	kIter := float64(me.TotalIters) / 1000.0
	var speed string
	if me.ElapsedMs > 0 {
		kps := float64(me.TotalIters) / me.ElapsedMs // k iter/s = iter / ms
		speed = fmt.Sprintf(" | %.1fk/s", kps)
	}
	return fmt.Sprintf("%.1fk iter%s", kIter, speed)
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
// via minimax. Bij ≤12 totale kaarten doorzoekt het ALLE mogelijke antwoorden
// van de tegenstander. Retourneert de eerste zet van het winnende pad en het
// aantal eigen beurten tot winst (1 = directe win, 2 = 2-staps combo, etc.).
func findImmediateWin(gs *GameState, knownHands bool) (*Move, int) {
	pid := gs.CurrentTurn
	handCount := gs.Hands[pid].Count()
	moves := gs.GetLegalMoves()

	// Snelle check: 1-zet win (geen Clone nodig)
	for _, m := range moves {
		if !m.IsPass && len(m.Cards) == handCount {
			mv := m
			return &mv, 1
		}
	}

	// Minimax forced-win zoektocht is alleen betrouwbaar als alle handen
	// bekend zijn (OmniscientMode / analyse). In play-mode bevatten de
	// tegenstander-handen placeholder-kaarten (rank 0) die nooit kunnen
	// antwoorden, waardoor elke zet vals als gedwongen winst wordt gezien.
	if !knownHands {
		return nil, 0
	}

	// Minimax forced-win zoektocht voor diepere schaakmatten
	totalCards := 0
	for _, h := range gs.Hands {
		totalCards += h.Count()
	}
	// Zodra iemand gewonnen heeft zijn alle overblijvende handen exact bekend (in OmniscientMode).
	// Dan kan de schaakmat-zoektocht altijd uitgevoerd worden, ongeacht het kaartaantal.
	// Zonder winnaar: harde grens van 12 kaarten om exponentiële blowup te voorkomen.
	someoneFinished := knownHands && gs.Winner != -1
	if !someoneFinished && totalCards > 12 {
		return nil, 0
	}
	// Adaptieve diepte: bij meer kaarten minder diep zoeken (bredere boom)
	maxDepth := totalCards * 3
	if totalCards <= 8 {
		maxDepth = totalCards * 4
	}
	nodeLimit := 500000
	if knownHands {
		nodeLimit = 2000000
	}
	nodes := 0

	// Probeer niet-pass zetten eerst (sneller naar winst)
	bestDepth := -1
	var bestMove *Move
	for _, m := range moves {
		if m.IsPass {
			continue
		}
		sim := gs.Clone()
		sim.ApplyMove(m)
		if notLastPlace(sim, pid) {
			mv := m
			return &mv, 1
		}
		d := forcedWinDepth(sim, pid, maxDepth-1, &nodes, nodeLimit)
		if d >= 0 {
			myMoves := d + 1 // +1 voor deze zet
			if bestMove == nil || myMoves < bestDepth {
				bestDepth = myMoves
				mv := m
				bestMove = &mv
			}
		}
	}
	if bestMove != nil {
		return bestMove, bestDepth
	}
	return nil, 0
}

// notLastPlace retourneert true als het spel voorbij is en myID NIET als laatste geëindigd is.
// Dit is de correcte win-definitie in meerspelersspellen: de verliezer is de laatste speler,
// iedereen daarvoor "wint" (ook 2e of 3e plaats). gs.Winner alleen geeft de 1e-plaatswinnaar.
func notLastPlace(gs *GameState, myID int) bool {
	if !gs.GameOver {
		return false
	}
	rank := gs.PlayerRank(myID)
	return rank >= 0 && rank < gs.NumPlayers-1
}

// forcedWinDepth bepaalt via minimax het aantal eigen beurten tot gedwongen
// winst. Retourneert -1 als geen forced win, of ≥0 (het aantal resterende
// eigen beurten). Bij onze beurt telt elke zet als +1. Bij tegenstander telt
// het niet mee (hun zet kost ons geen beurt), maar we nemen het worst-case pad.
func forcedWinDepth(gs *GameState, myID int, depth int, nodes *int, maxNodes int) int {
	*nodes++
	if *nodes > maxNodes {
		return -1
	}
	if gs.GameOver {
		if notLastPlace(gs, myID) {
			return 0
		}
		return -1
	}
	if depth <= 0 {
		return -1
	}

	moves := gs.GetLegalMoves()

	if gs.CurrentTurn == myID {
		// Onze beurt: zoek de snelste geforceerde winst
		best := -1
		// Probeer speel-zetten eerst
		for _, m := range moves {
			if m.IsPass {
				continue
			}
			sim := gs.Clone()
			sim.ApplyMove(m)
			d := forcedWinDepth(sim, myID, depth-1, nodes, maxNodes)
			if d >= 0 {
				total := d + 1 // +1 want wij speelden een zet
				if best < 0 || total < best {
					best = total
				}
			}
		}
		if best >= 0 {
			return best
		}
		// PASS alleen in response-rondes
		if !gs.Round.IsOpen {
			for _, m := range moves {
				if !m.IsPass {
					continue
				}
				sim := gs.Clone()
				sim.ApplyMove(m)
				d := forcedWinDepth(sim, myID, depth-1, nodes, maxNodes)
				if d >= 0 {
					return d // PASS kost ons geen "beurt" in de telling
				}
			}
		}
		return -1
	}

	// Tegenstander: ALLE zetten moeten naar onze winst leiden, neem worst-case
	worst := 0
	for _, m := range moves {
		sim := gs.Clone()
		sim.ApplyMove(m)
		d := forcedWinDepth(sim, myID, depth-1, nodes, maxNodes)
		if d < 0 {
			return -1 // tegenstander heeft een ontsnapping
		}
		if d > worst {
			worst = d // worst-case pad (tegenstander vertraagt maximaal)
		}
	}
	return worst
}

type workerResult struct {
	visits map[string]int
	wins   map[string]float64
	moves  map[string]Move
	iters  int
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

	// Bepaal de hoogste effectieve rank bereikbaar via:
	// 1. maxNaturalRank  = alleen normale kaarten (geen wild, geen reset)
	// 2. maxNonResetRank = normale kaarten + wildcards (geen reset/joker)
	maxNaturalRank := Rank(0)
	maxNonResetRank := Rank(0)
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
		if hasReset {
			continue // reset-zetten niet meerekenen in maxNonResetRank
		}
		er := m.EffectiveRank(tableRank)
		if !hasWild && hasNormal {
			if er > maxNaturalRank {
				maxNaturalRank = er
			}
		}
		if er > maxNonResetRank {
			maxNonResetRank = er
		}
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
		if len(m.Cards) > round.Count && (hasWild || hasReset) && maxNonResetRank > tableRank {
			continue // gefilterd: goedkopere zet is efficiënter dan deze "/" combo
		}

		// Pure natural (geen wild, geen reset): altijd bewaren.
		if !hasWild && !hasReset {
			filtered = append(filtered, m)
			continue
		}
		// Vanaf hier: minstens één speciale kaart (wild of reset/joker).

		// Hiërarchie van "kostprijs": natural < wild-only < wild+reset of reset+normal.
		// Gebruik altijd de goedkoopste effectieve optie.

		// Als naturelle zetten de tafel al verslaan: filter ALLE speciale zetten.
		// Jokers zijn te waardevol om te verspillen als naturellen al werken —
		// ook "pure" joker-resets (00) zijn hier verspilling: je krijgt waarschijnlijk
		// toch de open ronde als de tegenstander niet kan beantwoorden, en anders
		// heb je je jokers bewaard voor een kritiek moment.
		if maxNaturalRank > tableRank {
			continue
		}
		// Naturelle zetten kunnen de tafel niet verslaan.
		// Als niet-reset zetten (wild+normaal) de tafel verslaan: filter reset-zetten.
		// "8822" domineert "8820" — zelfde rang, minder kostbaar.
		if maxNonResetRank > tableRank {
			if !hasReset {
				filtered = append(filtered, m) // wild-only zet: bewaren
			}
			// reset-zet gedomineerd door wild-only zet: filter
			continue
		}
		// Geen non-reset zet verslaat de tafel: bewaar alles dat de tafel verslaat.
		// Uitzondering: pure wildcards op hoge tafel (zoals Aas). EffectiveRank = tableRank
		// (niet strikt groter), maar wildcards ZIJN een geldige respons in AZEN — ze "slaan"
		// elke naturelle kaart. Bewaar ze als er geen naturelle opties zijn.
		isPureWildResponse := hasWild && !hasNormal && !hasReset &&
			m.EffectiveRank(tableRank) == tableRank && maxNaturalRank == 0
		if hasReset || m.EffectiveRank(tableRank) > tableRank || isPureWildResponse {
			filtered = append(filtered, m)
		}
	}

	// Post-filter: verwijder reset-zetten met te veel jokers als goedkopere reset-opties
	// beschikbaar zijn. "J0" domineert "00": zelfde reset-effect, één joker minder verspild.
	// Bijv. als J0 of Q0 beschikbaar zijn, moet 00 gefilterd worden.
	minResetCount := 999
	for _, m := range filtered {
		if m.IsPass {
			continue
		}
		rc := 0
		for _, c := range m.Cards {
			if c.IsReset() {
				rc++
			}
		}
		if rc > 0 && rc < minResetCount {
			minResetCount = rc
		}
	}
	if minResetCount < 999 {
		keep := filtered[:0]
		for _, m := range filtered {
			if m.IsPass {
				keep = append(keep, m)
				continue
			}
			rc := 0
			for _, c := range m.Cards {
				if c.IsReset() {
					rc++
				}
			}
			if rc == 0 || rc <= minResetCount {
				keep = append(keep, m)
			}
		}
		filtered = keep
	}

	return filtered
}

func (e *Engine) runWorker(gs *GameState, kt *KnowledgeTracker, iters int, minIters int, seed int64, rootFiltered []Move) workerResult {
	workerCfg := e.Config
	workerCfg.NumWorkers = 1
	worker := &Engine{Config: workerCfg, rng: rand.New(rand.NewSource(seed))}
	root := newRoot()
	myID := gs.CurrentTurn
	hasDeadline := worker.Config.MaxTime > 0
	deadline := time.Now().Add(worker.Config.MaxTime)
	actualIters := 0
	for iter := 0; iter < iters; iter++ {
		if hasDeadline && time.Now().After(deadline) && actualIters >= minIters {
			break
		}
		detGS := worker.determinize(gs, kt)
		if detGS == nil {
			continue
		}
		node, simGS := worker.selectExpand(root, detGS, myID, rootFiltered)
		result := worker.simulate(simGS, myID)
		worker.backprop(node, result, myID)
		actualIters++
	}
	res := workerResult{
		visits: map[string]int{},
		wins:   map[string]float64{},
		moves:  map[string]Move{},
		iters:  actualIters,
	}
	for _, ch := range root.children {
		k := mkey(ch.move)
		res.visits[k] += ch.visits
		res.wins[k] += ch.wins
		res.moves[k] = ch.move
	}
	return res
}

// preferCheaperMove returns true if a is a "cheaper" play than b: lower effective rank,
// then fewer cards. Used as tiebreaker when win rates are nearly identical.
func preferCheaperMove(a, b Move, tableRank Rank) bool {
	rankA := a.EffectiveRank(tableRank)
	rankB := b.EffectiveRank(tableRank)
	if rankA != rankB {
		return rankA < rankB
	}
	return len(a.Cards) < len(b.Cards)
}

// findForcedHighResponse detecteert de "schaakmat-verdediging":
// als tegenstander 1 kaart heeft, jij 1 kaart MOET spelen, en je geen joker hebt,
// is de hoogste kaart spelen altijd de enige correcte zet — geen MCTS nodig.
func findForcedHighResponse(gs *GameState) *Move {
	if gs.Round.IsOpen || gs.Round.Count != 1 || gs.NumPlayers != 2 {
		return nil
	}
	pid := gs.CurrentTurn
	if gs.Hands[1-pid].Count() != 1 {
		return nil
	}
	if gs.Hands[pid].CountRank(RankJoker) > 0 {
		return nil // joker aanwezig: MCTS beslist of joker beter is
	}
	moves := gs.GetLegalMoves()
	var best *Move
	bestER := Rank(0)
	bestIsWild := false
	for i := range moves {
		m := &moves[i]
		if m.IsPass || len(m.Cards) != 1 || m.Cards[0].IsReset() {
			continue
		}
		er := m.EffectiveRank(gs.Round.TableRank)
		isWild := m.Cards[0].IsWild()
		// Geldig: naturelle kaart die tafel verslaat, of pure wildcard (enige optie op Aas-tafel)
		if er <= gs.Round.TableRank && !isWild {
			continue
		}
		// Kies hoogste; bij gelijke rank geef voorkeur aan naturel boven wildcard
		if best == nil || er > bestER || (er == bestER && bestIsWild && !isWild) {
			best = m
			bestER = er
			bestIsWild = isWild
		}
	}
	return best
}

// canForceWinVsOneCard: recursieve helper — kan speler `pid` zijn hand leegspelen
// via een reeks open zetten waarbij de tegenstander (1 kaart) altijd moet passen?
// Geeft (true, diepte) terug: diepte = aantal eigen zetten tot winst.
func canForceWinVsOneCard(pid int, hand *Hand, depth int) (bool, int) {
	if depth > 15 {
		return false, 0
	}
	count := hand.Count()
	if count == 0 {
		return true, 0
	}
	bestDepth := 999
	found := false
	for _, m := range genOpenMoves(pid, hand) {
		n := len(m.Cards)
		if n == count {
			if 1 < bestDepth {
				bestDepth = 1
				found = true
			}
			continue
		}
		if n >= 2 || m.ContainsReset() {
			newHand := hand.Clone()
			if err := newHand.Remove(m.Cards); err != nil {
				continue
			}
			if ok, d := canForceWinVsOneCard(pid, newHand, depth+1); ok {
				if d+1 < bestDepth {
					bestDepth = d + 1
					found = true
				}
			}
		}
	}
	return found, bestDepth
}

// findForcedWinVsOneCard: open ronde, tegenstander heeft 1 kaart.
// Geeft de eerste zet van een gegarandeerde winreeks terug met exacte diepte.
func findForcedWinVsOneCard(gs *GameState) (*Move, int) {
	if !gs.Round.IsOpen || gs.NumPlayers != 2 {
		return nil, 0
	}
	pid := gs.CurrentTurn
	if gs.Hands[1-pid].Count() != 1 {
		return nil, 0
	}
	hand := gs.Hands[pid]
	if hand.Count() == 0 {
		return nil, 0
	}
	moves := genOpenMoves(pid, hand)
	var bestMove *Move
	bestDepth := 999
	for i := range moves {
		m := &moves[i]
		n := len(m.Cards)
		remaining := hand.Count() - n
		if remaining == 0 {
			if 1 < bestDepth {
				bestDepth = 1
				bestMove = m
			}
			continue
		}
		if n >= 2 || m.ContainsReset() {
			newHand := hand.Clone()
			if err := newHand.Remove(m.Cards); err != nil {
				continue
			}
			if ok, d := canForceWinVsOneCard(pid, newHand, 0); ok {
				totalDepth := d + 1
				if totalDepth < bestDepth {
					bestDepth = totalDepth
					bestMove = m
				}
			}
		}
	}
	if bestMove != nil {
		return bestMove, bestDepth
	}
	return nil, 0
}

// findForcedWinVsOneCardResponse: response ronde, tegenstander heeft 1 kaart,
// engine heeft een joker. Speelt de joker (reset) als antwoord om de ronde te
// openen, en controleert dan via canForceWinVsOneCard of engine gegarandeerd wint.
// Hiermee wordt de combinatie "0/7777 - 6" gevonden die MCTS mist in play mode.
func findForcedWinVsOneCardResponse(gs *GameState) (*Move, int) {
	if gs.Round.IsOpen || gs.NumPlayers != 2 {
		return nil, 0
	}
	pid := gs.CurrentTurn
	oppID := 1 - pid
	if gs.Hands[oppID].Count() != 1 {
		return nil, 0
	}
	if gs.Hands[pid].CountRank(RankJoker) == 0 {
		return nil, 0
	}
	moves := genResponseMoves(pid, gs.Hands[pid], gs.Round)
	var bestMove *Move
	bestDepth := 999
	for i := range moves {
		m := &moves[i]
		if !m.ContainsReset() {
			continue
		}
		sim := gs.Clone()
		sim.ApplyMove(*m)
		if sim.GameOver && sim.Winner == pid {
			mv := *m
			return &mv, 1
		}
		// Na joker-reset: open ronde met pid aan beurt
		if !sim.Round.IsOpen || sim.CurrentTurn != pid {
			continue
		}
		if ok, d := canForceWinVsOneCard(pid, sim.Hands[pid], 0); ok {
			totalDepth := d + 1 // +1 voor de joker-zet zelf
			if totalDepth < bestDepth {
				bestDepth = totalDepth
				mv := *m
				bestMove = &mv
			}
		}
	}
	if bestMove != nil {
		return bestMove, bestDepth
	}
	return nil, 0
}

// distinctRankCombos enumereert alle verschillende multisets van `k` kaarten die
// uit `pool` getrokken kunnen worden, ontdubbeld op rank (suit is irrelevant in
// dit spel). Gebruikt om alle mogelijke tegenstanderhanden in een klein eindspel
// af te lopen.
func distinctRankCombos(pool []Card, k int) [][]Card {
	counts := map[Rank]int{}
	for _, c := range pool {
		counts[c.Rank]++
	}
	ranks := make([]Rank, 0, len(counts))
	for r := range counts {
		ranks = append(ranks, r)
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })

	var res [][]Card
	cur := make([]Card, 0, k)
	var rec func(idx, remaining int)
	rec = func(idx, remaining int) {
		if remaining == 0 {
			combo := make([]Card, len(cur))
			copy(combo, cur)
			res = append(res, combo)
			return
		}
		if idx >= len(ranks) {
			return
		}
		r := ranks[idx]
		maxTake := counts[r]
		if maxTake > remaining {
			maxTake = remaining
		}
		for take := 0; take <= maxTake; take++ {
			for i := 0; i < take; i++ {
				cur = append(cur, Card{Rank: r})
			}
			rec(idx+1, remaining-take)
			cur = cur[:len(cur)-take]
		}
	}
	rec(0, k)
	return res
}

// findForcedWinEndgame2P zoekt in een 2-speler eindspel met verborgen kaarten naar
// een zet die tegen ELKE nog mogelijke tegenstanderhand een gedwongen winst
// oplevert. Dit vult het gat tussen findImmediateWin (alleen bij volledig bekende
// handen) en findForcedWinVsOneCard (alleen als de tegenstander exact 1 kaart
// heeft). Het vangt tempo-winsten die MCTS mist, bijv. {X, J, Aas} tegen 2
// kaarten: speel eerst laag (X of J) om een hoge kaart uit de tegenstander te
// lokken, herover de leiding met de Aas en sluit af met je laatste kaart.
//
// De check is strikt: een zet wordt alleen teruggegeven als forcedWinDepth voor
// ELKE consistente tegenstanderhand een gedwongen winst vindt. Handen met een
// nog-onzichtbare 2 (wildcard) of joker breken zo'n bewijs vanzelf — dan valt de
// functie terug op MCTS. Er wordt dus nooit een valse "gedwongen winst" gemeld.
func findForcedWinEndgame2P(gs *GameState, kt *KnowledgeTracker) (*Move, int) {
	if gs == nil || kt == nil || gs.NumPlayers != 2 || gs.GameOver {
		return nil, 0
	}
	pid := gs.CurrentTurn
	oppID := 1 - pid
	myCount := gs.Hands[pid].Count()
	oppCount := gs.Hands[oppID].Count()
	// Grenzen zodat volledige minimax + handenumeratie goedkoop blijft.
	if myCount < 2 || myCount > 6 || oppCount < 1 || oppCount > 3 {
		return nil, 0
	}

	pool := kt.PossibleOpponentCards()
	if len(pool) < oppCount || len(pool) > 28 {
		return nil, 0
	}
	cands := distinctRankCombos(pool, oppCount)
	if len(cands) == 0 || len(cands) > 800 {
		return nil, 0
	}

	maxDepth := (myCount + oppCount) * 4
	const nodeBudget = 60000

	moves := gs.GetLegalMoves()
	bestDepth := -1
	var bestMove *Move
	for i := range moves {
		m := moves[i]
		if m.IsPass {
			continue
		}
		worst := 0
		forcedAll := true
		for _, cand := range cands {
			sim := gs.Clone()
			sim.Hands[oppID] = NewHand(cand)
			sim.ApplyMove(m)
			if sim.GameOver {
				if notLastPlace(sim, pid) {
					if worst < 1 {
						worst = 1
					}
					continue
				}
				forcedAll = false
				break
			}
			nodes := 0
			d := forcedWinDepth(sim, pid, maxDepth, &nodes, nodeBudget)
			if d < 0 {
				forcedAll = false
				break
			}
			if d+1 > worst {
				worst = d + 1
			}
		}
		if !forcedAll {
			continue
		}
		if bestMove == nil || worst < bestDepth ||
			(worst == bestDepth && preferCheaperMove(m, *bestMove, gs.Round.TableRank)) {
			bestDepth = worst
			mv := m
			bestMove = &mv
		}
	}
	if bestMove != nil {
		return bestMove, bestDepth
	}
	return nil, 0
}

// findForcedLoss detecteert of de huidige speler in een gedwongen verlies-positie zit
// en vindt diens beste vertragingszet (de zet die het verlies het langste uitstelt).
// Enkel van toepassing bij bekende handen (omniscient mode); tot 14 totale kaarten
// (12 zonder bekende handen), of onbeperkt zodra iemand geëindigd is.
// Retourneert (bestDelayMove, maxDelay) waarbij maxDelay = max. tegenstander-beurten
// tot verlies na de beste vertragingszet. Geeft (nil, 0) terug als niet van toepassing.
func findForcedLoss(gs *GameState, knownHands bool) (*Move, int) {
	if !knownHands || gs.GameOver {
		return nil, 0
	}
	pid := gs.CurrentTurn
	totalCards := 0
	for _, h := range gs.Hands {
		totalCards += h.Count()
	}

	// 2 spelers + bekende handen: exacte oplossing met transpositietabel.
	// Werkt tot ~36 kaarten (de hele resterende partij), dus veel eerder dan de
	// gewone minimax. De tabel wordt over de analyse hergebruikt.
	if gs.NumPlayers == 2 && totalCards <= 40 {
		tt := analyzeSolveTT
		if tt == nil {
			tt = make(map[uint64]sfEntry, 1<<20)
		}
		if mv, dist, solved := solveForcedLoss2P(gs, tt, solveNodeBudget); solved {
			return mv, dist // mv == nil ⇒ opgelost, geen gedwongen verlies
		}
	}

	someoneFinished := knownHands && gs.Winner != -1
	cardLimit := 12
	if knownHands {
		cardLimit = 14
	}
	if !someoneFinished && totalCards > cardLimit {
		return nil, 0
	}
	maxDepth := totalCards * 3
	if totalCards <= 8 {
		maxDepth = totalCards * 4
	}
	nodeLimit := 400000
	if knownHands {
		nodeLimit = 2000000
	}
	moves := gs.GetLegalMoves()
	bestDelay := -1
	var bestMove *Move
	for _, m := range moves {
		sim := gs.Clone()
		sim.ApplyMove(m)
		if sim.GameOver {
			if notLastPlace(sim, pid) {
				return nil, 0 // winning move: not a forced loss
			}
			continue
		}
		oppWinDepth := -1
		nodes := 0
		for oppID := 0; oppID < gs.NumPlayers; oppID++ {
			if oppID == pid || sim.Finished[oppID] {
				continue
			}
			d := forcedWinDepth(sim, oppID, maxDepth, &nodes, nodeLimit)
			if d >= 0 && (oppWinDepth < 0 || d < oppWinDepth) {
				oppWinDepth = d
			}
		}
		if oppWinDepth < 0 {
			return nil, 0 // opponent has no forced win after this move → no forced loss
		}
		if oppWinDepth > bestDelay {
			bestDelay = oppWinDepth
			mv := m
			bestMove = &mv
		}
	}
	if bestMove == nil {
		return nil, 0
	}
	return bestMove, bestDelay
}

// oppWinDepthAfterMove berekent het min. aantal tegenstander-beurten tot verlies na zet m,
// in de context van een geforceerd verlies. Geeft -1 als niet detecteerbaar.
func oppWinDepthAfterMove(gs *GameState, m Move) int {
	pid := gs.CurrentTurn
	totalCards := 0
	for _, h := range gs.Hands {
		totalCards += h.Count()
	}

	// 2 spelers + analyse-tabel: gebruik dezelfde exacte oplosser als
	// findForcedLoss, zodat de "verlies in X i.p.v. Y" vergelijking klopt.
	if analyzeSolveTT != nil && gs.NumPlayers == 2 && totalCards <= 40 {
		return distAfterMove2P(gs, m, analyzeSolveTT, solveNodeBudget)
	}

	maxDepth := totalCards * 3
	if totalCards <= 8 {
		maxDepth = totalCards * 4
	}
	sim := gs.Clone()
	sim.ApplyMove(m)
	if sim.GameOver {
		if notLastPlace(sim, pid) {
			return -1 // pid is niet laatste = pid wint = tegenstander wint niet
		}
		return 0 // pid is laatste = tegenstander wint in 0 beurten
	}
	nodes := 0
	minOpp := -1
	for oppID := 0; oppID < gs.NumPlayers; oppID++ {
		if oppID == pid || sim.Finished[oppID] {
			continue
		}
		d := forcedWinDepth(sim, oppID, maxDepth, &nodes, 2000000)
		if d >= 0 && (minOpp < 0 || d < minOpp) {
			minOpp = d
		}
	}
	return minOpp
}

// ── Exacte 2-speler eindspel-oplosser met transpositietabel ──────────────────
// In analysemodus zijn alle handen bekend; dan is de rest van de partij een
// eindige, perfecte-informatie 2-speler nulsomgame. Een negamax met
// transpositietabel lost die typisch in enkele miljoenen knopen op — ook vanaf
// het begin van een deel van 36 kaarten — waar de gewone minimax vastloopt.

type sfEntry struct {
	win  bool // speler-aan-zet eindigt NIET laatste
	dist int  // aantal zetten (incl. passen) tot einde bij optimaal spel
}

const solveNodeBudget = 15_000_000

// analyzeSolveTT wordt door analyzeMode gezet zodat de tabel over de hele partij
// hergebruikt wordt (elke latere stelling is een substelling → cache-hit).
var analyzeSolveTT map[uint64]sfEntry

func stateKey(gs *GameState) uint64 {
	var h uint64 = 1469598103934665603
	mix := func(x uint64) { h ^= x; h *= 1099511628211 }
	for pi, hd := range gs.Hands {
		var cnt [17]int
		for _, c := range hd.Cards {
			if int(c.Rank) < 17 {
				cnt[c.Rank]++
			}
		}
		for r := 0; r < 17; r++ {
			mix(uint64(pi)<<40 | uint64(r)<<8 | uint64(cnt[r]))
		}
	}
	r := gs.Round
	var b uint64
	if r.IsOpen {
		b = 1
	}
	mix(uint64(gs.CurrentTurn)<<48 | uint64(r.Count)<<32 | uint64(r.TableRank)<<16 |
		uint64(r.ConsecPasses)<<8 | uint64(r.LastPlayerID)<<4 | b)
	return h
}

// ttSolve2P geeft (win, dist, ok) voor de speler die in gs aan zet is.
// win = die speler eindigt niet-laatste bij optimaal spel van beide kanten.
// dist = zetten tot einde: de winnende kant minimaliseert, de verliezende
// maximaliseert (langst mogelijke weerstand). ok = false als het budget op is.
func ttSolve2P(gs *GameState, tt map[uint64]sfEntry, nodes *int, maxNodes int) (bool, int, bool) {
	*nodes++
	if *nodes > maxNodes {
		return false, 0, false
	}
	pid := gs.CurrentTurn
	key := stateKey(gs)
	if e, ok := tt[key]; ok {
		return e.win, e.dist, true
	}
	haveBest := false
	bestWin := false
	bestDist := 0
	for _, m := range gs.GetLegalMoves() {
		sim := gs.Clone()
		sim.ApplyMove(m)
		var cw bool
		var cd int
		if sim.GameOver {
			cw, cd = notLastPlace(sim, pid), 1
		} else {
			w, d, ok := ttSolve2P(sim, tt, nodes, maxNodes)
			if !ok {
				return false, 0, false
			}
			if sim.CurrentTurn == pid {
				cw = w
			} else {
				cw = !w // 2-speler nulsom
			}
			cd = d + 1
		}
		better := !haveBest
		if haveBest {
			switch {
			case cw != bestWin:
				better = cw // winst verslaat verlies
			case cw:
				better = cd < bestDist // winnen: sneller
			default:
				better = cd > bestDist // verliezen: langer overleven
			}
		}
		if better {
			haveBest, bestWin, bestDist = true, cw, cd
		}
	}
	tt[key] = sfEntry{bestWin, bestDist}
	return bestWin, bestDist, true
}

// solveForcedLoss2P lost het 2-speler eindspel exact op. Retourneert:
//   - (nil, 0, true)  : opgelost, speler-aan-zet staat NIET verloren
//   - (zet, N, true)  : opgelost, gedwongen verlies; `zet` rekt het verlies het
//                       langst, N = totaal aantal zetten tot verlies op dat pad
//   - (nil, 0, false) : niet opgelost binnen het budget
func solveForcedLoss2P(gs *GameState, tt map[uint64]sfEntry, maxNodes int) (*Move, int, bool) {
	if gs.NumPlayers != 2 || gs.GameOver {
		return nil, 0, false
	}
	nodes := 0
	win, _, ok := ttSolve2P(gs, tt, &nodes, maxNodes)
	if !ok {
		return nil, 0, false
	}
	if win {
		return nil, 0, true
	}
	var best *Move
	bestDist := -1
	for _, m := range gs.GetLegalMoves() {
		sim := gs.Clone()
		sim.ApplyMove(m)
		d := 1
		if !sim.GameOver {
			_, cd, ok2 := ttSolve2P(sim, tt, &nodes, maxNodes)
			if !ok2 {
				continue
			}
			d = cd + 1
		}
		if d > bestDist {
			bestDist, best = d, &Move{PlayerID: m.PlayerID, Cards: m.Cards, IsPass: m.IsPass}
		}
	}
	if best == nil {
		return nil, 0, true
	}
	return best, bestDist, true
}

// distAfterMove2P geeft het aantal zetten tot verlies na zet m (kleiner = sneller
// verloren). Geeft -1 als m juist ontsnapt (geen gedwongen verlies meer), of als
// het budget op is.
func distAfterMove2P(gs *GameState, m Move, tt map[uint64]sfEntry, maxNodes int) int {
	pid := gs.CurrentTurn
	sim := gs.Clone()
	sim.ApplyMove(m)
	if sim.GameOver {
		if notLastPlace(sim, pid) {
			return -1
		}
		return 0
	}
	nodes := 0
	w, d, ok := ttSolve2P(sim, tt, &nodes, maxNodes)
	if !ok {
		return -1
	}
	winForPid := w
	if sim.CurrentTurn != pid {
		winForPid = !w
	}
	if winForPid {
		return -1 // deze zet ontsnapt
	}
	return d + 1
}

func (e *Engine) BestMove(gs *GameState, kt *KnowledgeTracker) (Move, MoveEval) {
	// Als alle handen exact bekend zijn (OmniscientMode of gok volledig), gebruik een
	// deterministische toestand voor schaakmat-detectie — MCTS gebruikt gs + determinize zoals normaal.
	knownHands := e.Config.OmniscientMode
	searchGS := gs
	if !knownHands && kt != nil && kt.AllOpponentHandsKnown() {
		if det := e.determinize(gs, kt); det != nil {
			searchGS = det
			knownHands = true
		}
	}

	if win, depth := findImmediateWin(searchGS, knownHands); win != nil {
		return *win, MoveEval{Score: 1.0, Visits: 1, ForcedWinDepth: depth}
	}
	if forced := findForcedHighResponse(searchGS); forced != nil {
		return *forced, MoveEval{Score: 1.0, Visits: 1}
	}
	if win, depth := findForcedWinVsOneCard(searchGS); win != nil {
		return *win, MoveEval{Score: 1.0, Visits: 1, ForcedWinDepth: depth}
	}
	if win, depth := findForcedWinVsOneCardResponse(searchGS); win != nil {
		return *win, MoveEval{Score: 1.0, Visits: 1, ForcedWinDepth: depth}
	}
	// Klein 2-speler eindspel met verborgen kaarten: exacte tempo-oplossing tegen
	// alle nog mogelijke tegenstanderhanden. findImmediateWin draait hier niet
	// (handen niet volledig bekend) en findForcedWinVsOneCard evenmin (tegenstander
	// heeft >1 kaart). Overslaan als de handen wél bekend zijn — dan dekt
	// findImmediateWin het al af.
	if !knownHands {
		if win, depth := findForcedWinEndgame2P(gs, kt); win != nil {
			return *win, MoveEval{Score: 1.0, Visits: 1, ForcedWinDepth: depth}
		}
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
	minItersPerWorker := e.Config.MinIterations / numWorkers
	if minItersPerWorker < 1 && e.Config.MinIterations > 0 {
		minItersPerWorker = 1
	}
	seeds := make([]int64, numWorkers)
	for i := range seeds {
		seeds[i] = e.rng.Int63()
	}
	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	mctsStart := time.Now()
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			iters := itersPerWorker
			if idx == numWorkers-1 {
				iters = e.Config.Iterations - itersPerWorker*(numWorkers-1)
			}
			results[idx] = e.runWorker(gs, kt, iters, minItersPerWorker, seeds[idx], rootFiltered)
		}(w)
	}
	wg.Wait()
	mctsElapsedMs := time.Since(mctsStart).Seconds() * 1000

	totalVisits := map[string]int{}
	totalWins := map[string]float64{}
	moveMap := map[string]Move{}
	totalMctsIters := 0
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
		totalMctsIters += r.iters
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
	bestMove := moveMap[bestKey]
	wr := 0.0
	if bestVisits > 0 {
		wr = totalWins[bestKey] / float64(bestVisits)
	}
	eval := MoveEval{Score: wr, Visits: bestVisits, Details: details, TotalIters: totalMctsIters, ElapsedMs: mctsElapsedMs}
	bestMove, eval = overridePass(bestMove, eval, gs, rootFiltered)
	return bestMove, eval
}

func (e *Engine) bestMoveSingle(gs *GameState, kt *KnowledgeTracker, rootFiltered []Move) (Move, MoveEval) {
	root := newRoot()
	myID := gs.CurrentTurn
	hasDeadline := e.Config.MaxTime > 0
	deadline := time.Now().Add(e.Config.MaxTime)
	singleStart := time.Now()
	actualIters := 0
	for iter := 0; iter < e.Config.Iterations; iter++ {
		if hasDeadline && time.Now().After(deadline) && actualIters >= e.Config.MinIterations {
			break
		}
		detGS := e.determinize(gs, kt)
		if detGS == nil {
			continue
		}
		node, simGS := e.selectExpand(root, detGS, myID, rootFiltered)
		result := e.simulate(simGS, myID)
		e.backprop(node, result, myID)
		actualIters++
	}
	singleElapsedMs := time.Since(singleStart).Seconds() * 1000
	bestMove, eval := e.pickBest(root, myID, gs.Round.TableRank)
	eval.TotalIters = actualIters
	eval.ElapsedMs = singleElapsedMs
	bestMove, eval = overridePass(bestMove, eval, gs, rootFiltered)
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
			// Move ordering: kies de best-beoordeelde onverkende zet
			// i.p.v. willekeurig. QuickEvaluateMove geeft heuristische score.
			m := unexplored[0]
			if len(unexplored) > 1 {
				bestScore := -999.0
				for _, um := range unexplored {
					var sc float64
					if um.IsPass {
						sc = -1.0
					} else {
						sc = QuickEvaluateMove(simGS, um).Score
					}
					// Kleine random tiebreak zodat gelijke zetten niet altijd dezelfde volgorde hebben
					sc += e.rng.Float64() * 0.5
					if sc > bestScore {
						bestScore = sc
						m = um
					}
				}
			}
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
	// Adaptieve rollout-limiet: bij weinig kaarten altijd tot GameOver uitspelen.
	// Bij veel kaarten: max 200 zetten (meer dan genoeg, voorkomt oneindige loops).
	totalCards := 0
	for _, h := range sim.Hands {
		totalCards += h.Count()
	}
	maxSteps := 200
	if totalCards <= 12 {
		maxSteps = 800 // bij weinig kaarten ALTIJD tot GameOver, geen evalPos-afbreking
	}
	for i := 0; i < maxSteps && !sim.GameOver; i++ {
		moves := filterDominatedMoves(sim.GetLegalMoves(), sim.Round)
		if len(moves) == 0 {
			break
		}
		// Greedy/random rollout: QuickEvaluateMove kiest de best-beoordeelde zet
		// (overshoot-penalty, wild-straf, paar-breek, win-threats). De random
		// component (smartRandom) zorgt voor diversiteit.
		// OmniscientMode: 70/30 greedy/random (sterkere heuristiek bij bekende handen)
		// Play mode: 40/60 greedy/random (meer diversiteit bij onbekende handen)
		var m Move
		greedyThreshold := 0.4
		if e.Config.OmniscientMode {
			greedyThreshold = 0.7
		}
		if e.rng.Float64() < greedyThreshold {
			m = moves[0]
			bestQScore := -999.0
			for _, alt := range moves {
				var sc float64
				if alt.IsPass {
					sc = -1.0
				} else {
					sc = QuickEvaluateMove(sim, alt).Score
				}
				if sc > bestQScore {
					bestQScore = sc
					m = alt
				}
			}
		} else {
			m = e.smartRandom(moves, sim)
		}
		sim.ApplyMove(m)
	}
	var score float64
	if sim.GameOver {
		score = positionScore(sim, myID)
	} else {
		score = e.evalPos(sim, myID)
	}

	// Druk-modus: blend eigen score met "hoe slecht doet het doel het"
	if e.Config.PressureTarget >= 0 && e.Config.PressureTarget != myID {
		target := e.Config.PressureTarget
		var targetScore float64
		if sim.Finished[target] {
			targetScore = positionScore(sim, target)
		} else {
			targetCount := float64(sim.Hands[target].Count())
			minOpp := 999.0
			for i, h := range sim.Hands {
				if i != target && !sim.Finished[i] {
					if float64(h.Count()) < minOpp {
						minOpp = float64(h.Count())
					}
				}
			}
			if minOpp == 999 {
				minOpp = 0
			}
			targetScore = 0.5 + (minOpp-targetCount)*0.085
			if targetScore < 0 {
				targetScore = 0
			}
			if targetScore > 1 {
				targetScore = 1
			}
		}
		// 40% eigen positie + 60% nadeel voor het doelwit
		score = 0.4*score + 0.6*(1.0-targetScore)
	}

	return score
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
	passChance := 0.085 + specialRatio*0.225

	// Early-game pass bonus: alleen bij 3+ spelers.
	// In 2-speler is passen altijd gevaarlijk (tegenstander krijgt open ronde), dus nooit verhogen.
	// Gebruik het MINIMUM tegenstander-kaartaantal: als een tegenstander weinig kaarten heeft
	// (bijv. 5), mag de bonus NIET gegeven worden ook al heeft een andere tegenstander er 15.
	if handCount >= 8 && gs.activePlayerCount() > 2 {
		minOppForBonus := 999
		for i, h := range gs.Hands {
			if i != gs.CurrentTurn && !gs.Finished[i] && h.Count() < minOppForBonus {
				minOppForBonus = h.Count()
			}
		}
		if minOppForBonus != 999 && handCount <= minOppForBonus {
			passChance += 0.32
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

	// 2-speler: passen is bijna altijd tempo-verlies — tegenstander krijgt vrije open ronde
	// en kan ongehinderd doorspelen. Eénmalig passen = nooit meer aan bod komen.
	{
		activePlayers2 := 0
		for i, h := range gs.Hands {
			if !gs.Finished[i] && h.Count() > 0 {
				activePlayers2++
			}
		}
		if activePlayers2 <= 2 {
			passChance *= 0.02
		}
	}

	// Lage tafel (≤7): passen is bijna nooit zinvol — vrijwel elke kaart verslaat het
	if !gs.Round.IsOpen && gs.Round.TableRank <= RankSeven {
		passChance *= 0.04
	}

	if e.rng.Float64() < passChance {
		return pass
	}

	// === SPEEL-KEUZE ===
	acePlayFactor := 0.52
	wildPlayFactor := 0.33
	synergyPenalty := 0.40

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
		// SKIP als er ook een reset in de combo zit: joker+wilds = bewuste kill-shot (x220).
		if wilds > 0 && resets == 0 {
			if handCount >= 12 {           // nog veel kaarten
				if effective <= RankFive {              // extreem lage beat (zoals 3)
					w *= 0.09                           // bijna onmogelijk maken
				} else if effective <= RankEight {
					w *= 0.28
				} else if effective <= RankTen {
					w *= 0.40
				} else if effective <= RankQueen {
					w *= 0.40
				}
				// King+wild: geen extra straf (K is terecht moeilijk anders te spelen)
			} else {
				if effective <= RankSix {
					w *= 0.25
				} else if effective <= RankNine {
					w *= 0.45
				}
			}
		}
		// =====================================================

		// Extra straf: wildcards aan lage combo in open ronde (bijv. 33322 ipv 333).
		// Wildcards zijn te waardevol om te verspillen aan een lage tafelsetting.
		if gs.Round.IsOpen && wilds > 0 && resets == 0 && effective <= RankSeven {
			w *= 0.08 // bijna nooit: wildcard verspillen aan lage open combo
		}

		// Bonus voor dumpen van lage normale kaarten (4 4 krijgt voorkeur).
		// ONDERDRUK in eindspel (≤2 kaarten over): dan geldt STRATEGIE, niet dumporde.
		// Bijv. {3,K,K}: KK spelen (cardsAfter=1) is beter dan 3 spelen (cardsAfter=2).
		cardsAfterM := handCount - len(m.Cards)
		if wilds == 0 && resets == 0 && len(m.Cards) >= 1 && cardsAfterM > 2 {
			lowest := m.Cards[0].Rank
			if lowest <= RankFive {
				w *= 1.60
			} else if lowest <= RankEight {
				w *= 1.30
			}
		}

		// Drain strategie: geïsoleerde single in open ronde bij 2 spelers.
		// Forceer tegenstander zijn combo te breken → sterker dan eigen quad spelen.
		if gs.Round.IsOpen && gs.NumPlayers == 2 && len(m.Cards) == 1 &&
			wilds == 0 && resets == 0 && handCount >= 8 {
			rankCount := curHand.CountRank(effective)
			if rankCount == 1 {
				comboCount := 0
				for r := RankThree; r <= RankAce; r++ {
					if r == effective {
						continue
					}
					if curHand.CountRank(r) >= 2 {
						comboCount++
					}
				}
				remWilds := curHand.CountRank(RankTwo)
				remResets := curHand.CountRank(RankJoker)
				if comboCount >= 2 || (comboCount >= 1 && (remWilds+remResets) >= 1) {
					w *= 3.5
				}
			}
		}

		// Singleton dump vs cluster-breek: bij een single-antwoord op een single tafel,
		// sterk voorkeur voor LAGE kaarten waarvan je er maar 1 hebt (geïsoleerde kaarten).
		// Straf als je een paar of triple moet breken voor een enkele kaart.
		// MAAR: hoge singletons (Aas, Heer) zijn waardevol en moeten bewaard worden!
		if wilds == 0 && resets == 0 && !gs.Round.IsOpen &&
			len(m.Cards) == 1 && gs.Round.Count == 1 {
			rankCount := curHand.CountRank(effective)
			if rankCount == 1 {
				if effective >= RankAce {
					w *= 0.30 // Aas-singleton: BEWAREN, niet dumpen — uniek sterk
				} else if effective >= RankKing {
					w *= 0.60 // Heer-singleton: mild bewaren
				} else {
					w *= 2.2 // lage singleton: wil je kwijt
				}
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
		// Synergy joker + wildcards:
		// Open ronde: kleine netto-straf (reset+wilds is kostbaar, maar soms nodig).
		// Antwoord-ronde: GEEN straf — joker+wildcard als response is de kill-shot (x220).
		if resets > 0 && wilds > 0 {
			if gs.Round.IsOpen {
				w *= 2.1 * synergyPenalty // net ~0.84
			}
			// antwoord-ronde: geen synergyPenalty
		}

		for _, c := range m.Cards {
			// !IsSpecial() includes Ace now (natural card), but Ace has high rank
			// The formula rewards LOW ranks; Ace (14) gets slight penalty = correct
			if !c.IsSpecial() {
				w *= 1.0 + 0.11*(13.0-float64(c.Rank))
			}
		}

		// Response-overshoot: in response-rondes de LAAGSTE winnende zet prefereren.
		// KK op een X-tafel is verspilling als JJ of QQ ook wint.
		// Exponentiële afname: elke rank boven het minimum kost ~15% gewicht.
		// Uitzondering: ≤2 kaarten in hand of tegenstander heeft 1 kaart = eindspel.
		opponentHasOne := false
		if !gs.Round.IsOpen && gs.NumPlayers == 2 {
			opponentID := 1 - gs.CurrentTurn
			opponentHasOne = gs.Hands[opponentID].Count() == 1
		}
		if !gs.Round.IsOpen && wilds == 0 && resets == 0 && effective > gs.Round.TableRank && handCount >= 3 && !opponentHasOne {
			overshoot := float64(effective-gs.Round.TableRank) - 1.0
			if overshoot > 0 {
				w *= math.Pow(0.85, overshoot)
			}
		}

		// Aas-bescherming: Aas niet verspillen op lage tafel als goedkopere opties bestaan.
		// De Aas is het ultieme wapen tegen Heer/Aas van de tegenstander.
		// Uitzondering: eindspel met ≤2 kaarten of tegenstander heeft 1 kaart.
		if effective == RankAce && !gs.Round.IsOpen && gs.Round.TableRank <= RankJack {
			if handCount >= 3 && !opponentHasOne {
				w *= 0.15
			}
		}

		// Tegenstander heeft 1 kaart: speel HOOG om passen of speciale kaart te dwingen.
		// Zijn resterende kaart is sowieso zijn sterkste — laagste response = gratis verlies.
		// Exponentieel hogere boost voor hogere ranks om low-rank bias te compenseren.
		if !gs.Round.IsOpen && opponentHasOne && wilds == 0 && resets == 0 && effective > gs.Round.TableRank {
			overshootHigh := float64(effective-gs.Round.TableRank) - 1.0
			w *= math.Pow(1.45, overshootHigh+1)
		}

		// Near-win bonus: als je na deze zet ≤3 kaarten overhoudt, verhoog het gewicht
		// sterk. Dit overstijgt rank-voorkeur en dump-bonussen in de eindspelfase.
		// Bijv. QQQ (cardsAfter=1) wint terecht van 5 (cardsAfter=3) bij {5,Q,Q,Q}.
		switch cardsAfterM {
		case 1:
			w *= 6.0
		case 2:
			w *= 3.0
		case 3:
			w *= 1.5
		}

		// ── FIX 2: "Lage first, hoge last" bij 2 kaarten in open ronde ──────
		// Als je 2 kaarten hebt, beide singles, open ronde:
		// speel de LAGE eerst zodat de HOGE als sluitsteen overblijft.
		// Voorbeeld: hand={5, K}, open ronde → speel 5, bewaar K als finale.
		// Omgekeerd (K spelen, 5 over) = rampzalig: 5 kan niet antwoorden op doubles.
		//
		// Uitzondering: als de twee kaarten een paar zijn, maakt volgorde niet uit —
		// maar paren worden toch al als één zet gespeeld (cardsAfterM=0 → winst).
		if handCount == 2 && gs.Round.IsOpen && len(m.Cards) == 1 &&
			wilds == 0 && resets == 0 {
			// Bepaal welke kaart overblijft na deze zet
			var remainingRank Rank
			for _, c := range curHand.Cards {
				if c.Rank != effective {
					remainingRank = c.Rank
					break
				}
			}
			if remainingRank > 0 && remainingRank != effective {
				// We spelen `effective`, bewaren `remainingRank`
				// Bonus als de overblijvende kaart HOGER is (goede sluitsteen)
				// Penalty als de overblijvende kaart LAGER is (slechte sluitsteen)
				if remainingRank > effective {
					// Lage kaart spelen, hoge bewaren → correct
					w *= 4.0
				} else {
					// Hoge kaart spelen, lage bewaren → riskant
					// Hoe lager de overblijvende kaart, hoe groter de straf
					switch {
					case remainingRank <= RankFive:
						w *= 0.05 // catastrofaal: 3/4/5 als sluitsteen
					case remainingRank <= RankEight:
						w *= 0.15 // slecht
					case remainingRank <= RankTen:
						w *= 0.35 // riskant
					case remainingRank <= RankQueen:
						w *= 0.60 // matig
					}
				}
			}
		}
		// ────────────────────────────────────────────────────────────────────

		// ── Tempo bij 3 losse singles in open ronde (2-speler eindspel) ──────
		// Hand = 3 verschillende singles, open ronde, wij leiden. Speel de
		// LAAGSTE eerst: die lokt een hoge kaart uit de tegenstander. Bewaar de
		// HOOGSTE (boss-kaart) om de leiding te heroveren en de middelste als
		// sluitkaart. De boss-kaart als eerste inleggen terwijl de tegenstander
		// gewoon mag passen, verspilt je enige tempo-instrument.
		// Voorbeeld: {X, J, Aas} vs {Q, K} → speel X (of J), niet de Aas.
		if handCount == 3 && gs.Round.IsOpen && len(m.Cards) == 1 &&
			wilds == 0 && resets == 0 && activePlayerCount(gs) <= 2 {
			distinct := map[Rank]bool{}
			for _, c := range curHand.Cards {
				if !c.IsSpecial() {
					distinct[c.Rank] = true
				}
			}
			if len(distinct) == 3 {
				var lo Rank = RankJoker
				var hi Rank
				for r := range distinct {
					if r < lo {
						lo = r
					}
					if r > hi {
						hi = r
					}
				}
				switch {
				case effective == lo:
					w *= 2.2 // laagste eerst → correcte tempo-opbouw
				case effective == hi:
					w *= 0.30 // boss-kaart eerst → tempo verspild
				default:
					w *= 1.1 // middelste: acceptabel
				}
			}
		}
		// ────────────────────────────────────────────────────────────────────

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
	minOpp := 999
	for i, h := range gs.Hands {
		if i != myID && !gs.Finished[i] && h.Count() < minOpp {
			minOpp = h.Count()
		}
	}
	if minOpp == 999 {
		minOpp = 0
	}
	score := 0.5 + float64(minOpp-myCount)*0.085

	// Urgentiepenalty: bij 3+ kaarten achter een extra niet-lineaire straf.
	gap := myCount - minOpp
	if gap >= 3 {
		score -= math.Pow(float64(gap), 1.2) * 0.09
	}

	// 3+ spelers: extra dreiging-penalty als de leidende tegenstander ≤5 kaarten heeft.
	// MCTS rollouts onderschatten systematisch hoe snel een leider met weinig kaarten wint.
	// Door de positiescore extra te drukken worden zetten die de leider stoppen beloond.
	if activePlayerCount(gs) > 2 && minOpp <= 5 && minOpp < myCount {
		// Hoe minder kaarten de leider heeft, hoe ernstiger de dreiging.
		// Bij 5 kaarten: -0.04, bij 4: -0.08, bij 3: -0.12, bij 2: -0.16, bij 1: -0.20
		score -= float64(6-minOpp) * 0.04
	}

	hand := gs.Hands[myID]
	wilds := hand.CountRank(RankTwo)    // alleen 2 is wildcard
	resets := hand.CountRank(RankJoker) // joker is reset-kaart
	resetBonus := 0.33
	if activePlayerCount(gs) <= 2 && minOpp >= 6 {
		// In 2-speler mid-game is de joker uniek waardevol: enige manier om triple aas
		// te beantwoorden via "022". Zonder joker ben je verplicht te passen op aces.
		resetBonus = 0.55
	}
	score += float64(resets) * resetBonus
	score += float64(wilds) * 0.38
	if resets > 0 && wilds > 0 {
		score += float64(imin(resets, wilds)) * 0.20
	}
	kings := hand.CountRank(RankKing)
	if kings > 0 && wilds == 0 && resets == 0 {
		score -= float64(kings) * 0.05
	}
	queens := hand.CountRank(RankQueen)
	if queens > 0 && wilds == 0 && resets == 0 {
		score -= float64(queens) * 0.032
	}
	// Geïsoleerde lage kaarten (3-7): moeilijk te dumpen als single.
	// In 2-speler zijn ze duurder: tegenstander krijgt gratis open ronde als je ze speelt
	// en ze breken je tempo.
	isolatedPenalty := 0.042
	if activePlayerCount(gs) <= 2 {
		isolatedPenalty = 0.080
	}
	for r := RankThree; r <= RankSeven; r++ {
		if hand.CountRank(r) == 1 && wilds == 0 {
			score -= isolatedPenalty
		}
	}
	// Geïsoleerde midden-kaarten (8-X): ook lastig, maar iets minder erg
	for r := RankEight; r <= RankTen; r++ {
		if hand.CountRank(r) == 1 && wilds == 0 {
			score -= isolatedPenalty * 0.5
		}
	}
	for r := RankThree; r <= RankAce; r++ {
		cnt := hand.CountRank(r)
		if cnt >= 2 {
			score += float64(cnt-1) * 0.038
			// Hoge paren zijn meer waard: een paar Aces is veel sterker dan paar 3-en.
			// Bonus gebaseerd op rank (3=0.0, Ace=1.0) × 0.04 per extra kaart.
			rankFactor := float64(r-RankThree) / float64(RankAce-RankThree)
			score += float64(cnt-1) * rankFactor * 0.04
		}
	}
	// Sluitende combinatie: joker + pair/triple in hand ≤ 5 kaarten = bijna zekere win.
	// De joker reset de ronde, daarna dump je het pair in 1 zet.
	if resets > 0 && myCount <= 5 {
		for r := RankThree; r <= RankAce; r++ {
			cnt := hand.CountRank(r)
			if cnt >= 2 && cnt+resets >= myCount {
				// Joker + pair/triple = alle kaarten in 2 zetten
				score += 0.15
				break
			}
		}
	}

	// ── FIX 1: Eindkaart-kwaliteit ──────────────────────────────────────────
	// Met 1 kaart over is de kwaliteit van die kaart allesbepalend:
	//   - Lage single (3-7):  bijna zeker verlies — tegenstander heeft altijd
	//     iets hoger, en in open ronde ben je verplicht die lage kaart te openen.
	//   - Midden single (8-Q): neutraal tot licht negatief.
	//   - Hoge single (K, 1): sterk — verslaat bijna alles als single.
	//   - Wildcard (2) of Joker (0): uitstekend — altijd speelbaar.
	if myCount == 1 {
		c := hand.Cards[0]
		switch {
		case c.IsReset() || c.IsWild():
			score += 0.18 // joker of 2: altijd uitweg
		case c.Rank == RankAce:
			score += 0.12 // aas single: sterk slotkaart
		case c.Rank == RankKing:
			score += 0.06 // heer: goed maar niet onfeilbaar
		case c.Rank >= RankTen:
			score -= 0.04 // X/J/Q als single: riskant
		case c.Rank >= RankEight:
			score -= 0.12 // 8/9: slecht als laatste kaart
		default:
			score -= 0.22 // 3-7 als laatste kaart: bijna verloren
		}
	}

	// Met 2 kaarten over: de laagste kaart bepaalt het risico.
	// Paar = altijd in 1 zet kwijt → sterk.
	// Twee verschillende singles → de lage is een blok: penalty naar gelang rank.
	if myCount == 2 && wilds == 0 && resets == 0 {
		ranks := []Rank{}
		for _, c := range hand.Cards {
			if !c.IsSpecial() {
				ranks = append(ranks, c.Rank)
			}
		}
		if len(ranks) == 2 {
			if ranks[0] == ranks[1] {
				// Paar: altijd in 1 zet kwijt — sterk eindspel
				score += 0.10
			} else {
				// Twee verschillende singles: laagste kaart is het risico
				low := ranks[0]
				if ranks[1] < low {
					low = ranks[1]
				}
				// Hoe lager de laagste kaart, hoe groter het verliesrisico
				// (tegenstander kan doubles spelen waartegen je niet kunt)
				switch {
				case low <= RankFive:
					score -= 0.16
				case low <= RankEight:
					score -= 0.09
				case low <= RankTen:
					score -= 0.04
				}
			}
		}
	}
	// ────────────────────────────────────────────────────────────────────────
	// ── FIX 3: Sluitpatroon bij 3 kaarten ──────────────────────────────────
	// Bij 3 kaarten zijn er goede en slechte patronen:
	//   GOED:  paar + 1 hogere single  → dump single, dan paar als afsluiter
	//          paar + wildcard/joker   → bijna zekere win
	//          triple                  → 1 zet klaar
	//   SLECHT: 3 verschillende singles waarvan ≥1 laag → moeilijk te manoeuvreren
	//           de tegenstander kan doubles spelen waartegen je geen antwoord hebt
	if myCount == 3 && wilds == 0 && resets == 0 {
		rankCounts3 := map[Rank]int{}
		for _, c := range hand.Cards {
			if !c.IsSpecial() {
				rankCounts3[c.Rank]++
			}
		}
		hasPair3 := false
		hasTriple3 := false
		var pairRank3 Rank
		var singles3 []Rank
		for r, cnt := range rankCounts3 {
			if cnt >= 3 {
				hasTriple3 = true
			} else if cnt == 2 {
				hasPair3 = true
				pairRank3 = r
			} else {
				singles3 = append(singles3, r)
			}
		}
		switch {
		case hasTriple3:
			score += 0.12 // triple: altijd in 1 zet klaar
		case hasPair3 && len(singles3) == 1:
			single3 := singles3[0]
			if single3 < pairRank3 {
				// Single lager dan paar: dump single, sluit met paar → goed patroon
				score += 0.08
			} else {
				// Single hoger dan paar: onhandig — paar is moeilijk te dumpen
				score += 0.02
			}
		default:
			// 3 verschillende singles: kwetsbaar voor doubles
			var minRank3 Rank = RankAce + 1
			for r := range rankCounts3 {
				if r < minRank3 {
					minRank3 = r
				}
			}
			switch {
			case minRank3 <= RankFive:
				score -= 0.10
			case minRank3 <= RankEight:
				score -= 0.05
			}
		}
	}
	// ────────────────────────────────────────────────────────────────────────
	// niet zo groot dat het kaartdifferentieel overschaduwt.
	// Oude waarde (4.0x = 0.34 + 0.45 voor Joker = 0.79!) was absurd groot
	// en maakte dat PASS kunstmatig goed scoorde in MCTS rollouts,
	// omdat rolloutevaluaties met open-ronde-posities altijd ~1.0 teruggaven.
	if gs.Round.IsOpen && gs.CurrentTurn == myID {
		if activePlayerCount(gs) <= 2 {
			score += 0.20 // 2-speler: open ronde = groot tempo-voordeel
		} else {
			score += 0.085 * 1.2
		}
		if hand.CountResets() > 0 {
			score += 0.08
		}
	} else if gs.Round.IsOpen && gs.CurrentTurn != myID && activePlayerCount(gs) <= 2 {
		score -= 0.14 // 2-speler: tegenstander heeft open ronde = jij bent in het nadeel
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

func (e *Engine) pickBest(root *mctsNode, myID int, tableRank Rank) (Move, MoveEval) {
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
				// Prefer higher WR; tiebreak (within 1%) by cheaper move
				better := wr > bestWR+0.01 ||
					(wr >= bestWR-0.01 && bestNode != nil && preferCheaperMove(ch.move, bestNode.move, tableRank))
				if better {
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
// Bij win-rates binnen 1%: voorkeur voor goedkoopste zet (laagste rank, dan minste kaarten).
func bestNonPassFromDetails(details []MoveDetail, tableRank Rank) (Move, bool) {
	bestWR := -1.0
	var bestMove Move
	found := false
	for _, d := range details {
		if !d.Move.IsPass && d.Visits > 0 {
			better := d.WinRate > bestWR+0.01 ||
				(d.WinRate >= bestWR-0.01 && found && preferCheaperMove(d.Move, bestMove, tableRank))
			if better {
				bestWR = d.WinRate
				bestMove = d.Move
				found = true
			}
		}
	}
	return bestMove, found
}

// overridePass vervangt een MCTS-PASS door de beste speel-zet als de situatie urgent is.
// Retourneert de finale zet en een bijgewerkte MoveEval (score/visits van de override-zet).
func overridePass(bestMove Move, eval MoveEval, gs *GameState, rootFiltered []Move) (Move, MoveEval) {
	if !bestMove.IsPass {
		return bestMove, eval
	}
	myID := gs.CurrentTurn
	myCards := gs.Hands[myID].Count()
	oppCards := minOppHandCount(gs, myID)
	lowTable := !gs.Round.IsOpen && gs.Round.TableRank <= RankSeven
	twoPlayer := activePlayerCount(gs) <= 2
	threatenedMulti := !twoPlayer && oppCards <= 6 && myCards-oppCards >= 4
	urgent := myCards-oppCards >= 4 || (twoPlayer && oppCards <= 5) || lowTable || twoPlayer || threatenedMulti
	if !urgent {
		return bestMove, eval
	}
	passWR := eval.Score
	threshold := 0.03
	if twoPlayer {
		threshold = 0.15
	} else if threatenedMulti {
		threshold = 0.12
	}
	if lowTable && threshold < 0.07 {
		threshold = 0.07
	}
	if twoPlayer && lowTable {
		threshold = 0.22
	}
	withStats := func(d MoveDetail) MoveEval {
		return MoveEval{Score: d.WinRate, Visits: d.Visits, Details: eval.Details, TotalIters: eval.TotalIters, ElapsedMs: eval.ElapsedMs}
	}
	// Harde override: naturelle zet + lage tafel (≤5) + 2-speler → altijd spelen.
	if twoPlayer && !gs.Round.IsOpen && gs.Round.TableRank > 0 && gs.Round.TableRank <= RankFive {
		for _, m := range rootFiltered {
			if m.IsPass {
				continue
			}
			allNatural := true
			for _, c := range m.Cards {
				if c.IsWild() || c.IsReset() {
					allNatural = false
					break
				}
			}
			if allNatural {
				if bm, ok := bestNonPassFromDetails(eval.Details, gs.Round.TableRank); ok {
					for _, d := range eval.Details {
						if MovesEqual(d.Move, bm) && d.WinRate >= 0.02 {
							return bm, withStats(d)
						}
					}
				}
				break
			}
		}
	}
	// Zachte override: threshold-gebaseerd.
	if m, ok := bestNonPassFromDetails(eval.Details, gs.Round.TableRank); ok {
		for _, d := range eval.Details {
			if MovesEqual(d.Move, m) && d.WinRate >= 0.02 && d.WinRate >= passWR-threshold {
				return m, withStats(d)
			}
		}
	}
	return bestMove, eval
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
	eof     bool
}

func NewReader() *Reader {
	return &Reader{scanner: bufio.NewScanner(os.Stdin)}
}

func (r *Reader) ReadLine(prompt string) string {
	fmt.Print(prompt)
	if r.scanner.Scan() {
		return strings.TrimSpace(r.scanner.Text())
	}
	r.eof = true
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
	minIters   int
	maxIters   int
	thinkMs    int
}

func main() {
	reader := NewReader()
	cfg := settings{numThreads: 8, minIters: 20000, maxIters: 200000, thinkMs: 2000}
	for {
		PrintHeader("AZEN Engine UPDATE 19")
		fmt.Println("Welkom bij de AZEN kaartspel engine!")
		fmt.Println()
		fmt.Printf("  [0] Instellingen  (threads: %d | iter: %d–%d | %dms)\n", cfg.numThreads, cfg.minIters, cfg.maxIters, cfg.thinkMs)
		fmt.Println("  [1] Spelen  - Engine suggereert zetten voor jou")
		fmt.Println("  [2] Analyse - Analyseer een gespeelde partij")
		fmt.Println("  [3] Simuleer - Kijk hoe de engine tegen zichzelf speelt")
		fmt.Println("  [4] Kaartenset - Rating van een hand / hand voor een rating / partij-ratings")
		fmt.Println("  [q] Afsluiten")
		fmt.Println()
		modeStr := reader.ReadLine("Kies modus (0/1/2/3/4, q=stop): ")
		if reader.eof {
			return
		}
		if s := strings.ToLower(strings.TrimSpace(modeStr)); s == "q" || s == "quit" || s == "stop" || s == "exit" {
			fmt.Println("Tot ziens!")
			return
		}
		mode, _ := strconv.Atoi(modeStr)
		switch mode {
		case 0:
			cfg = settingsMenu(reader, cfg)
			continue
		case 1:
			playMode(reader, cfg)
		case 2:
			analyzeMode(reader, cfg)
		case 3:
			simulateMode(reader, cfg)
		case 4:
			dealFinderMode(reader, cfg)
		default:
			fmt.Println("Onbekende keuze — kies 0, 1, 2, 3, 4 of q.")
			continue
		}
		if reader.eof {
			return
		}
		reader.ReadLine("\nDruk Enter voor het hoofdmenu... ")
	}
}

func settingsMenu(reader *Reader, cfg settings) settings {
	PrintHeader("Instellingen")
	fmt.Printf("Huidige instellingen: %d threads | min %d – max %d iter | %d ms\n\n", cfg.numThreads, cfg.minIters, cfg.maxIters, cfg.thinkMs)

	fmt.Println("Threads bepalen hoeveel parallelle ISMCTS-bomen tegelijk draaien.")
	fmt.Println("Meer threads = sterkere engine binnen dezelfde denktijd.")
	fmt.Println("  1  = sequentieel (origineel gedrag)")
	fmt.Println("  2  = standaard (goed evenwicht, aanbevolen)")
	fmt.Println("  4+ = sterker maar meer CPU-gebruik")
	fmt.Println()
	if n, err := reader.ReadInt(fmt.Sprintf("Aantal threads (huidige: %d): ", cfg.numThreads)); err == nil && n >= 1 {
		if n > 64 {
			n = 64
		}
		cfg.numThreads = n
		fmt.Printf("✅ Threads: %d\n\n", n)
	} else {
		fmt.Printf("Ongewijzigd (%d threads).\n\n", cfg.numThreads)
	}

	fmt.Println("Iteraties bepalen de rekenkwaliteit van de engine.")
	fmt.Println("  Minimum: engine denkt altijd minstens dit aantal iteraties (ook al is de tijd al om).")
	fmt.Println("  Maximum: engine stopt zodra dit bereikt is, ook vóór de tijdslimiet.")
	fmt.Println()
	if n, err := reader.ReadInt(fmt.Sprintf("Minimum iteraties per zet (huidige: %d): ", cfg.minIters)); err == nil && n >= 1 {
		cfg.minIters = n
		fmt.Printf("✅ Minimum iteraties: %d\n", n)
	} else {
		fmt.Printf("Ongewijzigd (%d).\n", cfg.minIters)
	}
	if n, err := reader.ReadInt(fmt.Sprintf("Maximum iteraties per zet (huidige: %d): ", cfg.maxIters)); err == nil && n >= 1 {
		if n < cfg.minIters {
			n = cfg.minIters
			fmt.Printf("⚠️  Maximum kleiner dan minimum — aangepast naar %d.\n", n)
		}
		cfg.maxIters = n
		fmt.Printf("✅ Maximum iteraties: %d\n\n", n)
	} else {
		fmt.Printf("Ongewijzigd (%d).\n\n", cfg.maxIters)
	}

	fmt.Println("Denktijd: engine stopt na deze tijd (tenzij minimum nog niet gehaald is).")
	fmt.Println()
	if n, err := reader.ReadInt(fmt.Sprintf("Denktijd per zet in ms (huidige: %d): ", cfg.thinkMs)); err == nil && n > 0 {
		cfg.thinkMs = n
		fmt.Printf("✅ Denktijd: %d ms\n\n", n)
	} else {
		fmt.Printf("Ongewijzigd (%d ms).\n\n", cfg.thinkMs)
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
	// Als de gebruiker het volledige aantal kaarten invoert in één gok, verwijder eerst
	// automatische aannames (bijv. initiële wildcard-prior) om dubbelingen te voorkomen.
	// Zo klopt len(Suspicions) == HandCounts en werkt KnownOpponentCards correct.
	handCount := tracker.HandCounts[targetID]
	if handCount > 0 && len(parsed) >= handCount {
		tracker.ClearSuspicions(targetID)
	}
	added := tracker.AddSuspicion(targetID, parsed)
	susp := tracker.Suspicions[targetID]
	volledig := len(susp) == handCount
	volledigLabel := ""
	if volledig {
		volledigLabel = " ✅ volledig"
	}
	msg := fmt.Sprintf("🔍 Gok Speler %d heeft: %s  (%d kaart(en) toegevoegd%s, totaal vermoeden: %s)",
		playerNum, CardsToString(parsed), added, volledigLabel, CardsToString(susp))
	if added < len(parsed) {
		msg += fmt.Sprintf("\n   ⚠️  %d kaart(en) niet toegevoegd: al gespeeld of niet meer in pool", len(parsed)-added)
	}
	if volledig {
		msg += fmt.Sprintf("\n   🔍 Hand volledig bekend — kaarten van andere speler(s) worden afgeleid")
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
			deduced := tracker.KnownOpponentCards(i)
			if deduced != nil {
				h := NewHand(deduced)
				h.Sort()
				label := "[afgeleid]"
				if len(tracker.Suspicions[i]) == count {
					label = "[gok volledig]"
				}
				handDisplay = h.String() + " " + label
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
	engConfig := DefaultConfig(numPlayers)
	engConfig.MinIterations = cfg.minIters
	engConfig.Iterations = cfg.maxIters
	engConfig.MaxTime = time.Duration(cfg.thinkMs) * time.Millisecond
	engConfig.NumWorkers = cfg.numThreads
	if numPlayers > 2 {
		pressureStr := reader.ReadLine(fmt.Sprintf("Druk op welke speler? (0 = niemand, 1-%d, niet jezelf): ", numPlayers))
		if n, err := strconv.Atoi(strings.TrimSpace(pressureStr)); err == nil && n >= 1 && n <= numPlayers && n-1 != myPlayer {
			engConfig.PressureTarget = n - 1
			fmt.Printf("🎯 Druk op Speler %d — jij speelt om P%d te laten verliezen.\n", n, n)
		}
	}
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
			legalNow := gs.GetLegalMoves()
			isForcedPass := len(legalNow) == 1 && legalNow[0].IsPass
			if isForcedPass {
				fmt.Println("\n⏩ Geen speelbare kaarten — automatisch pas.")
				move := PassMove(myPlayer)
				tracker.RecordPass(myPlayer, gs.Round)
				gs.ApplyMove(move)
				tracker.RecordMove(move)
				fmt.Printf("✅ Pas.\n\n")
				continue
			}
			fmt.Println("\n🤔 Engine denkt na...")
			bestMove, eval := eng.BestMove(gs, tracker)
			if eval.ForcedWinDepth > 0 {
				fmt.Printf("\n♟️  Gedwongen winst in %d beurt(en)!\n", eval.ForcedWinDepth)
				fmt.Printf("💡 Engine suggereert: %s  [%s]\n\n", FormatMove(bestMove), eval.StatsString())
			} else {
				fmt.Printf("\n💡 Engine suggereert: %s (winst: %s)  [%s]\n\n",
					FormatMove(bestMove), FormatScore(eval.Score), eval.StatsString())
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
					if eval.ForcedWinDepth > 0 {
						fmt.Printf("\n♟️  Gedwongen winst in %d beurt(en)!\n", eval.ForcedWinDepth)
						fmt.Printf("💡 Nieuwe suggestie: %s  [%s]\n\n", FormatMove(bestMove), eval.StatsString())
					} else {
						fmt.Printf("\n💡 Nieuwe suggestie: %s (winst: %s)  [%s]\n\n",
							FormatMove(bestMove), FormatScore(eval.Score), eval.StatsString())
					}
					continue
				case "hint":
					fmt.Printf("💡 Suggestie: %s (winst: %s)  [%s]\n",
						FormatMove(bestMove), FormatScore(eval.Score), eval.StatsString())
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
	PrintHeader("Analyse")
	fmt.Println("Voer de partij in één keer in.")
	fmt.Println("Zetten: spatie-gescheiden tokens die alterneren tussen spelers.")
	fmt.Println("Aas+vervolg: schrijf als '1/5' (aas, dan 5 in dezelfde beurt).")
	fmt.Println("Pas: p of - of pass")

	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}

	// Welke speler(s) analyseren (kommagescheiden of leeg = alle)
	analyzeStr := reader.ReadLine(fmt.Sprintf("Welke speler(s) analyseren? (bv. '1' of '1,3', leeg = alle %d): ", numPlayers))
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

	// Startkaarten per speler (alleen aantallen)
	cardsPerPlayer := make([]int, numPlayers)
	for i := 0; i < numPlayers; i++ {
		cardsPerPlayer[i] = 18
		if n, err := reader.ReadInt(fmt.Sprintf("Aantal startkaarten Speler %d (standaard 18): ", i+1)); err == nil && n > 0 {
			cardsPerPlayer[i] = n
		}
	}

	startPlayer := 0
	if p, err := reader.ReadInt(fmt.Sprintf("Wie begint (spelernummer 1-%d): ", numPlayers)); err == nil && p >= 1 && p <= numPlayers {
		startPlayer = p - 1
	}

	// Druk-doelwit
	pressureTarget := -1
	pressureStr := reader.ReadLine(fmt.Sprintf("Druk op welke speler? (0 = niemand, 1-%d): ", numPlayers))
	if n, err := strconv.Atoi(strings.TrimSpace(pressureStr)); err == nil && n >= 1 && n <= numPlayers {
		pressureTarget = n - 1
	}

	fmt.Println()
	fmt.Printf("Voer alle zetten in als spatie-gescheiden tokens (bv: 8888 p 33 44 66 jj p 4 5 9 1/5 ...)\n")
	fmt.Printf("Voeg de resterende kaarten van de verliezer toe als laatste token (bv: ... 9 45678).\n")
	movesLine := reader.ReadLine("Zetten: ")
	tokens := strings.Fields(movesLine)
	if len(tokens) == 0 {
		fmt.Println("Geen zetten ingevoerd.")
		return
	}

	// Reconstrueer starthanden vanuit de zetten
	hands, deadCards, _, reconErr := reconstructStartingHands(numPlayers, cardsPerPlayer, startPlayer, tokens)
	if reconErr != nil {
		fmt.Printf("Fout bij reconstructie: %v\n", reconErr)
		return
	}

	// Ontbrekende resterende kaarten opvragen
	for p, h := range hands {
		if h.Count() < cardsPerPlayer[p] {
			need := cardsPerPlayer[p] - h.Count()
			fmt.Printf("\nSpeler %d heeft nog %d resterende kaarten die niet in de zetten stonden.\n", p+1, need)
			for {
				parsed, err2 := reader.ReadCards(fmt.Sprintf("Resterende kaarten Speler %d (%d kaarten): ", p+1, need))
				if err2 != nil {
					fmt.Printf("Fout: %v\n", err2)
					continue
				}
				if len(parsed) != need {
					fmt.Printf("Verwacht %d, kreeg %d\n", need, len(parsed))
					continue
				}
				hands[p] = NewHand(append(hands[p].Cards, parsed...))
				for _, c := range parsed {
					for i, dc := range deadCards {
						if dc == c {
							deadCards = append(deadCards[:i], deadCards[i+1:]...)
							break
						}
					}
				}
				break
			}
		}
	}

	gs := NewGameWithHands(hands, deadCards, startPlayer)

	if pressureTarget >= 0 {
		fmt.Printf("\n🎯 Druk op Speler %d — anderen spelen samenwerkend om P%d te laten verliezen.\n",
			pressureTarget+1, pressureTarget+1)
	}

	baseEngConfig := DefaultConfig(numPlayers)
	baseEngConfig.OmniscientMode = true
	baseEngConfig.MinIterations = cfg.minIters
	baseEngConfig.Iterations = cfg.maxIters
	baseEngConfig.MaxTime = time.Duration(cfg.thinkMs) * time.Millisecond
	baseEngConfig.NumWorkers = cfg.numThreads

	trackers := make([]*KnowledgeTracker, numPlayers)
	for p := 0; p < numPlayers; p++ {
		trackers[p] = NewKnowledgeTracker(numPlayers, p, gs.Hands[p], gs.DeadCards)
	}

	// Gedeelde transpositietabel voor de exacte 2-speler eindspel-oplossing:
	// éénmaal opgelost, daarna zijn alle latere stellingen cache-hits.
	if numPlayers == 2 {
		analyzeSolveTT = make(map[uint64]sfEntry, 1<<21)
		defer func() { analyzeSolveTT = nil }()
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

		doAnalysis := analyzeAll || analyzePlayers[playerID]

		// Engine config met druk-instelling voor deze speler
		engConfig := baseEngConfig
		underPressure := pressureTarget >= 0 && playerID != pressureTarget
		if underPressure {
			engConfig.PressureTarget = pressureTarget
		}

		var bestMove Move
		var bestEval MoveEval
		var actualDetail MoveDetail
		var bestLabel string
		var bestDelayMove *Move
		var maxDelay int
		var playedForcedWinDepth int
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
			} else if bestEval.ForcedWinDepth > 0 {
				simCheck := gs.Clone()
				simCheck.ApplyMove(move)
				if simCheck.GameOver && simCheck.Winner == playerID {
					actualDetail = MoveDetail{Move: move, WinRate: 1.0, Visits: 1}
					playedForcedWinDepth = 1
				} else if !simCheck.GameOver {
					tc := 0
					for _, h := range gs.Hands {
						tc += h.Count()
					}
					nodes := 0
					d := forcedWinDepth(simCheck, playerID, tc*4, &nodes, 500000)
					if d >= 0 {
						actualDetail = MoveDetail{Move: move, WinRate: 1.0, Visits: 1}
						playedForcedWinDepth = d + 1
					} else {
						actualDetail = eng.AnalyzeMove(gs, tracker, move)
					}
				} else {
					actualDetail = eng.AnalyzeMove(gs, tracker, move)
				}
			} else {
				actualDetail = eng.AnalyzeMove(gs, tracker, move)
			}
			// Gedwongen-verlies detectie: draai altijd wanneer de engine geen
			// gedwongen winst voor deze speler vond (niet alleen als de gespeelde
			// zet buiten de MCTS-evaluatie viel).
			if bestDelayMove == nil && bestEval.ForcedWinDepth <= 0 && playedForcedWinDepth == 0 {
				bestDelayMove, maxDelay = findForcedLoss(gs, baseEngConfig.OmniscientMode)
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
		// Stelling vóór de zet bewaren: oppWinDepthAfterMove past de zet zelf toe.
		preGS := gs.Clone()
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
			forcedWin := bestEval.ForcedWinDepth > 0
			forcedLoss := bestDelayMove != nil
			playedIsBest := MovesEqual(bestMove, move)
			playedIsBestDelay := forcedLoss && MovesEqual(*bestDelayMove, move)
			tempoOverride := !playedIsBest && move.IsPass && !bestMove.IsPass
			var diff float64
			emoji := "✅"
			if forcedLoss {
				if !playedIsBestDelay {
					emoji = "⚠️ "
					playedDelay := oppWinDepthAfterMove(preGS, move)
					if playedDelay >= 0 && maxDelay-playedDelay >= 2 {
						emoji = "❌"
					}
				}
			} else if !playedIsBest {
				diff = bestEval.Score - actualDetail.WinRate
				if forcedWin && playedForcedWinDepth == 0 {
					emoji = "❌"
				} else if diff > 0.15 {
					emoji = "❌"
				} else if diff > 0.02 || tempoOverride {
					emoji = "⚠️ "
				}
			}
			icon := emoji
			if playedIsBest || playedIsBestDelay {
				icon = "📘"
			}

			// Labelnamen afhankelijk van druk-modus
			scoreName := "score"
			bestPrefix := "Beste was"
			if underPressure {
				scoreName = "Score met druk"
				bestPrefix = "Beste drukzet"
			}

			var scoreStr string
			switch {
			case forcedLoss:
				playedDelay := oppWinDepthAfterMove(preGS, move)
				if playedIsBestDelay || playedDelay == maxDelay {
					scoreStr = fmt.Sprintf("verlies in %d zetten — beste weerstand", maxDelay)
				} else if playedDelay >= 0 {
					scoreStr = fmt.Sprintf("verlies in %d zetten", playedDelay)
				} else {
					scoreStr = scoreName + ": onbekend"
				}
			case forcedWin && (playedIsBest || playedForcedWinDepth > 0):
				depth := bestEval.ForcedWinDepth
				if playedForcedWinDepth > 0 {
					depth = playedForcedWinDepth
				}
				scoreStr = fmt.Sprintf("winst in %d zetten", depth)
			default:
				scoreStr = fmt.Sprintf("%s: %.1f%%", scoreName, actualDetail.WinRate*100)
			}
			fmt.Printf("%s Z%d P%d: %s (%s)  [%s]\n", icon, moveNum, playerID+1, moveLabel, scoreStr, bestEval.StatsString())
			if forcedLoss {
				playedDelay := oppWinDepthAfterMove(preGS, move)
				if playedIsBestDelay {
					fmt.Printf("   ⏳ Beste weerstand — verlies in %d beurt(en) van tegenstander\n", maxDelay)
				} else {
					fmt.Printf("   ⏳ Beste weerstand: %s (verlies in %d i.p.v. %d beurt(en))\n",
						FormatMove(*bestDelayMove), maxDelay, playedDelay)
				}
			} else if forcedWin {
				if playedIsBest {
					fmt.Printf("   ♟️  Gedwongen winst in %d beurt(en)!\n", bestEval.ForcedWinDepth)
				} else if playedForcedWinDepth > 0 {
					fmt.Printf("   ♟️  Winst in %d zetten (snelste: %s in %d zetten)\n",
						playedForcedWinDepth, bestLabel, bestEval.ForcedWinDepth)
				} else {
					fmt.Printf("   ♟️  Gedwongen winst in %d beurt(en) gemist! Beste was: %s\n",
						bestEval.ForcedWinDepth, bestLabel)
				}
			} else {
				if tempoOverride {
					fmt.Printf("   ⚡ Tempo-verlies: engine zou spelen — %s\n", bestLabel)
				} else {
					showBest := !playedIsBest && (diff > 0.02 || (bestEval.Score > 0.90 && diff > 0.005))
					if showBest {
						fmt.Printf("   %s: %s (%s: %.1f%%, verschil: %.1f%%)\n",
							bestPrefix, bestLabel, scoreName, bestEval.Score*100, diff*100)
					}
				}
			}
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
	fmt.Println("\nAnalyse klaar.")
}

// ─── gameSim: lichtgewicht turn-simulator voor hand-reconstructie ────────────

type gameSim struct {
	numPlayers   int
	finished     []bool
	cardCount    []int
	roundIsOpen  bool
	roundCount   int
	tableRank    Rank
	lastPlayerID int
	consecPasses int
	currentTurn  int
	GameOver     bool
	ranking      []int
}

func newGameSim(numPlayers int, cardsPerPlayer []int, startPlayer int) *gameSim {
	s := &gameSim{
		numPlayers:   numPlayers,
		finished:     make([]bool, numPlayers),
		cardCount:    make([]int, numPlayers),
		roundIsOpen:  true,
		lastPlayerID: startPlayer,
		currentTurn:  startPlayer,
	}
	for i, n := range cardsPerPlayer {
		s.cardCount[i] = n
	}
	return s
}

func (s *gameSim) activeCount() int {
	c := 0
	for _, f := range s.finished {
		if !f {
			c++
		}
	}
	return c
}

func (s *gameSim) nextActive(from int) int {
	for i := 1; i <= s.numPlayers; i++ {
		next := (from + i) % s.numPlayers
		if !s.finished[next] {
			return next
		}
	}
	return from
}

func (s *gameSim) passThreshold() int {
	active := s.activeCount()
	if s.finished[s.lastPlayerID] {
		return active
	}
	return active - 1
}

func (s *gameSim) applyPass() {
	s.consecPasses++
	if s.consecPasses >= s.passThreshold() {
		lastPID := s.lastPlayerID
		s.roundIsOpen = true
		s.roundCount = 0
		s.tableRank = 0
		s.consecPasses = 0
		if s.finished[lastPID] {
			s.currentTurn = s.nextActive(lastPID)
		} else {
			s.currentTurn = lastPID
		}
	} else {
		s.currentTurn = s.nextActive(s.currentTurn)
	}
}

// applyPlay verwerkt een zet en geeft true als het spel nu voorbij is.
func (s *gameSim) applyPlay(m Move) bool {
	pid := m.PlayerID
	n := len(m.Cards)
	s.cardCount[pid] -= n
	if s.cardCount[pid] < 0 {
		s.cardCount[pid] = 0
	}

	if s.cardCount[pid] == 0 {
		s.finished[pid] = true
		s.ranking = append(s.ranking, pid)
		if s.activeCount() <= 1 {
			for i, f := range s.finished {
				if !f {
					s.ranking = append(s.ranking, i)
					s.finished[i] = true
					break
				}
			}
			s.GameOver = true
			return true
		}
		if m.ContainsReset() {
			s.roundIsOpen = true
			s.roundCount = 0
			s.tableRank = 0
			s.consecPasses = 0
			s.lastPlayerID = pid
			s.currentTurn = s.nextActive(pid)
		} else {
			s.tableRank = m.EffectiveRank(s.tableRank)
			s.roundIsOpen = false
			s.roundCount = n
			s.consecPasses = 0
			s.lastPlayerID = pid
			s.currentTurn = s.nextActive(pid)
		}
		return false
	}

	if m.ContainsReset() {
		s.roundIsOpen = true
		s.roundCount = 0
		s.tableRank = 0
		s.consecPasses = 0
		s.lastPlayerID = pid
		s.currentTurn = pid // speler speelt opnieuw na reset
		return false
	}

	effectRank := m.EffectiveRank(s.tableRank)
	if s.roundIsOpen {
		s.roundIsOpen = false
		s.roundCount = n
		s.tableRank = effectRank
		s.lastPlayerID = pid
		s.consecPasses = 0
	} else {
		s.tableRank = effectRank
		s.lastPlayerID = pid
		s.consecPasses = 0
	}
	s.currentTurn = s.nextActive(pid)
	return false
}

// reconstructStartingHands bouwt starthanden op vanuit de zetten-reeks.
// tokens mag als laatste token de resterende kaarten van de verliezer bevatten.
// Geeft handen, dode kaarten, en het aantal verbruikte tokens terug.
func reconstructStartingHands(numPlayers int, cardsPerPlayer []int, startPlayer int, tokens []string) (
	[]*Hand, []Card, int, error,
) {
	sim := newGameSim(numPlayers, cardsPerPlayer, startPlayer)
	playedCards := make([][]Card, numPlayers)
	gameEndTI := len(tokens)

	for ti := 0; ti < len(tokens) && !sim.GameOver; ti++ {
		token := tokens[ti]
		mainStr, followStr, hasFollow := strings.Cut(token, "/")
		mainLower := strings.ToLower(strings.TrimSpace(mainStr))
		currentPlayer := sim.currentTurn

		if mainLower == "p" || mainLower == "pass" || mainLower == "-" {
			sim.applyPass()
		} else {
			parsed, err := ParseCards(mainStr)
			if err != nil {
				return nil, nil, ti, fmt.Errorf("token %d (%q): %v", ti+1, token, err)
			}
			move := Move{PlayerID: currentPlayer, Cards: parsed}
			over := sim.applyPlay(move)
			playedCards[currentPlayer] = append(playedCards[currentPlayer], parsed...)
			if over {
				gameEndTI = ti + 1
				break
			}
			// Follow-zet na reset (speler speelt opnieuw)
			if hasFollow && !sim.GameOver && sim.currentTurn == currentPlayer {
				followParsed, err2 := ParseCards(strings.TrimSpace(followStr))
				if err2 == nil {
					followMove := Move{PlayerID: currentPlayer, Cards: followParsed}
					over2 := sim.applyPlay(followMove)
					playedCards[currentPlayer] = append(playedCards[currentPlayer], followParsed...)
					if over2 {
						gameEndTI = ti + 1
						break
					}
				}
			}
		}
	}

	// Verliezer = laatste in de ranking
	loserID := -1
	if len(sim.ranking) == numPlayers {
		loserID = sim.ranking[len(sim.ranking)-1]
	} else if len(sim.ranking) > 0 {
		// Spel niet volledig afgelopen: zoek speler met kaarten over
		for p, f := range sim.finished {
			if !f {
				loserID = p
				break
			}
		}
	}

	// Controleer of het volgende token de resterende kaarten van de verliezer zijn
	var remainderCards []Card
	if gameEndTI < len(tokens) {
		if parsed, err2 := ParseCards(tokens[gameEndTI]); err2 == nil && len(parsed) > 0 {
			remainderCards = parsed
			gameEndTI++
		}
	}

	// Bouw starthanden
	hands := make([]*Hand, numPlayers)
	for p := 0; p < numPlayers; p++ {
		startCards := make([]Card, len(playedCards[p]))
		copy(startCards, playedCards[p])
		if p == loserID {
			startCards = append(startCards, remainderCards...)
		}
		hands[p] = NewHand(startCards)
	}

	// Dode kaarten = volledig deck minus alle starthanden
	numDecks := 1
	if numPlayers == 4 {
		numDecks = 2
	}
	var allDeckCards []Card
	if numDecks == 1 {
		allDeckCards = NewDeck().Cards
	} else {
		allDeckCards = NewMultiDeck(2).Cards
	}
	used := make(map[Card]int)
	for _, h := range hands {
		for _, c := range h.Cards {
			used[c]++
		}
	}
	var deadCards []Card
	for _, c := range allDeckCards {
		if used[c] > 0 {
			used[c]--
		} else {
			deadCards = append(deadCards, c)
		}
	}

	return hands, deadCards, gameEndTI, nil
}


func simulateMode(reader *Reader, cfg settings) {
	PrintHeader("Simulatie Modus")
	fmt.Println("Kijk hoe de engine tegen zichzelf speelt!")
	fmt.Println()
	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}
	fmt.Println()
	fmt.Println("  [1] Random kaarten")
	fmt.Println("  [2] Kaarten zelf kiezen")
	fmt.Println()
	kaartKeuze := reader.ReadLine("Kies optie (1/2): ")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var gs *GameState

	if strings.TrimSpace(kaartKeuze) == "2" {
		// Kaarten zelf kiezen
		hands := make([]*Hand, numPlayers)
		var allDealt []Card
		for i := 0; i < numPlayers; i++ {
			for {
				input := reader.ReadLine(fmt.Sprintf("Kaarten Speler %d (bijv. 22333667777xxk1110): ", i+1))
				parsed, err := ParseCards(strings.TrimSpace(input))
				if err != nil {
					fmt.Printf("⚠️  Ongeldige kaarten: %v — probeer opnieuw.\n", err)
					continue
				}
				hands[i] = NewHand(parsed)
				allDealt = append(allDealt, parsed...)
				break
			}
		}
		// Bereken dode kaarten (kaarten die niet uitgedeeld zijn)
		fullDeck := NewDeck()
		deckCards := fullDeck.Cards
		dealtCopy := make([]Card, len(allDealt))
		copy(dealtCopy, allDealt)
		var dead []Card
		for _, dc := range deckCards {
			found := false
			for j, ec := range dealtCopy {
				if ec.Rank == dc.Rank {
					dealtCopy = append(dealtCopy[:j], dealtCopy[j+1:]...)
					found = true
					break
				}
			}
			if !found {
				dead = append(dead, dc)
			}
		}
		startPlayer := 0
		if sp, err := reader.ReadInt(fmt.Sprintf("Welke speler begint? (1-%d): ", numPlayers)); err == nil && sp >= 1 && sp <= numPlayers {
			startPlayer = sp - 1
		}
		gs = NewGameWithHands(hands, dead, startPlayer)
	} else {
		// Random kaarten
		startPlayer := 0
		gs = NewGame(numPlayers, rng, startPlayer)
	}
	fmt.Println("\nStarthanden:")
	for i := 0; i < numPlayers; i++ {
		fmt.Printf("Speler %d: %s\n", i+1, gs.Hands[i])
	}
	fmt.Println()
	trackers := make([]*KnowledgeTracker, numPlayers)
	engines := make([]*Engine, numPlayers)
	for i := 0; i < numPlayers; i++ {
		engConfig := DefaultConfig(numPlayers)
		engConfig.MinIterations = cfg.minIters
		engConfig.Iterations = cfg.maxIters
		engConfig.MaxTime = time.Duration(cfg.thinkMs) * time.Millisecond
		engConfig.NumWorkers = cfg.numThreads
		trackers[i] = NewKnowledgeTracker(numPlayers, i, gs.Hands[i], gs.DeadCards)
		// Overschrijf HandCounts met werkelijke startgroottes (belangrijk bij custom handen)
		for j := 0; j < numPlayers; j++ {
			trackers[i].HandCounts[j] = gs.Hands[j].Count()
		}
		engines[i] = NewEngine(engConfig)
	}
	prevFinished := 0
	moveNum := 0
	for !gs.GameOver {
		moveNum++
		playerID := gs.CurrentTurn
		eng := engines[playerID]
		var bestMove Move
		var eval MoveEval
		legalSim := gs.GetLegalMoves()
		if len(legalSim) == 1 && legalSim[0].IsPass {
			bestMove = PassMove(playerID)
		} else {
			bestMove, eval = eng.BestMove(gs, trackers[playerID])
		}
		fmt.Printf("Zet %d | Speler %d: %s (score: %.1f%%)  [%s] | Kaarten:",
			moveNum, playerID+1, FormatMove(bestMove), eval.Score*100, eval.StatsString())
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

// ═══════════════════════════════════════════════════════════════
// KAARTENSET RATING  (modus 4)
// ═══════════════════════════════════════════════════════════════
//
// hands.txt is een brede steekproef over het VOLLEDIGE spectrum van mogelijke
// handen (van de zwakst denkbare tot 00221111KKKKQQQQJJ), gesorteerd op ruwe
// score. Elke hand krijgt een percentiel p (positie in dat spectrum) en daaruit
// een rating:
//
//     p <= 0,20 :  rating = 5000 * p               (lineair, 0 .. 1000)
//     p  > 0,20 :  rating = 1000 * 2^(10*(p-0,20))
//
// Zo verdubbelt de rating zelf elke 0,1 percentiel boven p0,2:
//   p0,2 -> 1000    p0,3 -> 2000    p0,4 -> 4000    p0,5 -> 8000
//   p0,6 -> 16000   p0,7 -> 32000   p0,8 -> 64000   p0,9 -> 128000   p1,0 -> 256000
// Een doorsnee geschud deel zit rond p0,24 -> rating ~1320.
// Geen simulatie: puur sorteren op score en de rating toekennen. De theoretische
// topkaart krijgt gegarandeerd rating 256000.
//
// De modus werkt twee kanten op: een hand invoeren en de rating opvragen, of
// een rating opgeven en een passende hand krijgen.

const handsFile = "hands.txt"
const handsHeader = "# AZEN kaartenset -- spectrum van mogelijke handen -- <kaarten> <score> <rating>"
const handsSampleN = 8000
const ratingMax = 256000 // rating bij p = 1,0

// handScore geeft de ruwe sterkte van een hand.
// Per kaart: 3..J = 1..9, Q 11, K 13, aas 15, twee 17, joker 20.
// Setbonus (hoe meer je in één zet kwijt kunt): paar +4, triple +10, quad +20.
// Dubbele joker of dubbele twee: nog eens +6 (wildcards als paar zijn extra sterk).
func handScore(cards []Card) int {
	val := func(r Rank) int {
		switch r {
		case RankQueen:
			return 11
		case RankKing:
			return 13
		case RankAce:
			return 15
		case RankTwo:
			return 17
		case RankJoker:
			return 20
		default:
			return int(r) - 2 // 3->1 ... 9->7, X->8, J->9
		}
	}
	total := 0
	rc := map[Rank]int{}
	for _, c := range cards {
		total += val(c.Rank)
		rc[c.Rank]++
	}
	for r, n := range rc {
		switch {
		case n >= 4:
			total += 20
		case n == 3:
			total += 10
		case n == 2:
			total += 4
		}
		if (r == RankJoker || r == RankTwo) && n >= 2 {
			total += 6
		}
	}
	return total
}

// handComposition telt joker/twee/aas/hoge kaarten en set-groepen.
func handComposition(cards []Card) (jokers, twos, aces, high, sets int) {
	rc := map[Rank]int{}
	for _, c := range cards {
		rc[c.Rank]++
		switch c.Rank {
		case RankJoker:
			jokers++
		case RankTwo:
			twos++
		case RankAce:
			aces++
		}
		if !c.IsSpecial() && c.Rank >= RankJack {
			high++
		}
	}
	for _, n := range rc {
		if n >= 2 {
			sets++
		}
	}
	return
}

type scoredHand struct {
	cards []Card
	raw   int
}

func handToTokens(cards []Card) string {
	cp := append([]Card{}, cards...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Rank < cp[j].Rank })
	var sb strings.Builder
	for _, c := range cp {
		sb.WriteString(c.RankStr())
	}
	return sb.String()
}

// De vaste ijkpunten voor score 100 en 0. Best = het door de gebruiker
// opgegeven voorbeeld (respecteert de regel dat elke speler minstens 1 twee
// houdt, dus max 2 tweeën in eigen hand).
func bestAnchorHand() []Card  { c, _ := ParseCards("00221111kkkkqqqqjj"); return c }
func worstAnchorHand() []Card { c, _ := ParseCards("233334444555566667"); return c }

// randomHand trekt 18 kaarten uit één volledig spel met een sterkte-bias
// (in [-1..1]) en dwingt 1..2 tweeën af (spelregel + realisme; jokers zijn er
// sowieso maar 2).
func randomHand(bias float64, rng *rand.Rand) []Card {
	weight := func(r Rank) float64 {
		s := 0.0
		switch r {
		case RankJoker:
			s = 1.0
		case RankTwo:
			s = 0.9
		case RankAce:
			s = 0.75
		default:
			s = (float64(r) - 8.0) / 6.0 // 3 -> -0.83 ... K -> 0.83
		}
		w := 1.0 + bias*s
		if w < 0.03 {
			w = 0.03
		}
		return w
	}
	pool := append([]Card{}, NewDeck().Cards...)
	var hand []Card
	for len(hand) < 18 && len(pool) > 0 {
		tot := 0.0
		for _, c := range pool {
			tot += weight(c.Rank)
		}
		x := rng.Float64() * tot
		idx, cum := len(pool)-1, 0.0
		for i, c := range pool {
			cum += weight(c.Rank)
			if x <= cum {
				idx = i
				break
			}
		}
		hand = append(hand, pool[idx])
		pool = append(pool[:idx], pool[idx+1:]...)
	}
	countRank := func(rank Rank) int {
		n := 0
		for _, c := range hand {
			if c.Rank == rank {
				n++
			}
		}
		return n
	}
	for countRank(RankTwo) < 1 {
		wi, wv := -1, 999
		for i, c := range hand {
			if c.IsSpecial() {
				continue
			}
			if v := int(c.Rank); v < wv {
				wv, wi = v, i
			}
		}
		if wi < 0 {
			break
		}
		hand[wi] = Card{Rank: RankTwo}
	}
	for countRank(RankTwo) > 2 {
		for i, c := range hand {
			if c.Rank == RankTwo {
				hand[i] = Card{Rank: RankThree}
				break
			}
		}
	}
	return hand
}

// generatePopulation bouwt een brede steekproef die het HELE sterktespectrum
// dekt: de twee vaste ijkpunten (zwakst / sterkst denkbaar) plus een sweep over
// het volledige bias-bereik, met extra dichtheid aan de sterke kant zodat de
// rating ook bovenin fijn genoeg verdeeld is. Gesorteerd op score, ontdubbeld.
func generatePopulation(n int, rng *rand.Rand) []scoredHand {
	seen := map[string]bool{}
	var out []scoredHand
	add := func(cards []Card) bool {
		if len(cards) != 18 {
			return false
		}
		tok := handToTokens(cards)
		if seen[tok] {
			return false
		}
		seen[tok] = true
		out = append(out, scoredHand{cards: cards, raw: handScore(cards)})
		return true
	}
	add(worstAnchorHand())
	add(bestAnchorHand())
	for i := 0; i < n; i++ {
		frac := (float64(i) + 0.5) / float64(n)
		bias := -3.2 + 6.4*frac + (rng.Float64()-0.5)*0.7
		add(randomHand(bias, rng))
	}
	for i := 0; i < n/2; i++ {
		add(randomHand(2.0+rng.Float64()*2.5, rng))
	}
	// Top van het spectrum dicht vullen: high-bias trekkingen, alleen de sterkste
	// bijhouden (die zeer zeldzame handen zijn nodig voor een fijne rating bovenin).
	topBest := handScore(bestAnchorHand())
	kept := 0
	for tries := 0; tries < n*40 && kept < n/3; tries++ {
		h := randomHand(3.0+rng.Float64()*3.5, rng)
		if handScore(h) >= topBest-45 && add(h) {
			kept++
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].raw < out[j].raw })
	return out
}

// ratingFromP zet een percentiel (0..1) om naar een rating: lineair tot p0,2
// (rating 1000), daarna verdubbelt de rating per 0,1 percentiel.
func ratingFromP(p float64) int {
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	var r float64
	if p <= 0.2 {
		r = 5000 * p
	} else {
		r = 1000 * math.Pow(2, 10*(p-0.2))
	}
	return int(math.Round(r))
}

// pFromRating is de inverse van ratingFromP.
func pFromRating(r float64) float64 {
	var p float64
	if r <= 1000 {
		p = r / 5000
	} else {
		p = 0.2 + 0.1*math.Log2(r/1000)
	}
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return p
}

type scoreBucket struct {
	score  int
	p      float64
	rating int
	hands  []scoredHand
}

// buildScoreBuckets groepeert de (op score gesorteerde) populatie per ruwe score
// en geeft elke groep een percentiel (midden van de groep) en een rating. De
// zwakste groep krijgt exact p=0 (rating 0), de sterkste exact p=1 (rating 256000).
func buildScoreBuckets(hands []scoredHand) []scoreBucket {
	n := len(hands)
	var out []scoreBucket
	for i := 0; i < n; {
		j := i
		for j < n && hands[j].raw == hands[i].raw {
			j++
		}
		p := (float64(i) + 0.5*float64(j-i)) / float64(n)
		grp := make([]scoredHand, j-i)
		copy(grp, hands[i:j])
		out = append(out, scoreBucket{score: hands[i].raw, p: p, rating: ratingFromP(p), hands: grp})
		i = j
	}
	if len(out) > 0 {
		out[0].p, out[0].rating = 0, 0
		out[len(out)-1].p, out[len(out)-1].rating = 1, ratingMax
	}
	return out
}

// scorePercentile geeft het percentiel van een willekeurige score binnen de
// populatie: op of boven de sterkste -> 1,0; op of onder de zwakste -> 0.
func scorePercentile(buckets []scoreBucket, n, score int) float64 {
	if len(buckets) == 0 {
		return 0
	}
	if score >= buckets[len(buckets)-1].score {
		return 1
	}
	if score <= buckets[0].score {
		return 0
	}
	below, equal := 0, 0
	for _, b := range buckets {
		switch {
		case b.score < score:
			below += len(b.hands)
		case b.score == score:
			equal = len(b.hands)
		}
	}
	return (float64(below) + 0.5*float64(equal)) / float64(n)
}

func formatHands(hands []scoredHand) string {
	ratingOf := map[int]int{}
	for _, b := range buildScoreBuckets(hands) {
		ratingOf[b.score] = b.rating
	}
	var sb strings.Builder
	sb.WriteString(handsHeader + "\n")
	for _, h := range hands {
		sb.WriteString(fmt.Sprintf("%s %d %d\n", handToTokens(h.cards), h.raw, ratingOf[h.raw]))
	}
	return sb.String()
}

func parseHands(data string) []scoredHand {
	var out []scoredHand
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		cards, err := ParseCards(fields[0])
		if err != nil || len(cards) != 18 {
			continue
		}
		// Score altijd herberekenen uit de kaarten: de scorefunctie is de enige
		// bron van waarheid, dus een oud bestand met verouderde getallen blijft werken.
		out = append(out, scoredHand{cards: cards, raw: handScore(cards)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].raw < out[j].raw })
	return out
}

func printHandRating(cards []Card, score int, p float64, rating int, hands []scoredHand) {
	jok, two, ace, high, sets := handComposition(cards)
	hh := NewHand(cards)
	hh.Sort()
	fmt.Println()
	fmt.Printf("Hand     : %s\n", hh.String())
	fmt.Printf("Tokens   : %s\n", handToTokens(cards))
	fmt.Printf("Score    : %d   (percentiel ~%.1f%% van alle mogelijke handen)\n", score, p*100)
	fmt.Printf("RATING   : %d\n", rating)
	fmt.Printf("Samenst. : %d joker, %d twee, %d aas, %d hoog (J+), %d set(s)\n", jok, two, ace, high, sets)
	fmt.Println()
	topRef := bestAnchorHand()
	fmt.Printf("Ref: %d handen in het spectrum, score %d (rating 0) .. %d\n",
		len(hands), hands[0].raw, hands[len(hands)-1].raw)
	fmt.Printf("Ref: theoretische topper %s -> score %d, rating %d\n",
		handToTokens(topRef), handScore(topRef), ratingMax)
}

// runPartijRating reconstrueert de starthanden uit een ingevoerde partij en
// toont per speler de rating van zijn starthand.
func runPartijRating(reader *Reader, buckets []scoreBucket, popN int) {
	PrintHeader("Partij -> ratings")
	numPlayers := 2
	if v, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && v >= 2 && v <= 4 {
		numPlayers = v
	}
	startPlayer := 0
	if v, err := reader.ReadInt(fmt.Sprintf("Wie begint (1-%d, standaard 1): ", numPlayers)); err == nil && v >= 1 && v <= numPlayers {
		startPlayer = v - 1
	}
	fmt.Println()
	fmt.Println("Voer alle zetten in als spatie-gescheiden tokens (pas = p / - / pass).")
	fmt.Println("Aas+vervolg als '1/5'. Zet de resterende kaarten van de verliezer als laatste token.")
	tokens := strings.Fields(reader.ReadLine("Zetten: "))
	if len(tokens) == 0 {
		fmt.Println("Geen zetten ingevoerd.")
		return
	}

	cardsPerPlayer := make([]int, numPlayers)
	for i := range cardsPerPlayer {
		cardsPerPlayer[i] = 18
	}
	hands, _, _, err := reconstructStartingHands(numPlayers, cardsPerPlayer, startPlayer, tokens)
	if err != nil {
		fmt.Printf("Fout bij reconstructie: %v\n", err)
		return
	}

	// Ontbrekende kaarten opvragen (meestal alleen de verliezer).
	for pi, h := range hands {
		if h.Count() >= cardsPerPlayer[pi] {
			continue
		}
		need := cardsPerPlayer[pi] - h.Count()
		fmt.Printf("\nSpeler %d mist nog %d kaart(en) die niet in de zetten stonden.\n", pi+1, need)
		for {
			parsed, e := reader.ReadCards(fmt.Sprintf("Resterende kaarten Speler %d (%d): ", pi+1, need))
			if e != nil {
				fmt.Printf("Fout: %v\n", e)
				continue
			}
			if len(parsed) != need {
				fmt.Printf("Verwacht %d, kreeg %d.\n", need, len(parsed))
				continue
			}
			hands[pi] = NewHand(append(hands[pi].Cards, parsed...))
			break
		}
	}

	PrintHeader("Resultaat")
	for pi, h := range hands {
		s := handScore(h.Cards)
		p := scorePercentile(buckets, popN, s)
		hh := NewHand(h.Cards)
		hh.Sort()
		note := ""
		if h.Count() != 18 {
			note = fmt.Sprintf("   [!] %d kaarten i.p.v. 18", h.Count())
		}
		fmt.Printf("Speler %d: rating %d   (score %d, percentiel ~%.1f%%)%s\n",
			pi+1, ratingFromP(p), s, p*100, note)
		fmt.Printf("          %s\n", hh.String())
	}
}

func dealFinderMode(reader *Reader, cfg settings) {
	_ = cfg
	PrintHeader("Kaartenset Rating")
	fmt.Println("Rating van een hand over het volledige spectrum van mogelijke handen.")
	fmt.Println("Lineair tot p0,2 (rating 1000), daarna verdubbelt de rating per 0,1 percentiel:")
	fmt.Println("  p0,2->1000  p0,3->2000  p0,4->4000  p0,6->16000  p0,8->64000  top(p1,0)->256000")
	fmt.Println()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var hands []scoredHand
	if raw, err := os.ReadFile(handsFile); err == nil {
		hands = parseHands(string(raw))
		if len(hands) > 0 {
			fmt.Printf("%s gevonden -- %d handen ingeladen.\n", handsFile, len(hands))
		}
	}
	if len(hands) < 100 {
		fmt.Printf("%s aanmaken (steekproef over alle mogelijke handen)...\n", handsFile)
		hands = generatePopulation(handsSampleN, rng)
		if err := os.WriteFile(handsFile, []byte(formatHands(hands)), 0o644); err != nil {
			fmt.Printf("Kon %s niet schrijven: %v\n", handsFile, err)
		} else {
			fmt.Printf("%s geschreven -- %d handen, score %d..%d.\n",
				handsFile, len(hands), hands[0].raw, hands[len(hands)-1].raw)
		}
	}
	if len(hands) < 2 {
		fmt.Println("Niet genoeg handen.")
		return
	}
	buckets := buildScoreBuckets(hands)
	n := len(hands)

	choice := strings.ToLower(reader.ReadLine("(h) hand -> rating, (r) rating -> hand, (p) partij -> ratings? [h/r/p]: "))

	if strings.HasPrefix(choice, "p") {
		runPartijRating(reader, buckets, n)
		return
	}

	if strings.HasPrefix(choice, "h") {
		for {
			line := reader.ReadLine("Voer 18 kaarten in (leeg = terug): ")
			if strings.TrimSpace(line) == "" {
				return
			}
			cards, err := ParseCards(line)
			if err != nil {
				fmt.Printf("Fout: %v\n", err)
				continue
			}
			if len(cards) != 18 {
				fmt.Printf("Verwacht 18 kaarten, kreeg %d.\n", len(cards))
				continue
			}
			s := handScore(cards)
			p := scorePercentile(buckets, n, s)
			PrintHeader("Resultaat")
			printHandRating(cards, s, p, ratingFromP(p), hands)
			return
		}
	}

	target := 1500
	if v, err := reader.ReadInt(fmt.Sprintf("Gewenste rating (0-%d, standaard 1500): ", ratingMax)); err == nil && v >= 0 {
		target = v
	}
	best := buckets[0]
	for _, b := range buckets {
		if math.Abs(float64(b.rating-target)) < math.Abs(float64(best.rating-target)) {
			best = b
		}
	}
	chosen := best.hands[rng.Intn(len(best.hands))]

	PrintHeader("Resultaat")
	fmt.Printf("Gevraagde rating : %d\n", target)
	printHandRating(chosen.cards, chosen.raw, best.p, best.rating, hands)
	if len(best.hands) > 1 {
		fmt.Printf("(%d handen delen deze score/rating -- willekeurig 1 gekozen)\n", len(best.hands))
	}
	fmt.Printf("Verwijder %s om een nieuwe steekproef te trekken.\n", handsFile)
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

var _ = SaveGame
var _ = LoadGame
var _ = EvaluateHand
var _ = QuickEvaluateMove
var _ = ShouldPass
var _ = gatherSpecials
