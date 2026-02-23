package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/azen-engine/pkg/engine"
	"github.com/azen-engine/pkg/game"
)

// ─── Tuner-instellingen ───────────────────────────────────────────────────────

const (
	numPlayers   = 3    // aantal spelers per self-play partij
	gamesPerEval = 40   // partijen per richting (80 totaal per param, in parallel)
	itersPerMove = 200  // MCTS-iteraties per zet
	maxRounds    = 30   // maximale coordinate-descent rondes
	delta        = 0.04 // stapgrootte per parameter
	minImprove   = 0.02 // minimale winrate boven 0.50 om verbetering te accepteren
	maxMoves     = 600  // veiligheidsgrens per partij
)

// numWorkers past zich aan het systeem aan: jouw 8 cores → 8 parallelle partijen
var numWorkers = runtime.NumCPU()

// ─── main ────────────────────────────────────────────────────────────────────

func main() {
	weightsPath := "weights.json"
	if len(os.Args) > 1 {
		weightsPath = os.Args[1]
	}

	best, err := engine.LoadWeights(weightsPath)
	if err != nil {
		fmt.Println("Geen weights.json gevonden, start met standaard-weights.")
		best = engine.DefaultWeights()
		if saveErr := engine.SaveWeights(best, weightsPath); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Waarschuwing: kan %s niet aanmaken: %v\n", weightsPath, saveErr)
		}
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   AZEN Coordinate-Descent Tuner          ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("Spelers: %d  |  Games/eval: %d×2  |  Iters/zet: %d\n",
		numPlayers, gamesPerEval, itersPerMove)
	fmt.Printf("Delta: %.3f  |  Min verbetering: %.1f%%  |  Workers: %d\n\n",
		delta, minImprove*100, numWorkers)

	// Hoofdrng enkel voor seed-generatie (sequentieel, geen races)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	anyImproved := true
	round := 0
	totalStart := time.Now()

	for anyImproved && round < maxRounds {
		anyImproved = false
		round++
		roundStart := time.Now()
		fmt.Printf("─── Ronde %d ───\n", round)

		params := best.Params()
		for pi, p := range params {
			original := *p.Ptr

			// Maak kopieën met +delta en -delta
			plusW := best
			plusParams := plusW.Params()
			*plusParams[pi].Ptr = clamp(original+delta, p.Min, p.Max)

			minusW := best
			minusParams := minusW.Params()
			*minusParams[pi].Ptr = clamp(original-delta, p.Min, p.Max)

			// Evalueer beide richtingen in één parallelle batch
			plusRate, minusRate := evalBothDirections(plusW, minusW, best, rng)

			// Kies de beste richting
			bestRate := plusRate
			newVal := clamp(original+delta, p.Min, p.Max)
			dir := "+"
			if minusRate > plusRate {
				bestRate = minusRate
				newVal = clamp(original-delta, p.Min, p.Max)
				dir = "-"
			}

			if bestRate > 0.5+minImprove {
				if dir == "+" {
					best = plusW
				} else {
					best = minusW
				}
				fmt.Printf("  ✓ %-24s %s%.3f → %.3f   win=%.1f%%\n",
					p.Name, dir, original, newVal, bestRate*100)
				anyImproved = true

				if saveErr := engine.SaveWeights(best, weightsPath); saveErr != nil {
					fmt.Fprintf(os.Stderr, "  ⚠ Fout bij opslaan: %v\n", saveErr)
				}
			} else {
				fmt.Printf("  · %-24s    %.3f          +%.1f%%  -%.1f%%\n",
					p.Name, original, plusRate*100, minusRate*100)
			}
		}

		fmt.Printf("  Rondetijd: %s\n\n", time.Since(roundStart).Round(time.Second))
	}

	// Altijd opslaan aan het einde
	if saveErr := engine.SaveWeights(best, weightsPath); saveErr != nil {
		fmt.Fprintf(os.Stderr, "⚠ Fout bij eindopslag: %v\n", saveErr)
	}

	fmt.Printf("Totale tuningtijd: %s\n", time.Since(totalStart).Round(time.Second))
	if round >= maxRounds {
		fmt.Printf("Gestopt na %d rondes (maximum bereikt).\n", maxRounds)
	} else {
		fmt.Println("Geen verdere verbetering gevonden — converged.")
	}
	fmt.Printf("Weights opgeslagen in: %s\n\n", weightsPath)
	printWeights(best)
}

