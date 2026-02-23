package game

import "github.com/azen-engine/pkg/cards"

// PassRecord slaat op wat een speler niet kon kloppen toen hij paste met < 9 kaarten.
type PassRecord struct {
	Count     int        // aantal kaarten vereist in die ronde
	TableRank cards.Rank // rank die geklopt moest worden
}

// KnowledgeTracker tracks what we know about opponents' hands.
type KnowledgeTracker struct {
	NumPlayers     int
	MyPlayerID     int
	MyHand         *cards.Hand
	CardsPlayed    []cards.Card
	DeadCards      []cards.Card
	HandCounts     []int
	PlayedByPlayer [][]cards.Card

	// PassRecords: als speler P past met < 9 kaarten, slaan we op wat hij niet kon kloppen.
	// Index = playerID. Enkel enkelvoudige zetten (Count=1) geven betrouwbare inferentie.
	PassRecords [][]PassRecord

	// Suspicions: handmatig ingevoerde positieve vermoedens ("ik denk dat speler X dit heeft").
	// Engine geeft deze voorrang bij determinisering.
	// Auto-bijgewerkt zodra kaarten gespeeld worden.
	Suspicions map[int][]cards.Card

	// Exclusions: handmatig ingevoerde negatieve vermoedens ("ik denk dat speler X dit NIET heeft").
	// Werkt als extra uitsluiting bovenop pas-inferentie.
	// Auto-bijgewerkt zodra kaarten gespeeld worden (want dan weet je wie het had).
	Exclusions map[int]map[cards.Rank]int // playerID → rank → aantal dat we denken dat ze NIET hebben
}

