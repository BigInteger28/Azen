package io

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/azen-engine/pkg/cards"
	"github.com/azen-engine/pkg/game"
)

// GameLog represents a recorded game
type GameLog struct {
	NumPlayers int
	Hands      [][]cards.Card // Initial hands (if known)
	DeadCards  []cards.Card
	Moves      []game.Move
	Winner     int
}

// SaveGame writes a game log to file
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

// LoadGame reads a game log from file
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
				cc, err := cards.ParseCards(parts[1])
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
			cc, err := cards.ParseCards(strings.TrimPrefix(line, "dead:"))
			if err != nil {
				return nil, err
			}
			log.DeadCards = cc
		}
	}

	return log, scanner.Err()
}

func parseMoveLog(line string) (game.Move, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return game.Move{}, fmt.Errorf("invalid move format")
	}

	pid := 0
	fmt.Sscanf(parts[0], "P%d", &pid)

	if strings.TrimSpace(parts[1]) == "PASS" {
		return game.PassMove(pid), nil
	}

	cc, err := cards.ParseCards(parts[1])
	if err != nil {
		return game.Move{}, err
	}

	return game.Move{PlayerID: pid, Cards: cc}, nil
}