// ─── Parallelle evaluatie ─────────────────────────────────────────────────────

// evalBothDirections evalueert plusW en minusW tegelijk in één parallelle batch.
// Seeds worden sequentieel gegenereerd om data-races op rng te vermijden;
// elke goroutine krijgt zijn eigen lokale RNG.
// Retourneert (plusScore, minusScore) als gemiddelde positiescore (0.0-1.0).
func evalBothDirections(plusW, minusW, baseline engine.Weights, rng *rand.Rand) (float64, float64) {
	totalGames := gamesPerEval * 2

	// Genereer alle seeds sequentieel (thread-safe)
	seeds := make([]int64, totalGames)
	for i := range seeds {
		seeds[i] = rng.Int63()
	}

	type result struct {
		isPlus bool
		score  float64
	}
	results := make([]result, totalGames)

	sem := make(chan struct{}, numWorkers)
	var wg sync.WaitGroup

	for g := 0; g < totalGames; g++ {
		g := g // capture loop variable
		isPlus := g < gamesPerEval
		localIdx := g % gamesPerEval

		candidateW := minusW
		if isPlus {
			candidateW = plusW
		}

		wg.Add(1)
		sem <- struct{}{} // bezet een worker-slot
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // geef worker-slot vrij

			localRng := rand.New(rand.NewSource(seeds[g]))
			candidatePos := localIdx % numPlayers
			score := playOneGame(candidateW, baseline, localRng, candidatePos)
			results[g] = result{isPlus: isPlus, score: score}
		}()
	}

	wg.Wait()

	var plusTotal, minusTotal float64
	for _, r := range results {
		if r.isPlus {
			plusTotal += r.score
		} else {
			minusTotal += r.score
		}
	}
	return plusTotal / float64(gamesPerEval),
		minusTotal / float64(gamesPerEval)
}

// playOneGame simuleert één volledige partij.
// candidateW speelt als candidatePos; baseline speelt de andere posities.
// Retourneert de positiescore van de kandidaat: 1e=1.0, 2e=0.5, laatste=0.0.
func playOneGame(candidate, baseline engine.Weights, rng *rand.Rand, candidatePos int) float64 {
	candidateCfg := engine.Config{
		Iterations:   itersPerMove,
		MaxTime:      60 * time.Second,
		ExploreConst: 1.4,
		NumPlayers:   numPlayers,
		Weights:      candidate,
	}
	baselineCfg := engine.Config{
		Iterations:   itersPerMove,
		MaxTime:      60 * time.Second,
		ExploreConst: 1.4,
		NumPlayers:   numPlayers,
		Weights:      baseline,
	}

	gs := game.NewGame(numPlayers, rng, 0)

	engs := make([]*engine.Engine, numPlayers)
	for p := 0; p < numPlayers; p++ {
		if p == candidatePos {
			engs[p] = engine.NewEngine(candidateCfg)
		} else {
			engs[p] = engine.NewEngine(baselineCfg)
		}
	}

	kts := make([]*game.KnowledgeTracker, numPlayers)
	for p := 0; p < numPlayers; p++ {
		kts[p] = game.NewKnowledgeTracker(numPlayers, p, gs.Hands[p], gs.DeadCards)
	}

	moves := 0
	for !gs.GameOver && moves < maxMoves {
		pid := gs.CurrentTurn
		move, _ := engs[pid].BestMove(gs, kts[pid])
		// Pas-inferentie bijhouden vóór ApplyMove
		if move.IsPass {
			for p := 0; p < numPlayers; p++ {
				kts[p].RecordPass(move.PlayerID, gs.Round)
			}
		}
		for p := 0; p < numPlayers; p++ {
			kts[p].RecordMove(move)
		}
		gs.ApplyMove(move)
		moves++
	}

	if !gs.GameOver {
		return 0.5 // onbeslist (timeout) → neutraal
	}
	// Positiescore: 1e=1.0, 2e (3 spelers)=0.5, laatste=0.0
	rank := gs.PlayerRank(candidatePos)
	if rank < 0 {
		return 0.0 // verliezer
	}
	return float64(numPlayers-1-rank) / float64(numPlayers-1)
}

// ─── Hulpfuncties ─────────────────────────────────────────────────────────────

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func printWeights(w engine.Weights) {
	fmt.Println("Huidige weights:")
	for _, p := range w.Params() {
		fmt.Printf("  %-24s = %.4f\n", p.Name, *p.Ptr)
	}
	fmt.Println()
}
