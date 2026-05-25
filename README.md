# AZEN Engine

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
| **Twee** | `2` | **Wildcard** — vult aan als elke willekeurige rank |
| **Joker** | `0` | **Reset-kaart** — verslaat altijd en opent een nieuwe ronde |

**Rangvolgorde (laag → hoog):** `3 4 5 6 7 8 9 X J Q K 1`

Wildcards en jokers hebben geen vaste rank; ze tellen niet mee als normaal kaart in de rangvolgorde.

### Verloop van een beurt

Een ronde begint altijd **open**: de eerste speler legt een combinatie naar keuze.

Daarna moet elke volgende speler:
- **Evenveel kaarten** leggen als er op tafel liggen
- Met een **hogere rank** dan de kaarten op tafel
- Of **passen**

Als alle andere spelers passen, begint de speler die het laatste speelde een nieuwe open ronde.

### Speciale kaarten

**Twee `2` (wildcard)**
- Vult een combinatie aan als elke rank
- Voorbeeld: `K 2` = paar koningen; `9 9 2` = triple negens
- Als antwoord mag een wildcard de tafelrank niet verlagen — de normale kaarten in de combinatie moeten de tafel verslaan

**Joker `0` (reset)**
- Mag altijd gespeeld worden, ongeacht rank of aantal op tafel
- Reset de ronde: de speler die de joker speelt, opent direct een nieuwe ronde
- **Mag gecombineerd worden met elke kaart** — normale kaarten, wildcards, of alleen
- Als de joker gecombineerd wordt met normale kaarten, moeten die normale kaarten wel dezelfde rank hebben
- Er zijn **2 jokers** per deck (4 bij vierspelersvariant met 2 decks)

### Combinaties

Alle kaarten in een combinatie moeten dezelfde rank hebben (wildcards of joker vullen aan):

| Combinatie | Voorbeeld |
|-----------|-----------|
| Enkele kaart | `7` |
| Paar | `Q Q` |
| Triple | `9 9 9` |
| Viertal | `K K K K` |
| Met wildcard | `J 2` (paar boeren), `9 9 2` (triple negens) |
| Joker reset alleen | `0`, `0 0` |
| Joker + wildcard | `0 2`, `0 2 2` |
| Joker + normale kaarten | `7 0` (paar: 7 + joker), `4 4 4 2 2 0` (6-kaartcombo) |

Er is geen maximum aan het aantal kaarten in één combinatie — `4 4 4 4 2 2 2 0 0` is geldig als alle normale kaarten dezelfde rank hebben.

### Slash-notatie `/`

Wanneer een speler een joker speelt (reset) en daarna direct een nieuwe combinatie opent, wordt dit genoteerd als:

```
0 / K K       → joker reset, opent met paar koningen
7 0 / 4 4 4   → joker + 7 als antwoord (reset), opent met triple vieren
0 2 / 5 5     → joker + wildcard reset, opent met paar vijven
```

---

## Kaart Invoer

Bij het invoeren van kaarten gebruik je de symbolen uit de tabel hierboven:

```
Jouw hand:  3 3 4 5 5 7 8 9 X X J Q K K 1 2 2 0
```

Kaarten mogen aaneengesloten of met spaties ingevoerd worden:

```
3344XJ2  of  3 3 4 4 X J 2  of  3,3,4,4,X,J,2
```

**Passen:** typ `pass`, `p` of `-`

**Slash-zet:** typ `70/444` of `7 0 / 4 4 4` (joker+7 reset, gevolgd door triple 4s)

---

## Modi

### [0] Instellingen
- Stel het aantal threads in dat de engine gebruikt (standaard: 2)
- Meer threads = sneller zoeken bij hoge iteratiecounts

### [1] Spelen — Engine-hulp bij een lopende partij
- Voer jouw starthand in
- De engine berekent de beste zet elke beurt
- Voer de zetten van tegenstanders handmatig in
- De engine houdt bij welke kaarten tegenstanders waarschijnlijk hebben op basis van gespeelde kaarten en passes

### [2] Analyse — Partij stap voor stap
- Voer de starthanden van alle spelers in
- Voer elke gespeelde zet interactief in
- De engine analyseert elke zet en toont alternatieven:
  - ✅ Goede zet (of maximaal ~2% slechter dan het beste)
  - ⚠️ Onnauwkeurigheid (2–15% slechter)
  - ❌ Blunder (15%+ slechter of gedwongen winst gemist)

### [3] Simuleer — Engine vs Engine
- Kijk hoe de engine tegen zichzelf speelt
- Handig om engine-kwaliteit te testen en spelverloop te observeren

### [4] Snelle analyse — Volledige partij in één keer
- Plak alle zetten van een partij als losse tokens in één invoer
- De engine analyseert alle zetten tegelijk
- Gebruikt **OmniscientMode**: met alle handen bekend selecteert de engine op winratio (niet bezoekcount), wat nauwkeuriger is
- Toont de top-alternatieven per zet

