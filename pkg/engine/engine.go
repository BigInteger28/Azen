package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/azen-engine/pkg/cards"
	"github.com/azen-engine/pkg/game"
)

type Config struct {
	Iterations     int
	MaxTime        time.Duration
	ExploreConst   float64
	NumPlayers     int
	Weights        Weights
	OmniscientMode bool // Alle handen zijn bekend → geen determinisering, gebruik werkelijke staat
	NumWorkers     int  // Parallelle ISMCTS-bomen (root-parallellisme). 0 of 1 = sequentieel.
}

// DefaultConfig maakt een standaard config. Laadt automatisch weights.json als dat bestaat.
func DefaultConfig(numPlayers int) Config {
	w, _ := LoadWeights("weights.json") // geen fout als bestand ontbreekt → defaults
	return Config{
		Iterations:   5000,
		MaxTime:      0,
		ExploreConst: 1.4,
		NumPlayers:   numPlayers,
		Weights:      w,
		NumWorkers:   2, // standaard 2 threads
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
	move     game.Move
	parent   *mctsNode
	children []*mctsNode
	visits   int
	wins     float64
	playerID int
}

func newRoot() *mctsNode { return &mctsNode{playerID: -1} }

// MoveEval contains the engine's evaluation of the best move
type MoveEval struct {
	Score   float64      // Win probability [0, 1]
	Visits  int
	Details []MoveDetail // All candidate moves ranked
}

func (me MoveEval) String() string {
	return fmt.Sprintf("Win%%: %.1f%% (%d visits)", me.Score*100, me.Visits)
}

type MoveDetail struct {
	Move    game.Move
	WinRate float64
	Visits  int
}

func (md MoveDetail) String() string {
	return fmt.Sprintf("  %s -> %.1f%% (%d visits)", md.Move, md.WinRate*100, md.Visits)
}

// findImmediateWin returns a move that empties the current player's hand (instant win), or nil.
func findImmediateWin(gs *game.GameState) *game.Move {
	pid := gs.CurrentTurn
	handCount := gs.Hands[pid].Count()
	for _, m := range gs.GetLegalMoves() {
		if !m.IsPass && len(m.Cards) == handCount {
			mv := m
			return &mv
		}
	}
	return nil
}

// workerResult bevat de gesommeerde statistieken van de wortelkinderen van één ISMCTS-boom.
type workerResult struct {
	visits map[string]int
	wins   map[string]float64
	moves  map[string]game.Move
}

// runWorker voert één onafhankelijke ISMCTS-boom uit en geeft de root-kind-statistieken terug.
// Elke worker heeft zijn eigen RNG en boom — geen gedeelde toestand, dus geen locks nodig.
func (e *Engine) runWorker(gs *game.GameState, kt *game.KnowledgeTracker, iters int, seed int64) workerResult {
	// Maak een worker-engine met eigen RNG (geen NumWorkers → geen recursie)
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
		node, simGS := worker.selectExpand(root, detGS, myID)
		result := worker.simulate(simGS, myID)
		worker.backprop(node, result, myID)
	}

	// Geef root-kind-statistieken terug
	res := workerResult{
		visits: map[string]int{},
		wins:   map[string]float64{},
		moves:  map[string]game.Move{},
	}
	for _, ch := range root.children {
		k := mkey(ch.move)
		res.visits[k] += ch.visits
		res.wins[k] += ch.wins
		res.moves[k] = ch.move
	}
	return res
}

