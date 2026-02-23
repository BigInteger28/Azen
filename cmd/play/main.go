package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/azen-engine/pkg/cards"
	"github.com/azen-engine/pkg/engine"
	"github.com/azen-engine/pkg/game"
	azenio "github.com/azen-engine/pkg/io"
)

// settings bevat de globale engine-instellingen die de gebruiker kan aanpassen via optie 0.
type settings struct {
	numThreads int // aantal parallelle ISMCTS-bomen (root-parallellisme)
}

func main() {
	reader := azenio.NewReader()

	cfg := settings{numThreads: 2} // standaard 2 threads

	for {
		azenio.PrintHeader("AZEN Engine v1.0")
		fmt.Println("Welkom bij de AZEN kaartspel engine!")
		fmt.Println()
		fmt.Printf("  [0] Instellingen  (threads: %d)\n", cfg.numThreads)
		fmt.Println("  [1] Spelen  - Engine suggereert zetten voor jou")
		fmt.Println("  [2] Analyse - Bekijk een gespeeld spel opnieuw")
		fmt.Println("  [3] Simuleer - Kijk hoe de engine tegen zichzelf speelt")
		fmt.Println()

		modeStr := reader.ReadLine("Kies modus (0/1/2/3): ")
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
		default:
			playMode(reader, cfg)
			return
		}
	}
}