---

## Engine Details

### IS-MCTS (Information Set Monte Carlo Tree Search)

Speciaal ontworpen voor kaartspellen met verborgen informatie:

1. **Determinisatie** — genereer een geloofwaardige verdeling van onbekende kaarten op basis van wat je weet (passen, gespeelde kaarten, sterke-kaarten-bias)
2. **Tree Search** — zoek de beste zet via UCB1-selectie
3. **Simulatie** — speel de partij willekeurig uit tot het einde met gewogen heuristiek
4. **Backpropagatie** — verwerk het resultaat terug in de boom
5. **Herhaal** duizenden keren en kies de zet met de beste statistieken

### Sterke-kaarten-bias

Bij het genereren van mogelijke tegenstander-handen wordt geprioriteerd dat de tegenstander **assen (1)** en **wildcards (2)** bezit. Dit is statistisch verantwoord: met 4 exemplaren per rank in een deck van 54 kaarten heeft de tegenstander ~84% kans op minstens één aas of wildcard als jij ze niet hebt.

Jokers (0) worden **niet** geprioriteerd: er zijn slechts 2 jokers in het deck (~76% kans).

### Filterlogica (filterDominatedMoves)

Vóór de MCTS-zoekfase worden aantoonbaar verspillende zetten gefilterd. De engine werkt in een kostenhiërarchie:

1. **Naturelle zet** (geen wildcard, geen joker) — goedkoopst
2. **Wildcard-zet** (met `2`, zonder joker) — midden
3. **Joker-zet** (met `0`) — duurste; joker bewaren voor noodsituaties

Filterregels bij antwoord op een ronde:
- Als een **naturelle zet de tafel al verslaat**: filter alle zetten met wildcards of jokers, behalve pure joker-resets
- Als een **wildcard-zet de tafel verslaat** maar geen naturelle zet: filter joker-houdende zetten (`8 8 2 2` wordt verkozen boven `8 8 2 0`)
- Anders: bewaar alles wat de tafel kan verslaan (resets of hogere rank)

Dit voorkomt blunders als `8 8 2` (wildcard verspild) wanneer `4 4 4` de tafel al verslaat — de wildcard blijft bewaard voor situaties waar hij écht nodig is (bijv. antwoord op triple aas via `0 2 2`).

### 2-speler tempo-strategie

In een partij met 2 spelers is **tempo** cruciaal: de speler die de open ronde heeft, dicteert het spel. De engine past speciale logica toe:

- **Pas-onderdrukking**: passes in antwoordrondes worden sterk gediscourageerd, maar niet geblinddoekt overschreven — als de beste alternatieve zet minder dan ~2% winstkans heeft (bijv. `0 2 2` op triple aas = 0.4%), kiest de engine toch voor pas om speciale kaarten te bewaren
- **Tempo-bonus**: de open ronde is +0.20 waard in positie-evaluatie; tegenstander die de open ronde heeft is −0.14
- **Joker-waarde**: in 2-speler mid-game is de joker extra waardevol (+0.55 vs +0.33 in meer-spelers), omdat hij de enige manier is om op triple aas te antwoorden via `0 2 2`
- **Pass-override**: als PASS het beste scoort in MCTS maar een alternatieve zet binnen 8% zit en ≥2% winstkans heeft, kiest de engine toch voor spelen

### Geheugen & Kennisopbouw (KnowledgeTracker)

De engine houdt bij:
- Welke kaarten gespeeld zijn
- Wanneer een speler past → ze hebben geen naturelle kaarten boven de tabelrank
- Welke kaarten uitgesloten zijn per speler (negatieve kennis)

---

## Configuratie

Pas instellingen aan via de interface (optie 0) of direct in de code:

```go
cfg := DefaultConfig(numPlayers)
cfg.Iterations = 50000        // meer = sterker maar trager
cfg.NumWorkers = 4            // aantal parallelle threads
cfg.MaxTime = 10 * time.Second // tijdslimiet per zet
```

Standaard iteraties (instelbaar via de interface):

| Doel | Iteraties |
|------|-----------|
| Snel testen | 3 000 |
| Normaal spel | 10 000 |
| Sterk spel | 50 000+ |

Bij **Snelle analyse** (modus 4) is de aanbevolen waarde 10 000–50 000 voor betrouwbare resultaten.

---

## Deck-samenstelling

| Variant | Decks | Kaarten totaal | Jokers |
|---------|-------|----------------|--------|
| 2–3 spelers | 1 | 54 | 2 |
| 4 spelers | 2 | 108 | 4 |

Per rank zijn er 4 exemplaren in een enkelvoudig deck (harten, ruiten, klaveren, schoppen).
