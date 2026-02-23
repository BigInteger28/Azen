package engine

import (
	"encoding/json"
	"os"
)

// Weights bevat alle instelbare evaluatie-parameters van de engine.
// Kan geladen/opgeslagen worden als weights.json voor zelfverbeterende tuning.
type Weights struct {
	// evalPos: positiebeoordeling aan het einde van simulaties
	AceBonus           float64 `json:"ace_bonus"`           // bonus per aas in hand
	WildBonus          float64 `json:"wild_bonus"`           // bonus per wildcard in hand
	SynergyBonus       float64 `json:"synergy_bonus"`        // extra bonus aas+wild combinatie
	CardDiffWeight     float64 `json:"card_diff_weight"`     // gewicht kaartenverschil vs tegenstander
	KingPenalty        float64 `json:"king_penalty"`         // penalty voor K zonder specials
	QueenPenalty       float64 `json:"queen_penalty"`        // penalty voor Q zonder specials
	IsolatedLowPenalty float64 `json:"isolated_low_penalty"` // penalty voor enkelvoudige lage kaart (3-7)
	ClusterBonus       float64 `json:"cluster_bonus"`        // bonus per extra kaart in cluster

	// smartRandom: rollout-gedrag (lagere waarde = minder kans die kaart te spelen)
	AcePlayFactor     float64 `json:"ace_play_factor"`     // basis Pow(x, nAssen)
	WildPlayFactor    float64 `json:"wild_play_factor"`    // basis Pow(x, nWilds)
	SynergyPenalty    float64 `json:"synergy_penalty"`     // extra vermenigvuldiger aas+wild samen
	RankPreference    float64 `json:"rank_preference"`     // voorkeur voor lagere ranks
	PassBase          float64 `json:"pass_base"`           // basiswaarschijnlijkheid om te passen
	PassSpecialFactor float64 `json:"pass_special_factor"` // extra pasdrempel per special-ratio
}

// DefaultWeights geeft de huidig best bekende handmatig ingestelde weights.
func DefaultWeights() Weights {
	return Weights{
		AceBonus:           0.14,
		WildBonus:          0.10,
		SynergyBonus:       0.05,
		CardDiffWeight:     0.05,
		KingPenalty:        0.04,
		QueenPenalty:       0.02,
		IsolatedLowPenalty: 0.03,
		ClusterBonus:       0.02,

		AcePlayFactor:     0.20,
		WildPlayFactor:    0.35,
		SynergyPenalty:    0.50,
		RankPreference:    0.10,
		PassBase:          0.10,
		PassSpecialFactor: 0.25,
	}
}

// WeightParam beschrijft één instelbare parameter met naam, pointer en grenzen.
type WeightParam struct {
	Name string
	Ptr  *float64
	Min  float64
	Max  float64
}

// Params geeft een slice van alle parameters, klaar voor iteratie door de tuner.
func (w *Weights) Params() []WeightParam {
	return []WeightParam{
		{"ace_bonus", &w.AceBonus, 0.0, 0.50},
		{"wild_bonus", &w.WildBonus, 0.0, 0.40},
		{"synergy_bonus", &w.SynergyBonus, 0.0, 0.30},
		{"card_diff_weight", &w.CardDiffWeight, 0.01, 0.20},
		{"king_penalty", &w.KingPenalty, 0.0, 0.20},
		{"queen_penalty", &w.QueenPenalty, 0.0, 0.15},
		{"isolated_low_penalty", &w.IsolatedLowPenalty, 0.0, 0.15},
		{"cluster_bonus", &w.ClusterBonus, 0.0, 0.15},
		{"ace_play_factor", &w.AcePlayFactor, 0.05, 0.80},
		{"wild_play_factor", &w.WildPlayFactor, 0.05, 0.80},
		{"synergy_penalty", &w.SynergyPenalty, 0.10, 0.95},
		{"rank_preference", &w.RankPreference, 0.0, 0.50},
		{"pass_base", &w.PassBase, 0.03, 0.40},
		{"pass_special_factor", &w.PassSpecialFactor, 0.0, 0.50},
	}
}

// clamp zorgt dat een waarde binnen [min, max] blijft.
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// LoadWeights laadt weights uit een JSON-bestand.
// Geeft DefaultWeights terug als het bestand niet bestaat of ongeldig is.
func LoadWeights(path string) (Weights, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultWeights(), err
	}
	w := DefaultWeights() // start met defaults zodat ontbrekende velden worden ingevuld
	if err := json.Unmarshal(data, &w); err != nil {
		return DefaultWeights(), err
	}
	return w, nil
}

// SaveWeights slaat weights op als een leesbaar JSON-bestand.
func SaveWeights(w Weights, path string) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
