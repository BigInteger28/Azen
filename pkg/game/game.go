package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/azen-engine/pkg/cards"
)

// Move represents a play (or pass) by a player
type Move struct {
	PlayerID int
	Cards    []cards.Card
	IsPass   bool
}

func PassMove(playerID int) Move {
	return Move{PlayerID: playerID, IsPass: true}
}

func (m Move) String() string {
	if m.IsPass {
		return fmt.Sprintf("P%d: PASS", m.PlayerID)
	}
	return fmt.Sprintf("P%d: %s", m.PlayerID, cards.CardsToString(m.Cards))
}

// ContainsAce checks if the move contains at least one ace
func (m Move) ContainsAce() bool {
	for _, c := range m.Cards {
		if c.IsAce() {
			return true
		}
	}
	return false
}

// EffectiveRank returns the highest normal rank in the played cards.
// Used to determine what the next player must beat.
// If only specials (wilds/aces), returns the current table rank (wilds inherit).
func (m Move) EffectiveRank(tableRank cards.Rank) cards.Rank {
	best := cards.Rank(0)
	for _, c := range m.Cards {
		if !c.IsSpecial() && c.Rank > best {
			best = c.Rank
		}
	}
	if best == 0 {
		// All specials: wilds take the value of whatever they're played on
		return tableRank
	}
	return best
}

// RoundState tracks the current trick/round context
type RoundState struct {
	Count        int        // Number of cards per play this round
	TableRank    cards.Rank // Current rank to beat
	IsOpen       bool       // New round, anything goes
	LastPlayerID int        // Last player who actually played (not passed)
	ConsecPasses int        // Consecutive passes
}

// GameState is the full game state
type GameState struct {
	NumPlayers  int
	Hands       []*cards.Hand
	CurrentTurn int // 0-based
	Round       RoundState
	Played      []cards.Card // All played cards (discard pile)
	History     []Move
	GameOver    bool
	Winner      int          // First player to empty hand (-1 if not over)
	Ranking     []int        // Players in finish order; len == NumPlayers when GameOver
	Finished    []bool       // Finished[i] true = player i emptied their hand
	DeadCards   []cards.Card // Out of play (2-player leftover)
}