// BestMove finds the best move using ISMCTS.
// Bij NumWorkers > 1 worden meerdere onafhankelijke bomen parallel gebouwd (root-parallellisme)
// en worden de resultaten samengevoegd op basis van bezoektal.
func (e *Engine) BestMove(gs *game.GameState, kt *game.KnowledgeTracker) (game.Move, MoveEval) {
	// Directe winst: als een zet de hand leegmaakt, altijd spelen (geen search nodig)
	if win := findImmediateWin(gs); win != nil {
		return *win, MoveEval{Score: 1.0, Visits: 1}
	}

	numWorkers := e.Config.NumWorkers
	if numWorkers <= 1 {
		return e.bestMoveSingle(gs, kt)
	}

	// Verdeel iteraties over workers (rest gaat naar de laatste worker)
	itersPerWorker := e.Config.Iterations / numWorkers
	if itersPerWorker < 1 {
		itersPerWorker = 1
	}

	// Genereer seeds sequentieel (thread-safe: enkel main-goroutine raakt rng aan)
	seeds := make([]int64, numWorkers)
	for i := range seeds {
		seeds[i] = e.rng.Int63()
	}

	// Start workers parallel
	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			iters := itersPerWorker
			// Laatste worker krijgt de resterende iteraties
			if idx == numWorkers-1 {
				iters = e.Config.Iterations - itersPerWorker*(numWorkers-1)
			}
			results[idx] = e.runWorker(gs, kt, iters, seeds[idx])
		}(w)
	}
	wg.Wait()

	// Samenvoegen: sommeer bezoeken en winsten per zet-sleutel
	totalVisits := map[string]int{}
	totalWins := map[string]float64{}
	moveMap := map[string]game.Move{}
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
		return game.PassMove(gs.CurrentTurn), MoveEval{}
	}

	// Kies zet met meest bezoeken (standaard MCTS-criterium)
	bestKey := ""
	bestVisits := -1
	for k, v := range totalVisits {
		if v > bestVisits {
			bestVisits = v
			bestKey = k
		}
	}

	bestMove := moveMap[bestKey]
	wr := 0.0
	if bestVisits > 0 {
		wr = totalWins[bestKey] / float64(bestVisits)
	}

	// Bouw details voor alle kandidaat-zetten
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

// bestMoveSingle is de originele sequentiële ISMCTS (1 boom, 1 goroutine).
func (e *Engine) bestMoveSingle(gs *game.GameState, kt *game.KnowledgeTracker) (game.Move, MoveEval) {
	root := newRoot()
	myID := gs.CurrentTurn
	hasDeadline := e.Config.MaxTime > 0
	deadline := time.Now().Add(e.Config.MaxTime)

	for iter := 0; iter < e.Config.Iterations; iter++ {
		if hasDeadline && time.Now().After(deadline) {
			break
		}

		// 1. Determinize: create a possible concrete game state
		detGS := e.determinize(gs, kt)
		if detGS == nil {
			continue
		}

		// 2. Select + Expand
		node, simGS := e.selectExpand(root, detGS, myID)

		// 3. Simulate (random playout)
		result := e.simulate(simGS, myID)

		// 4. Backpropagate
		e.backprop(node, result, myID)
	}

	return e.pickBest(root, myID)
}

func (e *Engine) determinize(gs *game.GameState, kt *game.KnowledgeTracker) *game.GameState {
	// In alwetende modus (analysemode) zijn alle handen bekend.
	// Geen randomisering nodig: gebruik de werkelijke toestand direct.
	if e.Config.OmniscientMode {
		return gs.Clone()
	}

	det := gs.Clone()
	possible := kt.PossibleOpponentCards()

	e.rng.Shuffle(len(possible), func(i, j int) {
		possible[i], possible[j] = possible[j], possible[i]
	})

	// Tier-systeem voor slimme kaart-toewijzing per tegenstander:
	//   tier1 = vermoeide kaarten (gok-commando) → eerste keuze
	//   tier2 = niet-uitgesloten kaarten (normale pool)
	//   tier3 = uitgesloten door pas-inferentie → enkel als noodoplossing
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

		// Hoeveel vermoeide kaarten per rank we willen voor speler p
		suspCount := map[cards.Rank]int{}
		for _, c := range kt.Suspicions[p] {
			suspCount[c.Rank]++
		}
		assignedSusp := map[cards.Rank]int{}

		var tier1, tier2, tier3 []int // indices in possible
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

		ordered := append(append(tier1, tier2...), tier3...)
		if len(ordered) < need {
			return nil
		}

		hand := make([]cards.Card, need)
		for i := 0; i < need; i++ {
			idx := ordered[i]
			hand[i] = possible[idx]
			used[idx] = true
		}
		det.Hands[p] = cards.NewHand(hand)
	}
	return det
}

