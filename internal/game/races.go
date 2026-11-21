package game

import "math"

// Race ability data (section 10.5 of the design doc).
// Tiers: every race ability has "tiers" that are bought with gold.
// A tier can only be bought once the player reached its minimum level.

var RaceAbilityName = [NumRaces]string{
	"", "شینرا ایبوکی", "زتسو", "کوکتسو", "رایکو", "راستسو", "ریوکتسو", "شینگو", "کیوبی",
}

var raceAbilityByNorm = map[string]Race{}

func init() {
	for r := Elf; r < NumRaces; r++ {
		raceAbilityByNorm[Norm(RaceAbilityName[r])] = r
	}
}

// RaceAbilityByName returns the race whose ability is called text.
func RaceAbilityByName(text string) (Race, bool) {
	r, ok := raceAbilityByNorm[Norm(text)]
	return r, ok
}

// RaceActive reports whether the race ability must be triggered manually.
func RaceActive(r Race) bool { return r == Elf || r == DarkElf || r == Dwarf || r == Fox }

func RaceMaxTier(r Race) int {
	switch r {
	case Human:
		return 0
	case Angel:
		return 20
	case Fox:
		return 9
	}
	return 10
}

// RaceInitialTier is the tier a new character owns for free.
func RaceInitialTier(r Race) int {
	if r == Human || r == DarkElf {
		return 0
	}
	return 1
}

// RaceTierMinLevel is the minimum player level needed to own the tier.
func RaceTierMinLevel(r Race, tier int) int {
	if tier <= 1 {
		return 1
	}
	switch r {
	case Fox:
		return 1 + (tier-1)*11
	case Angel:
		return (tier - 1) * 5
	}
	return (tier - 1) * 10
}

// RaceTierPrice is the gold price of buying the given tier.
func RaceTierPrice(r Race, tier int) int64 {
	t := int64(tier)
	switch r {
	case Human:
		return 0
	case DarkElf:
		return 500 * t
	case Angel:
		return 100 * (t - 1)
	}
	return 300 * (t - 1)
}

func clampTier(t, max int) int {
	if t < 1 {
		return 1
	}
	if t > max {
		return max
	}
	return t
}

// ---- Elf: Shinra Ibuki ----

type ElfVals struct{ Fixed, Pct, Cap float64 }

const ElfCooldown = 30.0

func ElfTier(t int) ElfVals {
	t = clampTier(t, 10)
	return ElfVals{Fixed: float64(40 + 10*t), Pct: float64(5 + 5*t), Cap: float64(170 + 30*t)}
}

// ---- Dark elf: Zetsu ----

// DarkElfVals returns the chance (%) and cooldown (s) for an effective level.
func DarkElfVals(effLevel int) (chance, cd float64) {
	if effLevel < 1 {
		effLevel = 1
	}
	if effLevel > 100 {
		effLevel = 100
	}
	chance = float64(effLevel)
	cd = 10 + float64(effLevel-1)*110.0/99.0
	return math.Round(chance*100) / 100, math.Round(cd)
}

// ---- Dwarf: Kotsu ----

type DwarfVals struct {
	Reflect float64
	Hits    int
	CD      float64
}

var dwarfTable = [10]DwarfVals{
	{0.5, 1, 10}, {5, 1, 20}, {10, 1, 32}, {15, 1, 44}, {20, 2, 56},
	{25, 2, 68}, {30, 2, 80}, {35, 2, 92}, {40, 3, 104}, {50, 3, 120},
}

func DwarfTier(t int) DwarfVals { return dwarfTable[clampTier(t, 10)-1] }

// ---- Angel: Raiko (passive, drop boost; drops are not implemented yet) ----

// AngelBoosts returns the multipliers of the rarest, second and third rarest
// drop for a scale level (1..20).
func AngelBoosts(l int) (rare, second, third float64) {
	l = clampTier(l, 20)
	fl := float64(l)
	return 1 + 0.45*fl, 1 + 0.2*fl, 1 + 0.075*fl
}

// ---- Tiger: Rastsu ----

func TigerVals(t int) (mid, low float64) {
	t = clampTier(t, 10)
	return float64(5 * t), float64(10 * t)
}

// ---- Dragon: Ryuketsu ----

func DragonVals(t int) (dealt, taken float64) {
	t = clampTier(t, 10)
	return float64(20 * t), float64(5 * t)
}

// ---- Wolf: Shingo ----

const WolfMaxStacks = 15

func WolfPerStack(t int) float64 { return float64(2 * clampTier(t, 10)) }

// ---- Fox: Kyubi ----

var foxDodges = [9]int{1, 1, 1, 1, 1, 1, 1, 1, 2}
var foxCD = [9]float64{25, 27, 30, 33, 36, 38, 40, 42, 45}

func FoxTier(t int) (dodges int, cd float64) {
	t = clampTier(t, 9)
	return foxDodges[t-1], foxCD[t-1]
}

// ---- Human ----

var humanWeak = [10]float64{30, 28, 26, 23, 21, 19, 17, 14, 12, 10}

// HumanWeakness is the damage/effect penalty (%) of the second element
// depending on the upgrade level (1..10) of the spell.
func HumanWeakness(spellLevel int) float64 {
	if spellLevel < 1 {
		spellLevel = 1
	}
	if spellLevel > 10 {
		spellLevel = 10
	}
	return humanWeak[spellLevel-1]
}

const (
	HumanBuyMinLevel = 50
	HumanBuyMax      = 100
)

// HumanPointPrice is the gold price of the n-th (1-based) bought stat point.
func HumanPointPrice(n int) int64 { return int64(100 * ((n-1)/20 + 1)) }