// NewGame creates and deals a new game
func NewGame(numPlayers int, rng *rand.Rand, startPlayer int) *GameState {
	numDecks := 1
	if numPlayers == 4 {
		numDecks = 2
	}
	var deck *cards.Deck
	if numDecks == 1 {
		deck = cards.NewDeck()
	} else {
		deck = cards.NewMultiDeck(numDecks)
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

// NewGameWithHands creates a game from known hands (for analysis)
func NewGameWithHands(hands []*cards.Hand, dead []cards.Card, startPlayer int) *GameState {
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

// Clone deep-copies the game state
func (gs *GameState) Clone() *GameState {
	n := &GameState{
		NumPlayers:  gs.NumPlayers,
		CurrentTurn: gs.CurrentTurn,
		Round:       gs.Round,
		GameOver:    gs.GameOver,
		Winner:      gs.Winner,
	}
	n.Hands = make([]*cards.Hand, len(gs.Hands))
	for i, h := range gs.Hands {
		n.Hands[i] = h.Clone()
	}
	n.Played = make([]cards.Card, len(gs.Played))
	copy(n.Played, gs.Played)
	n.History = make([]Move, len(gs.History))
	copy(n.History, gs.History)
	n.DeadCards = make([]cards.Card, len(gs.DeadCards))
	copy(n.DeadCards, gs.DeadCards)
	n.Finished = make([]bool, len(gs.Finished))
	copy(n.Finished, gs.Finished)
	n.Ranking = make([]int, len(gs.Ranking))
	copy(n.Ranking, gs.Ranking)
	return n
}

// ─── Multi-place helpers ──────────────────────────────────────────────────────

// activePlayerCount returns the number of players who have not yet finished.
func (gs *GameState) activePlayerCount() int {
	count := 0
	for _, f := range gs.Finished {
		if !f {
			count++
		}
	}
	return count
}

// nextActiveTurn returns the first non-finished player after fromPID.
func (gs *GameState) nextActiveTurn(fromPID int) int {
	for i := 1; i <= gs.NumPlayers; i++ {
		next := (fromPID + i) % gs.NumPlayers
		if !gs.Finished[next] {
			return next
		}
	}
	return fromPID // only reached when game is over
}

// passThreshold returns how many consecutive passes trigger a new round.
// If the last player who played has since finished, all remaining active
// players must pass. Otherwise all except that last player must pass.
func (gs *GameState) passThreshold() int {
	active := gs.activePlayerCount()
	if gs.Finished[gs.Round.LastPlayerID] {
		return active // last player finished — all remaining must pass
	}
	return active - 1 // standard: all except last player
}

// PlayerRank returns the 0-based finishing position of pid (0 = 1st).
// Returns -1 if the player has not yet finished.
func (gs *GameState) PlayerRank(pid int) int {
	for i, p := range gs.Ranking {
		if p == pid {
			return i
		}
	}
	return -1
}

// finishPlayer marks pid as finished and updates Ranking/Winner.
// Returns true if the game is now over (only 1 active player left = loser).
func (gs *GameState) finishPlayer(pid int) bool {
	gs.Finished[pid] = true
	gs.Ranking = append(gs.Ranking, pid)
	if gs.Winner == -1 {
		gs.Winner = pid
	}
	if gs.activePlayerCount() <= 1 {
		// Add the loser (last remaining player) to Ranking
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

// ValidateMove checks if a move is legal in the current state
func (gs *GameState) ValidateMove(m Move) error {
	if gs.GameOver {
		return fmt.Errorf("game is over")
	}
	if m.PlayerID != gs.CurrentTurn {
		return fmt.Errorf("not player %d's turn (current: %d)", m.PlayerID, gs.CurrentTurn)
	}
	if m.IsPass {
		return nil // Always legal
	}
	if len(m.Cards) == 0 {
		return fmt.Errorf("must play at least one card (or pass)")
	}

	hand := gs.Hands[m.PlayerID]
	// Check all cards are in hand
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
	hasAce, hasNormal, normalRank, err := classifyCards(m.Cards)
	if err != nil {
		return err
	}
	if hasAce && hasNormal {
		return fmt.Errorf("een aas mag enkel samen met een 2 of joker gespeeld worden, niet met normale kaarten")
	}
	_ = normalRank
	return nil
}

func (gs *GameState) validateResponsePlay(m Move) error {
	hasAce, hasNormal, normalRank, err := classifyCards(m.Cards)
	if err != nil {
		return err
	}
	if hasAce && hasNormal {
		return fmt.Errorf("een aas mag enkel samen met een 2 of joker gespeeld worden, niet met normale kaarten")
	}

	// Alle zetten (ook aas) moeten de tel van de ronde matchen.
	// Assen bypassen alleen de rang-eis, niet de tel-eis.
	if len(m.Cards) != gs.Round.Count {
		return fmt.Errorf("moet exact %d kaart(en) spelen (gespeeld: %d)", gs.Round.Count, len(m.Cards))
	}

	// Normale kaarten moeten de tafel-rank verslaan
	if normalRank != 0 && normalRank <= gs.Round.TableRank {
		return fmt.Errorf("rank %d verslaat tafel-rank %d niet", normalRank, gs.Round.TableRank)
	}

	return nil
}

// classifyCards analyseert de samenstelling van een zet.
// Geeft hasAce, hasNormal, de normale rank (0 als geen) en een eventuele fout terug.
func classifyCards(cc []cards.Card) (hasAce bool, hasNormal bool, normalRank cards.Rank, err error) {
	for _, c := range cc {
		if c.IsAce() {
			hasAce = true
		} else if c.IsWild() {
			// wildcards zijn neutraal
		} else {
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

// ApplyMove applies a validated move. Call ValidateMove first!
func (gs *GameState) ApplyMove(m Move) {
	gs.History = append(gs.History, m)
	pid := m.PlayerID

	// ── Pass ─────────────────────────────────────────────────────────────────
	if m.IsPass {
		gs.Round.ConsecPasses++

		if gs.Round.ConsecPasses >= gs.passThreshold() {
			// All remaining active players have passed → new open round.
			// The player who last played cards starts the new round (if still active),
			// otherwise the next active player after them.
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

	// ── Play cards ───────────────────────────────────────────────────────────
	gs.Hands[pid].Remove(m.Cards)
	gs.Played = append(gs.Played, m.Cards...)

	// Did this player empty their hand?
	if gs.Hands[pid].IsEmpty() {
		if gs.finishPlayer(pid) {
			return // game over
		}
		// Game continues — determine who plays next and how the round resets.
		if m.ContainsAce() {
			// Ace resets to open round; finisher is done so next active player opens.
			gs.Round = RoundState{IsOpen: true, LastPlayerID: pid}
			gs.CurrentTurn = gs.nextActiveTurn(pid)
		} else {
			// Normal/wild play: update round, next active player must respond.
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

	// ── Player did not finish ─────────────────────────────────────────────────

	// Ace resets round: this player opens a new round immediately.
	if m.ContainsAce() {
		gs.Round = RoundState{IsOpen: true, LastPlayerID: pid}
		gs.CurrentTurn = pid
		return
	}

	// Normal/wild play: update round state.
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

// GetLegalMoves generates all legal moves for the current player
func (gs *GameState) GetLegalMoves() []Move {
	if gs.GameOver {
		return nil
	}

	pid := gs.CurrentTurn
	hand := gs.Hands[pid]
	moves := []Move{PassMove(pid)} // Can always pass

	if gs.Round.IsOpen {
		moves = append(moves, genOpenMoves(pid, hand)...)
	} else {
		moves = append(moves, genResponseMoves(pid, hand, gs.Round)...)
	}

	return moves
}

// genOpenMoves generates all valid opening plays
func genOpenMoves(pid int, hand *cards.Hand) []Move {
	var moves []Move

	byRank := map[cards.Rank][]cards.Card{}
	for _, c := range hand.Cards {
		byRank[c.Rank] = append(byRank[c.Rank], c)
	}

	wilds := gatherWilds(hand)
	aces := gatherAces(hand)

	// Normale kaarten, optioneel aangevuld met wildcards (GEEN assen)
	for _, rank := range cards.NormalRanks() {
		normals := byRank[rank]
		if len(normals) == 0 {
			continue
		}
		maxTotal := min(len(normals)+len(wilds), 6)

		for total := 1; total <= maxTotal; total++ {
			for numNorm := max(1, total-len(wilds)); numNorm <= min(len(normals), total); numNorm++ {
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
							merged := append(append([]cards.Card{}, nc...), wc...)
							moves = append(moves, Move{PlayerID: pid, Cards: merged})
						}
					}
				}
			}
		}
	}

	// Pure wildcard-zetten
	for total := 1; total <= min(len(wilds), 6); total++ {
		for _, wc := range combos(wilds, total) {
			moves = append(moves, Move{PlayerID: pid, Cards: wc})
		}
	}

	// Aas-zetten: minstens 1 aas, rest wildcards (GEEN normale kaarten)
	moves = append(moves, genAceMoves(pid, aces, wilds)...)

	return dedup(moves)
}

// genResponseMoves generates valid response plays
func genResponseMoves(pid int, hand *cards.Hand, round RoundState) []Move {
	var moves []Move
	need := round.Count
	tableRank := round.TableRank

	wilds := gatherWilds(hand)
	aces := gatherAces(hand)

	// Normale kaarten die de tafel verslaan, aangevuld met wildcards (GEEN assen)
	for _, rank := range cards.NormalRanks() {
		if rank <= tableRank {
			continue
		}
		normals := hand.GetByRank(rank)
		if len(normals) == 0 {
			continue
		}

		for numNorm := max(1, need-len(wilds)); numNorm <= min(len(normals), need); numNorm++ {
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
						merged := append(append([]cards.Card{}, nc...), wc...)
						moves = append(moves, Move{PlayerID: pid, Cards: merged})
					}
				}
			}
		}
	}

	// Pure wildcard-zetten (moeten de tel matchen)
	if len(wilds) >= need {
		for _, wc := range combos(wilds, need) {
			moves = append(moves, Move{PlayerID: pid, Cards: wc})
		}
	}

	// Aas-zetten: bypassen rang maar moeten WEL de tel matchen
	moves = append(moves, genAceResponseMoves(pid, aces, wilds, need)...)

	return dedup(moves)
}

// genAceMoves genereert alle aas-combinaties: minstens 1 aas, rest wildcards.
func genAceMoves(pid int, aces, wilds []cards.Card) []Move {
	var moves []Move
	for numAce := 1; numAce <= len(aces); numAce++ {
		maxW := min(len(wilds), 6-numAce)
		aCombos := combos(aces, numAce)
		for numWild := 0; numWild <= maxW; numWild++ {
			if numWild == 0 {
				for _, ac := range aCombos {
					moves = append(moves, Move{PlayerID: pid, Cards: ac})
				}
			} else {
				wCombos := combos(wilds, numWild)
				for _, ac := range aCombos {
					for _, wc := range wCombos {
						merged := append(append([]cards.Card{}, ac...), wc...)
						moves = append(moves, Move{PlayerID: pid, Cards: merged})
					}
				}
			}
		}
	}
	return moves
}

// genAceResponseMoves genereert aas-combinaties die precies 'need' kaarten bevatten.
// Gebruikt in responsmodus: assen bypassen rang maar moeten de tel matchen.
func genAceResponseMoves(pid int, aces, wilds []cards.Card, need int) []Move {
	var moves []Move
	for numAce := 1; numAce <= min(len(aces), need); numAce++ {
		numWild := need - numAce
		if numWild > len(wilds) {
			continue
		}
		aCombos := combos(aces, numAce)
		if numWild == 0 {
			for _, ac := range aCombos {
				moves = append(moves, Move{PlayerID: pid, Cards: ac})
			}
		} else {
			wCombos := combos(wilds, numWild)
			for _, ac := range aCombos {
				for _, wc := range wCombos {
					merged := append(append([]cards.Card{}, ac...), wc...)
					moves = append(moves, Move{PlayerID: pid, Cards: merged})
				}
			}
		}
	}
	return moves
}

func gatherSpecials(hand *cards.Hand) []cards.Card {
	var sp []cards.Card
	for _, c := range hand.Cards {
		if c.IsSpecial() {
			sp = append(sp, c)
		}
	}
	return sp
}

func gatherWilds(hand *cards.Hand) []cards.Card {
	var wilds []cards.Card
	for _, c := range hand.Cards {
		if c.IsWild() {
			wilds = append(wilds, c)
		}
	}
	return wilds
}

func gatherAces(hand *cards.Hand) []cards.Card {
	var aces []cards.Card
	for _, c := range hand.Cards {
		if c.IsAce() {
			aces = append(aces, c)
		}
	}
	return aces
}

// combos returns all k-element subsets of a slice
func combos(arr []cards.Card, k int) [][]cards.Card {
	if k <= 0 || k > len(arr) {
		if k == 0 {
			return [][]cards.Card{{}}
		}
		return nil
	}
	var result [][]cards.Card
	var helper func(start int, curr []cards.Card)
	helper = func(start int, curr []cards.Card) {
		if len(curr) == k {
			c := make([]cards.Card, k)
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

// dedup removes duplicate moves based on card identity
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
	sorted := make([]cards.Card, len(m.Cards))
	copy(sorted, m.Cards)
	// Sort for consistent key
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

// MovesEqual returns true if two moves play the same cards, regardless of order.
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

// StatusString returns a human-readable game status
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

func fmtRank(r cards.Rank) string {
	return (cards.Card{Rank: r}).RankStr()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
