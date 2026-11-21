// Package game contains the pure game logic of Ardian (no I/O, no Telegram, no DB).
package game

// ---------- Elements ----------

type Element int

const (
	Fire Element = iota
	Water
	Earth
	Wind
)

const NumElements = 4

var ElementNames = [NumElements]string{"آتش", "آب", "خاک", "باد"}
var ElementEmoji = [NumElements]string{"🔥", "💧", "🪨", "🌪"}

func (e Element) String() string { return ElementEmoji[e] + " " + ElementNames[e] }

// Beats implements the cycle: Fire → Wind → Earth → Water → Fire.
func (e Element) Beats(o Element) bool {
	switch e {
	case Fire:
		return o == Wind
	case Wind:
		return o == Earth
	case Earth:
		return o == Water
	case Water:
		return o == Fire
	}
	return false
}

// ElementMod returns the magic damage modifier in percent (+10 / -10 / 0).
func ElementMod(att, def Element) float64 {
	if att.Beats(def) {
		return 10
	}
	if def.Beats(att) {
		return -10
	}
	return 0
}

// ---------- Races ----------

type Race int

const (
	Human Race = iota
	Elf
	DarkElf
	Dwarf
	Angel
	Tiger
	Dragon
	Wolf
	Fox
)

const NumRaces = 9

var RaceNames = [NumRaces]string{
	"انسان", "الف", "دارک الف", "دورف", "فرشته",
	"نیمه‌ببر", "نیمه‌اژدها", "نیمه‌گرگ", "نیمه‌روباه",
}
var RaceEmoji = [NumRaces]string{"👤", "🧝", "🌑", "🪨", "👼", "🐯", "🐉", "🐺", "🦊"}

func (r Race) String() string { return RaceEmoji[r] + " " + RaceNames[r] }

// ---------- Stats ----------

type Stat int

const (
	StatHP Stat = iota
	StatMana
	StatPower
	StatControl
	StatSpeed
	StatDefense
	StatPerception
	StatLuck
	StatCrit
)

const NumStats = 9

var StatNames = [NumStats]string{"HP", "مانا", "قدرت", "کنترل", "سرعت", "دفاع", "ادراک", "شانس", "کریتیکال"}
var StatEmoji = [NumStats]string{"❤️", "💧", "⚔️", "🔮", "💨", "🛡", "👁", "🍀", "💥"}

// StatMax returns the cap of a single stat (base + allocated).
func StatMax(s Stat) int {
	if s == StatControl {
		return 125
	}
	return 100
}

// Base stats per race: [0]=male, [1]=female.
// Order: HP, Mana, Power, Control, Speed, Defense, Perception, Luck, Crit.
var baseStats = map[Race][2][NumStats]int{
	Human:   {{10, 10, 6, 3, 4, 5, 4, 3, 1}, {10, 12, 4, 5, 4, 3, 6, 5, 1}},
	Elf:     {{10, 14, 5, 7, 8, 2, 1, 3, 1}, {10, 14, 4, 8, 5, 1, 4, 2, 1}},
	DarkElf: {{10, 14, 6, 4, 9, 4, 3, 1, 1}, {10, 14, 4, 5, 5, 4, 5, 2, 1}},
	Dwarf:   {{12, 5, 10, 1, 2, 8, 3, 5, 2}, {12, 6, 9, 3, 5, 4, 4, 4, 3}},
	Angel:   {{10, 15, 4, 10, 8, 2, 4, 7, 1}, {10, 15, 3, 11, 9, 1, 3, 8, 1}},
	Tiger:   {{11, 8, 9, 3, 9, 5, 4, 3, 1}, {11, 9, 8, 4, 8, 4, 5, 2, 2}},
	Dragon:  {{14, 16, 15, 1, 3, 8, 1, 2, 1}, {14, 17, 14, 1, 4, 6, 2, 2, 1}},
	Wolf:    {{12, 9, 13, 4, 7, 4, 1, 2, 1}, {11, 10, 10, 6, 5, 2, 1, 2, 1}},
	Fox:     {{10, 10, 5, 7, 7, 2, 3, 2, 1}, {10, 10, 4, 8, 7, 1, 4, 2, 1}},
}

func BaseStat(r Race, female bool, s Stat) int {
	i := 0
	if female {
		i = 1
	}
	return baseStats[r][i][s]
}

// ---------- Tunable constants (balance list lives here) ----------

const (
	MaxLevel        = 100
	PointsPerLevel  = 5
	DeathSeconds    = 300
	HPPerPoint      = 60.0
	ManaPerPoint    = 50.0
	PhysPerPower    = 20.0 // physical damage per power point
	PhysSpread      = 0.10 // ± random range of physical damage (spec: "random range", value assumed)
	BaseRegenPerMin = 50.0
	DefensePerPoint = 0.5 // % damage reduction
	CritPerPoint    = 0.1 // % crit chance per point
	CritMult        = 4.0
	ZetsuMult       = 2.5
	GlobalCDSec     = 3.0
	PhysCDSec       = 5.0
	MaxDefenseRed   = 90.0 // cap of total damage reduction (%)
	OffIncPerLevel  = 10.0 // offensive spell center growth per upgrade level
	DefaultEffDur   = 60.0 // buff duration (s)

	XPHit, GoldHit   = 25, 50
	XPKill, GoldKill = 100, 200
)

func XPNeeded(level int) int64 { return int64(100 * level) }

// SpellLevel returns the upgrade level (1..10) from the number of uses.
// Going from n to n+1 needs 5*n uses (225 in total).
func SpellLevel(uses int) int {
	lvl := 1
	for l := 2; l <= 10; l++ {
		if uses >= 5*l*(l-1)/2 {
			lvl = l
		}
	}
	return lvl
}

// UsesToNext returns uses remaining for the next upgrade level (0 at max).
func UsesToNext(uses int) int {
	l := SpellLevel(uses)
	if l >= 10 {
		return 0
	}
	return 5*(l+1)*l/2 - uses
}

// MagicPowerShare is how much of the power-based damage is added to spells
// (spec: 30%). A var so the balance test can sweep it.
var MagicPowerShare = 0.30