func (e *Engine) selectExpand(node *mctsNode, gs *game.GameState, myID int) (*mctsNode, *game.GameState) {
	simGS := gs.Clone()

	for !simGS.GameOver {
		moves := simGS.GetLegalMoves()
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

func (e *Engine) unexploredMoves(node *mctsNode, moves []game.Move) []game.Move {
	explored := map[string]bool{}
	for _, ch := range node.children {
		explored[mkey(ch.move)] = true
	}
	var result []game.Move
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

func (e *Engine) simulate(gs *game.GameState, myID int) float64 {
	sim := gs.Clone()
	// Meer moves nodig: meerdere spelers moeten uitkomen voor het spel stopt.
	for i := 0; i < 400 && !sim.GameOver; i++ {
		moves := sim.GetLegalMoves()
		if len(moves) == 0 {
			break
		}
		m := e.smartRandom(moves, sim)
		sim.ApplyMove(m)
	}
	if sim.GameOver {
		return positionScore(sim, myID)
	}
	return e.evalPos(sim, myID)
}

// positionScore berekent de score op basis van eindpositie.
// 1e plaats = 1.0, laatste plaats = 0.0, tussenposities lineair.
// Als de speler nog niet gefinished is in een geëindigd spel = hij is de verliezer (0.0).
func positionScore(gs *game.GameState, myID int) float64 {
	numP := gs.NumPlayers
	if numP <= 1 {
		return 1.0
	}
	rank := gs.PlayerRank(myID)
	if rank < 0 {
		return 0.0 // niet in ranking = verliezer
	}
	// rank 0 = 1e: score 1.0; rank numP-1 = laatste: score 0.0
	return float64(numP-1-rank) / float64(numP-1)
}

func (e *Engine) smartRandom(moves []game.Move, gs *game.GameState) game.Move {
	wts := e.Config.Weights

	// Directe winst altijd pakken tijdens simulaties
	handCount := gs.Hands[gs.CurrentTurn].Count()
	for _, m := range moves {
		if !m.IsPass && len(m.Cards) == handCount {
			return m
		}
	}

	var plays []game.Move
	var pass game.Move
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
	// Adaptieve pasdrempel: meer specials in hand → vaker passen (bewaren voor later)
	// Zwakke hand (veel geïsoleerde lage kaarten) → minder passen (actief afspelen)
	curHand := gs.Hands[gs.CurrentTurn]
	curWilds := curHand.CountRank(cards.RankTwo) + curHand.CountRank(cards.RankJoker)
	curAces := curHand.CountRank(cards.RankAce)
	specialRatio := 0.0
	if handCount > 0 {
		specialRatio = float64(curWilds+curAces) / float64(handCount)
	}
	passChance := wts.PassBase + specialRatio*wts.PassSpecialFactor
	if e.rng.Float64() < passChance {
		return pass
	}

	// Weighted selection:
	// - Assen zijn het meest waardevol (geven initiatief terug): Pow(AcePlayFactor, n)
	// - Wilds zijn ook kostbaar: Pow(WildPlayFactor, n)
	// - Aas+wild samen krijgt extra SynergyPenalty
	// - Lagere normale ranks worden licht verkozen via RankPreference
	weights := make([]float64, len(plays))
	total := 0.0
	for i, m := range plays {
		w := 1.0
		wilds := 0
		aces := 0
		for _, c := range m.Cards {
			if c.IsWild() {
				wilds++
			} else if c.IsAce() {
				aces++
			}
		}
		// Assen zijn het meest waardevol: geven initiatief terug via reset
		w *= math.Pow(wts.AcePlayFactor, float64(aces))
		// Wilds zijn ook kostbaar maar iets minder dan assen
		w *= math.Pow(wts.WildPlayFactor, float64(wilds))
		// Synergy-penalty: aas+wild samen is extra kostbaar
		if aces > 0 && wilds > 0 {
			w *= wts.SynergyPenalty
		}
		// Lagere normale ranks licht verkozen
		for _, c := range m.Cards {
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

func (e *Engine) evalPos(gs *game.GameState, myID int) float64 {
	// Speler al gefinished: positie ligt vast, geef definitieve score terug.
	if gs.Finished[myID] {
		return positionScore(gs, myID)
	}

	myCount := gs.Hands[myID].Count()
	if myCount == 0 {
		return 1.0 // fallback (zou niet voor moeten komen na Finished-check)
	}

	wts := e.Config.Weights

	// Vergelijk enkel met nog actieve tegenstanders (niet al gefinished)
	minOpp := 999
	for i, h := range gs.Hands {
		if i != myID && !gs.Finished[i] && h.Count() < minOpp {
			minOpp = h.Count()
		}
	}
	if minOpp == 999 {
		minOpp = 0 // alle tegenstanders al klaar: wij zijn de enige over = verliezer
	}

	score := 0.5 + float64(minOpp-myCount)*wts.CardDiffWeight

	hand := gs.Hands[myID]
	wilds := hand.CountRank(cards.RankTwo) + hand.CountRank(cards.RankJoker)
	aces := hand.CountRank(cards.RankAce)

	score += float64(aces) * wts.AceBonus
	score += float64(wilds) * wts.WildBonus

	if aces > 0 && wilds > 0 {
		score += float64(min(aces, wilds)) * wts.SynergyBonus
	}

	kings := hand.CountRank(cards.RankKing)
	if kings > 0 && wilds == 0 && aces == 0 {
		score -= float64(kings) * wts.KingPenalty
	}

	queens := hand.CountRank(cards.RankQueen)
	if queens > 0 && wilds == 0 && aces == 0 {
		score -= float64(queens) * wts.QueenPenalty
	}

	for r := cards.RankThree; r <= cards.RankSeven; r++ {
		if hand.CountRank(r) == 1 && wilds == 0 {
			score -= wts.IsolatedLowPenalty
		}
	}

	for r := cards.RankThree; r <= cards.RankKing; r++ {
		cnt := hand.CountRank(r)
		if cnt >= 2 {
			score += float64(cnt-1) * wts.ClusterBonus
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
		if node.playerID == myID {
			node.wins += result
		} else if node.playerID >= 0 {
			node.wins += 1.0 - result
		}
		node = node.parent
	}
}

func (e *Engine) pickBest(root *mctsNode, myID int) (game.Move, MoveEval) {
	if len(root.children) == 0 {
		return game.PassMove(myID), MoveEval{}
	}

	var bestNode *mctsNode
	bestV := -1
	for _, ch := range root.children {
		if ch.visits > bestV {
			bestV = ch.visits
			bestNode = ch
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
	// Sort by visits desc
	for i := 0; i < len(details); i++ {
		for j := i + 1; j < len(details); j++ {
			if details[j].Visits > details[i].Visits {
				details[i], details[j] = details[j], details[i]
			}
		}
	}

	return bestNode.move, MoveEval{Score: wr, Visits: bestV, Details: details}
}

// AnalyzeMove evaluates a specific move for post-game analysis
func (e *Engine) AnalyzeMove(gs *game.GameState, kt *game.KnowledgeTracker, m game.Move) MoveDetail {
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

func mkey(m game.Move) string {
	if m.IsPass {
		return "PASS"
	}
	sorted := make([]cards.Card, len(m.Cards))
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