// settingsMenu laat de gebruiker de engine-instellingen aanpassen.
func settingsMenu(reader *azenio.Reader, cfg settings) settings {
	azenio.PrintHeader("Instellingen")
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

// handleGok verwerkt het 'gok'-commando voor handmatige vermoedens.
// Formaat:  gok 2:KK   → voeg K,K toe als vermoeden voor speler 2
//           gok 2:clear → wis alle vermoedens voor speler 2
//           gok         → toon alle huidige vermoedens
// Geeft (true, bericht) terug als het input een gok-commando was.
func handleGok(input string, tracker *game.KnowledgeTracker, myPlayer int, numPlayers int) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(input))
	if !strings.HasPrefix(lower, "gok") {
		return false, ""
	}

	rest := strings.TrimSpace(input[3:]) // alles na "gok"

	// "gok" zonder argument: toon alle vermoedens
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
				sb.WriteString(fmt.Sprintf("  Speler %d heeft:      %s\n", p+1, cards.CardsToString(susp)))
				any = true
			}
			if len(excl) > 0 {
				var parts []string
				for r, cnt := range excl {
					for i := 0; i < cnt; i++ {
						parts = append(parts, (cards.Card{Rank: r}).RankStr())
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

	// "gok N:kaarten" of "gok N:clear"
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

	// Negatief vermoeden: "-KK" = ik denk dat speler X geen 2 koningen heeft
	isNegative := strings.HasPrefix(arg, "-")
	if isNegative {
		arg = arg[1:] // verwijder het minteken
	}

	// Parseer kaarten
	parsed, err := cards.ParseCards(arg)
	if err != nil {
		return true, fmt.Sprintf("⚠️  Kaarten niet herkend: %v", err)
	}

	if isNegative {
		added := tracker.AddExclusion(targetID, parsed)
		exclMap := tracker.Exclusions[targetID]
		var exclParts []string
		for r, cnt := range exclMap {
			exclParts = append(exclParts, fmt.Sprintf("%dx%s", cnt, (cards.Card{Rank: r}).RankStr()))
		}
		msg := fmt.Sprintf("🚫 Speler %d heeft NIET: %s  (%d toegevoegd)",
			playerNum, cards.CardsToString(parsed), added)
		_ = exclParts
		return true, msg
	}

	// Positief vermoeden
	added := tracker.AddSuspicion(targetID, parsed)
	susp := tracker.Suspicions[targetID]
	msg := fmt.Sprintf("🔍 Gok Speler %d heeft: %s  (%d kaart(en) toegevoegd, totaal vermoeden: %s)",
		playerNum, cards.CardsToString(parsed), added, cards.CardsToString(susp))
	if added < len(parsed) {
		msg += fmt.Sprintf("\n   ⚠️  %d kaart(en) niet toegevoegd: al gespeeld of niet meer in pool", len(parsed)-added)
	}
	return true, msg
}

// printGameStatus toont de spelstatus met vermoedens voor tegenstanders.
// Vervangt gs.StatusString() in speelmodus zodat gok-info zichtbaar is.
func printGameStatus(gs *game.GameState, tracker *game.KnowledgeTracker, myPlayer int) {
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
		rankStr := (cards.Card{Rank: gs.Round.TableRank}).RankStr()
		fmt.Printf("Ronde: %dx kaarten, rank %s verslaan\n", gs.Round.Count, rankStr)
	}
	if gs.GameOver && len(gs.Ranking) > 0 {
		fmt.Printf("🏆 Speler %d WINT!\n", gs.Ranking[0]+1)
	}
	fmt.Println()
}

func playMode(reader *azenio.Reader, cfg settings) {
	azenio.PrintHeader("Speel Modus")

	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}

	myPlayer := 0
	if p, err := reader.ReadInt("Jouw spelernummer (1-" + strconv.Itoa(numPlayers) + "): "); err == nil && p >= 1 && p <= numPlayers {
		myPlayer = p - 1
	}

	fmt.Println("\nVoer jouw 18 kaarten in (komma, spatie of aaneengesloten):")
	fmt.Println("  Voorbeeld: KK3XJ19Q25  of  K,K,3,X,J  of  K K 3 X J")
	fmt.Println("  Typ 'help' voor uitleg.")
	fmt.Println()

	var myHand *cards.Hand
	for {
		input := reader.ReadLine("Jouw kaarten: ")
		if strings.ToLower(input) == "help" {
			azenio.PrintHelp()
			continue
		}
		parsed, err := cards.ParseCards(input)
		if err != nil {
			fmt.Printf("Fout: %v\n", err)
			continue
		}
		if len(parsed) != 18 {
			fmt.Printf("Verwacht 18 kaarten, kreeg %d. Probeer opnieuw.\n", len(parsed))
			continue
		}
		myHand = cards.NewHand(parsed)
		break
	}

	fmt.Println("\nJouw hand:")
	azenio.PrintCards(myHand)

	var deadCards []cards.Card
	if numPlayers == 2 {
		fmt.Println("\nMet 2 spelers zijn 18 kaarten niet in spel (engine houdt hiermee rekening).")
	}

	tracker := game.NewKnowledgeTracker(numPlayers, myPlayer, myHand, deadCards)

	// Opponenten als placeholder-handen (rank=0); engine gebruikt determinisatie
	hands := make([]*cards.Hand, numPlayers)
	for i := 0; i < numPlayers; i++ {
		if i == myPlayer {
			hands[i] = myHand
		} else {
			ph := make([]cards.Card, 18) // rank=0 placeholders
			hands[i] = cards.NewHand(ph)
		}
	}

	gs := game.NewGameWithHands(hands, deadCards, 0)

	iters := 5000
	if n, err := reader.ReadInt("Engine-iteraties per zet (standaard 5000, meer = nauwkeuriger maar trager): "); err == nil && n > 0 {
		iters = n
	}
	engConfig := engine.DefaultConfig(numPlayers)
	engConfig.Iterations = iters
	engConfig.MaxTime = 0 // geen tijdslimiet
	engConfig.NumWorkers = cfg.numThreads
	eng := engine.NewEngine(engConfig)

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
			azenio.PrintSubHeader("Jouw beurt")
			azenio.PrintCards(gs.Hands[myPlayer])

			fmt.Println("\n🤔 Engine denkt na...")
			bestMove, eval := eng.BestMove(gs, tracker)
			fmt.Printf("\n💡 Engine suggereert: %s (winst: %s)\n\n",
				azenio.FormatMove(bestMove), azenio.FormatScore(eval.Score))

			for {
				input := reader.ReadLine("Jouw zet (of 'hint'/'rethink'/'help'/'hand'/'status'/'moves'/'gok'): ")
				lower := strings.ToLower(input)

				switch lower {
				case "help":
					azenio.PrintHelp()
					continue
				case "hand":
					azenio.PrintCards(gs.Hands[myPlayer])
					continue
				case "status":
					printGameStatus(gs, tracker, myPlayer)
					continue
				case "hint":
					fmt.Printf("💡 Suggestie: %s (winst: %s)\n",
						azenio.FormatMove(bestMove), azenio.FormatScore(eval.Score))
					continue
				case "rethink":
					fmt.Println("\n🤔 Engine herdenkt de situatie...")
					bestMove, eval = eng.BestMove(gs, tracker)
					fmt.Printf("\n💡 Nieuwe suggestie: %s (winst: %s)\n\n",
						azenio.FormatMove(bestMove), azenio.FormatScore(eval.Score))
					continue
				case "moves":
					azenio.PrintMoveOptions(gs.GetLegalMoves(), 20)
					continue
				case "quit", "exit":
					fmt.Println("Tot ziens!")
					os.Exit(0)
				}

				// Gok-commando: bv. "gok 2:KK"
				if handled, msg := handleGok(input, tracker, myPlayer, numPlayers); handled {
					fmt.Println(msg)
					continue
				}

				// Splits op '/' voor aas+vervolg notatie (bv. "11/444")
				mainInput, followInput, hasFollow := strings.Cut(input, "/")
				mainInput = strings.TrimSpace(mainInput)
				mainLower := strings.ToLower(mainInput)

				var move game.Move
				if mainLower == "pass" || mainLower == "p" || mainLower == "-" {
					move = game.PassMove(myPlayer)
				} else {
					parsed, err := cards.ParseCards(mainInput)
					if err != nil {
						fmt.Printf("Fout: %v\n", err)
						continue
					}
					move = game.Move{PlayerID: myPlayer, Cards: parsed}
				}

				if err := gs.ValidateMove(move); err != nil {
					fmt.Printf("Ongeldige zet: %v\n", err)
					continue
				}
				// Pas-inferentie bijhouden vóór ApplyMove
				if move.IsPass {
					tracker.RecordPass(move.PlayerID, gs.Round)
				}
				gs.ApplyMove(move)
				tracker.RecordMove(move)

				// Vervolg-zet na aas-reset (bv. het "444" deel van "11/444")
				if hasFollow && !gs.GameOver && gs.CurrentTurn == myPlayer {
					followInput = strings.TrimSpace(followInput)
					parsed, err := cards.ParseCards(followInput)
					if err != nil {
						fmt.Printf("✅ Gespeeld: %s\n⚠️  Fout in vervolg-zet: %v\n\n", azenio.FormatMove(move), err)
						break
					}
					followMove := game.Move{PlayerID: myPlayer, Cards: parsed}
					if err := gs.ValidateMove(followMove); err != nil {
						fmt.Printf("✅ Gespeeld: %s\n⚠️  Ongeldige vervolg-zet: %v\n\n", azenio.FormatMove(move), err)
						break
					}
					gs.ApplyMove(followMove)
					tracker.RecordMove(followMove)
					fmt.Printf("✅ Gespeeld: %s / %s\n\n", azenio.FormatMove(move), azenio.FormatMove(followMove))
				} else {
					fmt.Printf("✅ Gespeeld: %s\n\n", azenio.FormatMove(move))
				}
				break
			}
		} else {
			playerNum := gs.CurrentTurn + 1
			oppID := gs.CurrentTurn
			azenio.PrintSubHeader(fmt.Sprintf("Beurt van Speler %d", playerNum))

			for {
				input := reader.ReadLine(fmt.Sprintf("Zet van Speler %d (of '-' voor pas, 'gok' voor vermoeden): ", playerNum))
				lower := strings.ToLower(strings.TrimSpace(input))

				if lower == "help" {
					azenio.PrintHelp()
					continue
				}
				if lower == "quit" || lower == "exit" {
					fmt.Println("Tot ziens!")
					os.Exit(0)
				}

				// Gok-commando ook beschikbaar bij tegenstanders
				if handled, msg := handleGok(input, tracker, myPlayer, numPlayers); handled {
					fmt.Println(msg)
					continue
				}

				// Splits op '/' voor aas+vervolg notatie
				mainInput, followInput, hasFollow := strings.Cut(input, "/")
				mainInput = strings.TrimSpace(mainInput)
				mainLower := strings.ToLower(mainInput)

				var move game.Move
				if mainLower == "pass" || mainLower == "p" || mainLower == "-" {
					move = game.PassMove(oppID)
				} else {
					parsed, err := cards.ParseCards(mainInput)
					if err != nil {
						fmt.Printf("Fout: %v\n", err)
						continue
					}
					move = game.Move{PlayerID: oppID, Cards: parsed}
				}

				// Pas-inferentie bijhouden vóór ApplyMove
				if move.IsPass {
					tracker.RecordPass(move.PlayerID, gs.Round)
				}
				gs.ApplyMove(move)
				tracker.RecordMove(move)

				// Vervolg-zet na aas-reset
				if hasFollow && !gs.GameOver && gs.CurrentTurn == oppID {
					followInput = strings.TrimSpace(followInput)
					if parsed, err := cards.ParseCards(followInput); err == nil {
						followMove := game.Move{PlayerID: oppID, Cards: parsed}
						gs.ApplyMove(followMove)
						tracker.RecordMove(followMove)
						fmt.Printf("📝 Speler %d speelde: %s / %s\n\n", playerNum, azenio.FormatMove(move), azenio.FormatMove(followMove))
						break
					}
				}
				fmt.Printf("📝 Speler %d speelde: %s\n\n", playerNum, azenio.FormatMove(move))
				break
			}
		}
	}

	azenio.PrintHeader("Spel Voorbij!")
	printRanking(gs)
}

