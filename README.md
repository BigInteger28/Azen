# AZEN Engine 🃏

Een AI-engine voor het AZEN kaartspel, gebouwd in Go. Gebruikt **Information Set Monte Carlo Tree Search (ISMCTS)** om de beste zet te berekenen, zelfs met onvolledige informatie over de handen van tegenstanders.

## Installatie

Vereist: Go 1.22+ geïnstalleerd op je systeem.

```bash
# Clone of kopieer de bestanden naar een map
cd azen-engine

# Build
go build -o azen.exe ./cmd/play
```

## Gebruik

```bash
./azen.exe
```

Je krijgt 3 modi:

### 1. Play Mode (Spelen met engine hulp)
- Voer je 18 kaarten in
- Engine berekent de beste zet voor elke beurt
- Voer de zetten van tegenstanders in
- Engine houdt bij wat tegenstanders mogelijk hebben

### 2. Analyze Mode (Partij analyseren)
- Voer de starthanden van alle spelers in
- Voer elke zet in
- Engine analyseert elke zet en toont:
  - ✅ Goede zet
  - ⚠️ Onnauwkeurigheid
  - ❌ Blunder
  - Wat de beste zet was geweest

### 3. Simulate Mode (Engine vs Engine)
- Kijk hoe de engine tegen zichzelf speelt
- Handig om de engine-kwaliteit te testen

## Kaart Invoer Formaat

| Kaart | Formaat | Voorbeeld |
|-------|---------|-----------|
| 3 t/m 9 | `{rank}{suit}` | `3h`, `7d`, `9c` |
| 10 | `T{suit}` | `Th`, `Td` |
| Boer | `J{suit}` | `Jh`, `Js` |
| Vrouw | `Q{suit}` | `Qc`, `Qd` |
| Koning | `K{suit}` | `Kh`, `Ks` |
| 2 (wild) | `2{suit}` | `2h`, `2c` |
| Aas | `A{suit}` | `Ah`, `As` |
| Joker | `Jo` | `Jo` |


**Passen:** type `pass` of `p` of `-`

## Spelregels Samenvatting

- Elke speler krijgt 18 kaarten
- Doel: als eerste alle kaarten kwijtraken
- Elke beurt: zelfde aantal kaarten leggen, hoger dan vorige
- **3** is laagst, **K** is hoogst (normaal)
- **2** en **Joker**: wildcards, tellen als elke kaart
- **Aas**: wildcard + reset de ronde (jij begint opnieuw)
- Je mag altijd passen

## Engine Details

### ISMCTS (Information Set Monte Carlo Tree Search)

De engine gebruikt ISMCTS, speciaal ontworpen voor spellen met verborgen informatie:

1. **Determinization**: Genereer een mogelijke verdeling van onbekende kaarten
2. **Tree Search**: Zoek de beste zet in deze gesimuleerde wereld
3. **Herhaal**: Over duizenden simulaties, convergeer naar de robuust beste zet
4. **Evaluatie**: Combinatie van:
   - Win/verlies uit random playouts
   - Heuristische hand-evaluatie (wildcards, paren, tempo)

### Configuratie

Standaard: 5000 simulaties, max 5 seconden per zet. Pas aan in de code:

```go
engConfig := engine.DefaultConfig()
engConfig.Simulations = 10000  // Meer simulaties = sterker maar trager
engConfig.MaxTime = 10 * time.Second
engConfig.Verbose = true       // Toon analyse details
```

## Projectstructuur

```
azen-engine/
├── cmd/play/main.go          # Hoofdprogramma met 3 modi
├── pkg/
│   ├── cards/cards.go        # Kaart definities, deck, hand
│   ├── game/
│   │   ├── game.go           # Spelregels, state, validatie
│   │   └── knowledge.go      # Tegenstander informatie tracker
│   ├── engine/
│   │   ├── engine.go         # ISMCTS zoekalgoritme
│   │   └── heuristics.go     # Hand evaluatie heuristieken
│   └── io/io.go              # Input/output helpers
├── go.mod
└── README.md
```

## Toekomstige Uitbreidingen

- [ ] Parallel ISMCTS (meerdere goroutines)
- [ ] Opponent modeling (leren van speelstijl)
- [ ] Opening book (veelvoorkomende openingszetten)
- [ ] PGN-achtig formaat voor game export/import
- [ ] Web interface
- [ ] Elo rating systeem voor engine versies
