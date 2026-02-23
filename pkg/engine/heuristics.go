package engine

import (
	"github.com/azen-engine/pkg/cards"
	"github.com/azen-engine/pkg/game"
)

// HandStrength evaluates the overall strength of a hand
type HandStrength struct {
	CardCount      int
	WildCount      int     // 2s and Jokers
	AceCount       int
	HighCardCount  int     // J, Q, K
	LonelyKings    int     // Kings without wild support
	PairCount      int     // Number of ranks with 2+ cards
	TripleCount    int     // Number of ranks with 3+ cards
	TempoScore     float64 // Ability to control rounds
	OverallScore   float64
}

// EvaluateHand performs a detailed hand evaluation
func EvaluateHand(hand *cards.Hand) HandStrength {
	hs := HandStrength{
		CardCount: hand.Count(),
	}

	// Count specials
	hs.WildCount = hand.CountWilds()
	hs.AceCount = hand.CountAces()

	// Count rank groups
	rankCounts := make(map[cards.Rank]int)
	for _, c := range hand.Cards {
		if !c.IsSpecial() {
			rankCounts[c.Rank]++
		}
	}

	for rank, count := range rankCounts {
		if rank >= cards.RankJack {
			hs.HighCardCount += count
		}
		if count >= 2 {
			hs.PairCount++
		}
		if count >= 3 {
			hs.TripleCount++
		}
		if rank == cards.RankKing {
			hs.LonelyKings = count // Kings that need wilds to be played on
		}
	}

	// Tempo score: aces give us round control
	hs.TempoScore = float64(hs.AceCount) * 2.0

	// Overall score (lower is better - closer to winning)
	// Base: card count (less is better)
	hs.OverallScore = 100.0 - float64(hs.CardCount)*5.0

	// Bonus for specials
	hs.OverallScore += float64(hs.WildCount) * 8.0
	hs.OverallScore += float64(hs.AceCount) * 10.0

	// Bonus for pairs/triples (easier to play out)
	hs.OverallScore += float64(hs.PairCount) * 3.0
	hs.OverallScore += float64(hs.TripleCount) * 5.0

	// Penalty for lonely kings (hard to get rid of without wilds)
	kingPenalty := hs.LonelyKings - hs.WildCount
	if kingPenalty > 0 {
		hs.OverallScore -= float64(kingPenalty) * 6.0
	}

	// Penalty for low cards without tempo
	lowCards := 0
	for _, rank := range []cards.Rank{cards.RankThree, cards.RankFour, cards.RankFive} {
		lowCards += rankCounts[rank]
	}
	// Low cards are only good if you have tempo (aces)
	if hs.AceCount == 0 && lowCards > 0 {
		hs.OverallScore -= float64(lowCards) * 2.0
	}

	return hs
}

// MoveQuality evaluates a specific move in context
type MoveQuality struct {
	Move            game.Move
	Score           float64
	Reasoning       string
	WastesWilds     bool
	WastesAces      bool
	CreatesWinThreat bool
}

// QuickEvaluateMove provides a fast heuristic evaluation of a move
// This is used for move ordering (not the full ISMCTS evaluation)
func QuickEvaluateMove(gs *game.GameState, move game.Move) MoveQuality {
	mq := MoveQuality{Move: move}
	hand := gs.Hands[move.PlayerID]

	if move.IsPass {
		mq.Score = 0.0
		mq.Reasoning = "Pass"
		return mq
	}

	cardsAfter := hand.Count() - len(move.Cards)

	// Winning move!
	if cardsAfter == 0 {
		mq.Score = 100.0
		mq.CreatesWinThreat = true
		mq.Reasoning = "Winning move!"
		return mq
	}

	// Start with neutral score
	mq.Score = 50.0

	// Count specials used in this move
	wildsUsed := 0
	acesUsed := 0
	for _, c := range move.Cards {
		if c.IsWild() {
			wildsUsed++
		}
		if c.IsAce() {
			acesUsed++
		}
	}

	// Penalty for wasting wilds on low plays
	effectiveRank := move.EffectiveRank(gs.Round.TableRank)
	if wildsUsed > 0 && effectiveRank < cards.RankTen {
		mq.Score -= float64(wildsUsed) * 5.0
		mq.WastesWilds = true
		mq.Reasoning = "Wastes wildcards on low play"
	}

	// Aces are great for tempo control
	if acesUsed > 0 {
		mq.Score += 5.0 // Tempo advantage
		mq.WastesAces = acesUsed > 1 // Using multiple aces at once is wasteful
		if mq.WastesAces {
			mq.Score -= float64(acesUsed-1) * 8.0
			mq.Reasoning = "Uses multiple aces unnecessarily"
		}
	}

	// Prefer playing low cards first (save high cards for later)
	if effectiveRank > 0 {
		rankValue := float64(effectiveRank-cards.RankThree) / float64(cards.RankKing-cards.RankThree)
		mq.Score -= rankValue * 10.0 // Lower cards get higher score
	}

	// Prefer moves that reduce hand size significantly
	mq.Score += float64(len(move.Cards)) * 2.0

	// Near-win bonus
	if cardsAfter <= 3 {
		mq.Score += 15.0
		mq.CreatesWinThreat = true
	}

	// Opening advantage: prefer using aces to control tempo
	if gs.Round.IsOpen && acesUsed > 0 {
		mq.Score += 10.0
	}

	return mq
}

// ShouldPass heuristically determines if passing might be better
func ShouldPass(gs *game.GameState, playerID int) bool {
	hand := gs.Hands[playerID]

	// Never pass if we have very few cards
	if hand.Count() <= 3 {
		return false
	}

	// If the round is open (we control), never pass
	if gs.Round.IsOpen {
		return false
	}

	// If we'd have to waste precious wilds/aces on a low play, consider passing
	if gs.Round.TableRank <= cards.RankSix {
		// Low rank on table - probably worth playing if we can
		return false
	}

	// If table has K and we only have wilds to beat it, maybe pass
	if gs.Round.TableRank >= cards.RankKing {
		normalCardsAbove := 0
		for _, c := range hand.Cards {
			if !c.IsSpecial() && c.Rank > gs.Round.TableRank {
				normalCardsAbove++
			}
		}
		if normalCardsAbove == 0 {
			return true // Would need to waste specials
		}
	}

	return false
}