func NewKnowledgeTracker(numPlayers, myID int, myHand *cards.Hand, deadCards []cards.Card) *KnowledgeTracker {
	kt := &KnowledgeTracker{
		NumPlayers:     numPlayers,
		MyPlayerID:     myID,
		MyHand:         myHand.Clone(),
		DeadCards:      make([]cards.Card, len(deadCards)),
		HandCounts:     make([]int, numPlayers),
		PlayedByPlayer: make([][]cards.Card, numPlayers),
		PassRecords:    make([][]PassRecord, numPlayers),
		Suspicions:     map[int][]cards.Card{},
		Exclusions:     map[int]map[cards.Rank]int{},
	}
	copy(kt.DeadCards, deadCards)
	for i := range kt.HandCounts {
		kt.HandCounts[i] = 18
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
	// Update vermoedens: gespeelde kaarten zijn nu bekend
	kt.updateSuspicions(m.Cards)
}

// RecordPass slaat een pas-inferentie op. Roep dit aan VOOR ApplyMove.
// Als de passende speler < 9 kaarten heeft en de ronde enkelvoudig is,
// leiden we af dat hij geen kaart had om de tafel-rank te kloppen.
func (kt *KnowledgeTracker) RecordPass(passerID int, round RoundState) {
	if passerID == kt.MyPlayerID {
		return // eigen kaarten kennen we al
	}
	if round.IsOpen {
		return // open ronde: passen kan strategisch zijn
	}
	if kt.HandCounts[passerID] >= 9 { // >= helft van 18 startkaarten
		return // nog veel kaarten; kan specials bewaren
	}
	if round.Count != 1 {
		return // meerdere kaarten: te complex voor betrouwbare individuele inferentie
	}
	kt.PassRecords[passerID] = append(kt.PassRecords[passerID], PassRecord{
		Count:     round.Count,
		TableRank: round.TableRank,
	})
}

// AddSuspicion registreert kaarten die we denken dat een speler heeft.
// Enkel kaarten die nog plausibel in de pool zitten worden toegevoegd.
// Geeft terug hoeveel kaarten effectief toegevoegd zijn.
func (kt *KnowledgeTracker) AddSuspicion(playerID int, cc []cards.Card) int {
	if playerID == kt.MyPlayerID {
		return 0
	}
	// Tel de mogelijke pool
	pool := kt.PossibleOpponentCards()
	poolCount := map[cards.Rank]int{}
	for _, c := range pool {
		poolCount[c.Rank]++
	}
	// Trek reeds-vermoeide kaarten van andere spelers af
	for pid, susp := range kt.Suspicions {
		if pid == playerID {
			continue
		}
		for _, c := range susp {
			poolCount[c.Rank]--
		}
	}
	// Tel al bestaande vermoedens voor deze speler
	suspCount := map[cards.Rank]int{}
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

// ClearSuspicions verwijdert alle vermoedens voor een speler.
func (kt *KnowledgeTracker) ClearSuspicions(playerID int) {
	kt.Suspicions[playerID] = nil
}

// AddExclusion registreert ranks die we denken dat een speler NIET heeft.
// Bv. gok 2:-KK → we denken dat speler 2 geen 2 koningen heeft.
// Geeft terug hoeveel kaarten effectief toegevoegd zijn.
func (kt *KnowledgeTracker) AddExclusion(playerID int, cc []cards.Card) int {
	if playerID == kt.MyPlayerID {
		return 0
	}
	if kt.Exclusions[playerID] == nil {
		kt.Exclusions[playerID] = map[cards.Rank]int{}
	}
	added := 0
	for _, c := range cc {
		// Controleer of deze rank nog in de pool zit (anders is de exclusie zinloos)
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

// ClearExclusions verwijdert alle negatieve vermoedens voor een speler.
func (kt *KnowledgeTracker) ClearExclusions(playerID int) {
	kt.Exclusions[playerID] = nil
}

// ExcludedRanks geeft de ranks terug die de engine NIET aan playerID mag toewijzen
// tijdens determinisering. Combineert:
//  1. Pas-inferentie: enkelvoudige passen met < 9 kaarten
//  2. Handmatige negatieve vermoedens (gok 2:-KK)
func (kt *KnowledgeTracker) ExcludedRanks(playerID int) map[cards.Rank]bool {
	excluded := map[cards.Rank]bool{}

	// 1. Pas-inferentie
	for _, pr := range kt.PassRecords[playerID] {
		// Aas + wilds kunnen altijd een enkelvoudige zet kloppen
		excluded[cards.RankAce] = true
		excluded[cards.RankTwo] = true
		excluded[cards.RankJoker] = true
		// Normale ranks hoger dan de tafel-rank kloppen ook
		for _, r := range cards.NormalRanks() {
			if r > pr.TableRank {
				excluded[r] = true
			}
		}
	}

	// 2. Handmatige exclusies: elke rank waarvoor we vermoeden dat de speler ze niet heeft
	// We markeren als uitgesloten als de exclusie-count >= het aantal beschikbare kaarten van die rank
	pool := kt.PossibleOpponentCards()
	poolCount := map[cards.Rank]int{}
	for _, c := range pool {
		poolCount[c.Rank]++
	}
	for rank, exclCount := range kt.Exclusions[playerID] {
		if exclCount > 0 {
			// Voeg toe als "soft exclusion" (tier3 in determinize)
			excluded[rank] = true
		}
		_ = poolCount // toekomstige verfijning mogelijk
	}

	return excluded
}

// updateSuspicions verwijdert gespeelde kaarten automatisch uit de vermoedens en exclusies.
// Als een "uitgesloten" kaart toch gespeeld wordt door iemand anders, reduceer de exclusie.
func (kt *KnowledgeTracker) updateSuspicions(played []cards.Card) {
	playedCount := map[cards.Rank]int{}
	for _, c := range played {
		playedCount[c.Rank]++
	}

	// Positieve vermoedens bijwerken
	for pid, suspected := range kt.Suspicions {
		if len(suspected) == 0 {
			continue
		}
		removed := map[cards.Rank]int{}
		var newSusp []cards.Card
		for _, c := range suspected {
			if removed[c.Rank] < playedCount[c.Rank] {
				removed[c.Rank]++ // gespeeld → verwijder uit vermoeden
			} else {
				newSusp = append(newSusp, c)
			}
		}
		kt.Suspicions[pid] = newSusp
	}

	// Negatieve vermoedens bijwerken: als een rank gespeeld wordt,
	// reduceer de exclusie (er waren blijkbaar toch minder kaarten dan gedacht)
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

// PossibleOpponentCards geeft kaarten terug die tegenstanders mogelijk hebben.
// Telt per rank hoeveel er "bekend" zijn (eigen hand + gespeeld + dood)
// en geeft het resterende aantal terug als mogelijke tegenstanderkaarten.
func (kt *KnowledgeTracker) PossibleOpponentCards() []cards.Card {
	// Tel bekende kaarten per rank (suit doet er niet toe)
	knownCount := map[cards.Rank]int{}
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

	// Volledig deck: 4 exemplaren van elke normale rank + 2 jokers per deck
	normalRanks := []cards.Rank{
		cards.RankAce, cards.RankTwo, cards.RankThree, cards.RankFour,
		cards.RankFive, cards.RankSix, cards.RankSeven, cards.RankEight,
		cards.RankNine, cards.RankTen, cards.RankJack, cards.RankQueen, cards.RankKing,
	}
	totalCount := map[cards.Rank]int{}
	for _, r := range normalRanks {
		totalCount[r] = 4 * numDecks
	}
	totalCount[cards.RankJoker] = 2 * numDecks

	var possible []cards.Card
	allRanks := append(normalRanks, cards.RankJoker)
	for _, r := range allRanks {
		available := totalCount[r] - knownCount[r]
		for i := 0; i < available; i++ {
			possible = append(possible, cards.Card{Rank: r})
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
