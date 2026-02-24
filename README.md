# AZEN Engine 🃏

Een AI-engine voor het AZEN kaartspel, gebouwd in Go. Gebruikt **Information Set Monte Carlo Tree Search (IS-MCTS)** om de beste zet te berekenen, zelfs met onvolledige informatie over de handen van tegenstanders.

De volledige engine zit in één bestand: `azen-termux.go`.

---

## Installatie & Starten

Vereist: Go 1.22+

```bash
go build azen-termux.go
./azen-termux
```

Of direct draaien:

```bash
go run azen-termux.go
```

---

## Spelregels

### Doel
Als eerste alle kaarten kwijtraken.

### Kaarten

| Kaart | Symbool | Betekenis |
|-------|---------|-----------|
| Drie t/m Negen | `3` `4` `5` `6` `7` `8` `9` | Normale kaarten, laag naar hoog |
| Tien | `X` | Normale kaart |
| Boer | `J` | Normale kaart |
| Vrouw | `Q` | Normale kaart |
| Koning | `K` | Hoogste normale kaart |
| **Aas** | `1` | **Sterkste naturelle kaart** (boven Koning) |
| **Twee** | `2` | **Wildcard** — telt als elke willekeurige kaart |
| **Joker** | `0` | **Reset-kaart** — verslaat alles en opent een nieuwe ronde |

**Rangvolgorde (laag → hoog):** `3 4 5 6 7 8 9 X J Q K 1`

### Verloop van een beurt

Een ronde begint altijd **open**: de eerste speler legt een combinatie naar keuze.

Daarna moet elke volgende speler:
- **Evenveel kaarten** leggen als er op tafel liggen
- Met een **hogere rank** dan de kaarten op tafel
- Of **passen**

Als alle andere spelers passen, begint de laatste speler die speelde een nieuwe open ronde.

### Speciale kaarten

**Twee `2` (wildcard)**
- Kan elke kaart vervangen
- Mag gecombineerd worden met normale kaarten van dezelfde rank
- Voorbeeld: `K 2` = paar koningen

**Joker `0` (reset)**
- Mag op élke tafelstand gespeeld worden, ongeacht rank of aantal
- Reset de ronde: de speler die de joker speelt, opent direct een nieuwe ronde
- Mag gecombineerd worden met andere jokers of wildcards (`2`), **niet** met normale kaarten
- Er zijn slechts **2 jokers** in het spel

### Combinaties

Alle kaarten in een combinatie moeten dezelfde rank hebben (wildcards vullen aan):

| Combinatie | Voorbeeld |
|-----------|-----------|
| Enkele kaart | `7` |
| Paar | `Q Q` |
| Triple | `9 9 9` |
| Viertal | `K K K K` |
| Met wildcard | `J 2` (paar boeren), `9 9 2` (triple negens) |
| Joker reset | `0` (reset + open), `0 2` (reset met wildcard) |

### Slash-notatie `/`

Wanneer een speler een joker speelt (reset) en daarna direct een nieuwe combinatie opent, wordt dit genoteerd als:

```
0 / K K     → joker reset, opent met paar koningen
0 2 / 5 5   → joker+wildcard reset, opent met paar vijven
```

---

## Kaart Invoer

Bij het invoeren van kaarten gebruik je de symbolen uit de tabel hierboven:

```
Jouw hand:  3 3 4 5 5 7 8 9 X X J Q K K 1 2 2 0
```

**Passen:** typ `pass`, `p` of `-`

**Slash-zet:** typ `0 / K K` (joker reset gevolgd door opening)

---

## Modi

### 1. Play Mode — Spelen met engine-hulp
- Voer jouw 18 kaarten in
- De engine berekent de beste zet elke beurt
- Voer de zetten van tegenstanders handmatig in
- De engine houdt bij welke kaarten tegenstanders mogelijk hebben

### 2. Analyze Mode — Partij analyseren
- Voer de starthanden van alle spelers in
- Voer elke gespeelde zet in
- De engine analyseert elke zet:
  - ✅ Goede zet
  - ⚠️ Onnauwkeurigheid (5–15% slechter)
  - ❌ Blunder (15%+ slechter)
  - Toont wat de beste zet was geweest

### 3. Simulate Mode — Engine vs Engine
- Kijk hoe de engine tegen zichzelf speelt
- Handig om de engine-kwaliteit te testen

---

## Engine Details

### IS-MCTS (Information Set Monte Carlo Tree Search)

Speciaal ontworpen voor kaartspellen met verborgen informatie:

1. **Determinisatie** — genereer een geloofwaardige verdeling van onbekende kaarten op basis van wat je weet (passen, gespeelde kaarten)
2. **Tree Search** — zoek de beste zet in deze gesimuleerde wereld via UCB1-selectie
3. **Simulatie** — speel de partij willekeurig uit tot het einde
4. **Backpropagatie** — verwerk het resultaat terug in de boom
5. **Herhaal** duizenden keren en kies de zet met de hoogste winratio

### Sterke-kaarten-bias

Bij het genereren van mogelijke tegenstander-handen (determinisatie) wordt geprioriteerd dat de tegenstander **assen (1)** en **wildcards (2)** bezit. Dit is statistisch verantwoord: met 4 exemplaren per rank in een deck van 54 kaarten heeft de tegenstander ~84% kans op minstens één aas of wildcard als jij ze niet hebt.

Jokers (0) worden **niet** geprioriteerd: er zijn slechts 2 jokers in het deck (~76% kans), te laag om altijd te assumeren.

### Filterlogica (filterDominatedMoves)

Vóór de MCTS-zoekfase worden duidelijk verspillende zetten gefilterd:
- Wild-zetten die slechts 1 rank hoger zijn dan de beste naturelle zet worden verwijderd
- Oversized "slash"-combos worden gefilterd als een simpelere naturelle zet volstaat
- Dit versnelt de engine en voorkomt blunders bij hoge iteratiecounts

### Geheugen & Kennisopbouw

De engine houdt bij:
- Welke kaarten gespeeld zijn
- Wanneer een speler past (→ ze hebben geen kaarten boven de tabelrank)
- Welke kaarten uitgesloten zijn per speler

---

## Configuratie

Pas het aantal iteraties aan via de interface of direct in de code:

```go
cfg := DefaultConfig(numPlayers)
cfg.Iterations = 50000   // meer = sterker maar trager
cfg.MaxTime = 10 * time.Second
```

Aanbevolen iteraties:
| Doel | Iteraties |
|------|-----------|
| Snel testen | 5 000 |
| Normaal spel | 50 000 |
| Sterk spel | 200 000+ |