func analyzeMode(reader *azenio.Reader, cfg settings) {
	azenio.PrintHeader("Analyse Modus")
	fmt.Println("Voer het volledige spel in voor analyse.")
	fmt.Println()

	numPlayers := 2
	if n, err := reader.ReadInt("Aantal spelers (2/3/4): "); err == nil && n >= 2 && n <= 4 {
		numPlayers = n
	}

	hands := make([]*cards.Hand, numPlayers)
	for i := 0; i < numPlayers; i++ {
		fmt.Printf("\nVoer de starthand van Speler %d in (18 kaarten):\n", i+1)
		for {
			parsed, err := reader.ReadCards(fmt.Sprintf("Speler %d kaarten: ", i+1))
			if err != nil {
				fmt.Printf("Fout: %v\n", err)
				continue
			}
			if len(parsed) != 18 {
				fmt.Printf("Verwacht 18, kreeg %d\n", len(parsed))
				continue
			}
			hands[i] = cards.NewHand(parsed)
			break
		}
	}

	var deadCards []cards.Card
	if numPlayers == 2 {
		if reader.ReadYesNo("Dode kaarten invoeren?") {
			for {
				parsed, err := reader.ReadCards("Dode kaarten: ")
				if err != nil {
					fmt.Printf("Fout: %v\n", err)
					continue
				}
				if len(parsed) != 18 {
					fmt.Printf("Verwacht 18, kreeg %d\n", len(parsed))
					continue
				}
				deadCards = parsed
				break
			}
		}
	}

	gs := game.NewGameWithHands(hands, deadCards, 0)

	engConfig := engine.DefaultConfig(numPlayers)
	// In analysemode zijn alle handen bekend → alwetende modus voor exacte analyse
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
			analyzeAll = true // ongeldige invoer → analyseer iedereen
		}
	}

	// Persistente trackers: één per geanalyseerde speler, bijgehouden doorheen het hele spel.
	trackers := make([]*game.KnowledgeTracker, numPlayers)
	for p := 0; p < numPlayers; p++ {
		if analyzeAll || analyzePlayers[p] {
			trackers[p] = game.NewKnowledgeTracker(numPlayers, p, gs.Hands[p], gs.DeadCards)
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

		// Splits op ':' voor optioneel spelernummer
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

		// Splits op '/' voor aas+vervolg notatie (bv. "11/444")
		mainCardsStr, followCardsStr, hasFollowCards := strings.Cut(cardsStr, "/")
		mainCardsStr = strings.TrimSpace(mainCardsStr)
		mainCardsLower := strings.ToLower(mainCardsStr)

		var move game.Move
		if mainCardsLower == "pass" || mainCardsLower == "p" || mainCardsStr == "-" {
			move = game.Move{PlayerID: playerID, IsPass: true}
		} else {
			parsed, err := cards.ParseCards(mainCardsStr)
			if err != nil {
				fmt.Printf("Fout: %v\n", err)
				moveNum--
				continue
			}
			move = game.Move{PlayerID: playerID, Cards: parsed}
		}

		// Analyseer VOOR het toepassen van de zet (alleen voor geselecteerde spelers)
		doAnalysis := analyzeAll || analyzePlayers[playerID]
		var bestMove game.Move
		var bestEval engine.MoveEval
		var actualDetail engine.MoveDetail
		var bestLabel string

		if doAnalysis {
			tracker := trackers[playerID] // persistente tracker (bevat volledige spelgeschiedenis)
			eng := engine.NewEngine(engConfig)
			bestMove, bestEval = eng.BestMove(gs, tracker)
			bestLabel = azenio.FormatMove(bestMove)
			if bestMove.ContainsAce() {
				gsClone := gs.Clone()
				gsClone.ApplyMove(bestMove)
				if !gsClone.GameOver && gsClone.CurrentTurn == playerID {
					bestFollow, _ := eng.BestMove(gsClone, tracker)
					bestLabel = fmt.Sprintf("%s / %s", azenio.FormatMove(bestMove), azenio.FormatMove(bestFollow))
				}
			}
			actualDetail = eng.AnalyzeMove(gs, tracker, move)
		}

		if err := gs.ValidateMove(move); err != nil {
			fmt.Printf("Ongeldige zet: %v\n", err)
			moveNum--
			continue
		}

		// Pas-inferentie bijhouden vóór ApplyMove
		if move.IsPass {
			for p := 0; p < numPlayers; p++ {
				if trackers[p] != nil {
					trackers[p].RecordPass(move.PlayerID, gs.Round)
				}
			}
		}
		gs.ApplyMove(move)

		// Trackers bijwerken met de gespeelde zet (ook voor niet-geanalyseerde spelers)
		for p := 0; p < numPlayers; p++ {
			if trackers[p] != nil {
				trackers[p].RecordMove(move)
			}
		}

		// Vervolg-zet verwerken als de gebruiker "11/444" invoerde
		moveLabel := azenio.FormatMove(move)
		if hasFollowCards && !gs.GameOver && gs.CurrentTurn == playerID {
			followCardsStr = strings.TrimSpace(followCardsStr)
			parsed, err := cards.ParseCards(followCardsStr)
			if err != nil {
				fmt.Printf("⚠️  Fout in vervolg-zet: %v\n", err)
			} else {
				followMove := game.Move{PlayerID: playerID, Cards: parsed}
				if err2 := gs.ValidateMove(followMove); err2 != nil {
					fmt.Printf("⚠️  Ongeldige vervolg-zet: %v\n", err2)
				} else {
					gs.ApplyMove(followMove)
					moveLabel = fmt.Sprintf("%s / %s", azenio.FormatMove(move), azenio.FormatMove(followMove))
				}
			}
		}

		if doAnalysis {
			// Als de gespeelde zet gelijk is aan de beste zet van de engine,
			// dan altijd goed teken — ongeacht het scoreverschil tussen BestMove (MCTS-boom)
			// en AnalyzeMove (losse simulaties): beide zijn schattingen van dezelfde zet.
			playedIsBest := game.MovesEqual(bestMove, move)

			var diff float64
			emoji := "✅"
			if !playedIsBest {
				diff = bestEval.Score - actualDetail.WinRate
				if diff > 0.15 {
					emoji = "❌"
				} else if diff > 0.05 {
					emoji = "⚠️ "
				}
			}
			fmt.Printf("%s Gespeeld: %s (score: %.1f%%)\n", emoji, moveLabel, actualDetail.WinRate*100)
			// Toon "Beste was" alleen als een andere zet beter is én het verschil groot genoeg.
			showBest := !playedIsBest &&
				(diff > 0.02 || (bestEval.Score > 0.90 && diff > 0.005))
			if showBest {
				fmt.Printf("   Beste was: %s (score: %.1f%%, verschil: %.1f%%)\n",
					bestLabel, bestEval.Score*100, diff*100)
			}
		} else {
			fmt.Printf("⏭️  Speler %d: %s\n", playerID+1, moveLabel)
		}

		// Toon melding als een speler net gefinished is (na deze zet)
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

func simulateMode(reader *azenio.Reader, cfg settings) {
	azenio.PrintHeader("Simulatie Modus")
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
	gs := game.NewGame(numPlayers, rng, 0)

	fmt.Println("\nStarthanden:")
	for i := 0; i < numPlayers; i++ {
		fmt.Printf("Speler %d: %s\n", i+1, gs.Hands[i])
	}
	fmt.Println()

	trackers := make([]*game.KnowledgeTracker, numPlayers)
	engines := make([]*engine.Engine, numPlayers)
	for i := 0; i < numPlayers; i++ {
		engConfig := engine.DefaultConfig(numPlayers)
		engConfig.Iterations = sims
		engConfig.NumWorkers = cfg.numThreads
		trackers[i] = game.NewKnowledgeTracker(numPlayers, i, gs.Hands[i], gs.DeadCards)
		engines[i] = engine.NewEngine(engConfig)
	}

	prevFinished := 0
	moveNum := 0
	for !gs.GameOver {
		moveNum++
		playerID := gs.CurrentTurn
		eng := engines[playerID]

		bestMove, eval := eng.BestMove(gs, trackers[playerID])

		fmt.Printf("Zet %d | Speler %d: %s (score: %.1f%%) | Kaarten:",
			moveNum, playerID+1, azenio.FormatMove(bestMove), eval.Score*100)
		for i := 0; i < numPlayers; i++ {
			if gs.Finished[i] {
				fmt.Printf(" P%d:✓", i+1)
			} else {
				fmt.Printf(" P%d:%d", i+1, gs.Hands[i].Count())
			}
		}
		fmt.Println()

		// Pas-inferentie bijhouden vóór ApplyMove
		if bestMove.IsPass {
			for i := 0; i < numPlayers; i++ {
				trackers[i].RecordPass(bestMove.PlayerID, gs.Round)
			}
		}
		gs.ApplyMove(bestMove)
		for i := 0; i < numPlayers; i++ {
			trackers[i].RecordMove(bestMove)
		}

		// Toon melding als een speler net gefinished is
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
		azenio.PrintHeader("Spel Voorbij!")
		printRanking(gs)
	}
}

// printRanking toont de eindrangschikking van alle spelers.
func printRanking(gs *game.GameState) {
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
