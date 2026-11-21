package game

import (
	"math"
	"time"
)

const (
	DeathGoldLossPct   = 10
	DeathXPLossPct     = 25
	SpeedRegenPerPoint = 0.03 // +3% recovery per speed point
	RaceKey            = "race"
)

// Effect is an active buff/debuff sitting on a fighter.
type Effect struct {
	Key     string // replacement key (ability norm-name or RaceKey)
	Name    string
	Type    EffType
	Value   float64
	Filter  Filter
	Charges int
	OnMiss  bool
	Timed   bool
	Expires time.Time
	// heal over time
	Every     time.Duration
	TicksLeft int
	Next      time.Time
}

func (e *Effect) alive(now time.Time) bool {
	if !now.Before(e.Expires) {
		return false
	}
	if e.Type == EffHoT {
		return e.TicksLeft > 0
	}
	return e.Timed || e.Charges > 0
}

type manaEntry struct {
	t   time.Time
	amt float64
}

// Fighter is a player with its persistent and runtime (in-memory) state.
type Fighter struct {
	// persistent
	ID        int64
	Name      string
	Race      Race
	Female    bool
	Elems     []Element // 1 element (2 for humans: first normal, second weaker)
	Level     int
	XP, Gold  int64
	Points    int
	Alloc     [NumStats]int
	Bought    int // human: stat points bought from the shop
	RaceTier  int
	Uses      map[string]int
	HP, Mana  float64
	Updated   time.Time
	DeadUntil time.Time
	Kills     int
	Resets    int // number of stat resets done
	ClanID    int64
	Inv       map[string]int // item id → count (not equipped)
	Equipped  [NumSlots]string

	// runtime only
	Effects     []*Effect
	Ready       map[string]time.Time // ability norm-name (or RaceKey) → ready time
	GlobalReady time.Time
	PhysReady   time.Time
	Stacks      int // wolf
	RastsuReady time.Time
	LastNotice  time.Time // last "wait" message shown in chat
	manaLog     []manaEntry
	LostGold    int64 // set by Die: what the last death cost
	LostXP      int64
	Gear        [NumStats]int // stat bonuses from equipment
	WeaponDmg   float64       // flat physical damage of the main weapon
}

// NewFighter creates a level-1 character with full HP and mana.
func NewFighter(id int64, name string, race Race, female bool, elems []Element, now time.Time) *Fighter {
	f := &Fighter{
		ID: id, Name: name, Race: race, Female: female, Elems: elems,
		Level: 1, Points: PointsPerLevel, RaceTier: RaceInitialTier(race),
		Uses: map[string]int{}, Ready: map[string]time.Time{}, Updated: now,
	}
	f.HP, f.Mana = f.MaxHP(), f.MaxMana()
	return f
}

// Init makes sure runtime maps exist (used after loading from the DB).
func (f *Fighter) Init() {
	if f.Uses == nil {
		f.Uses = map[string]int{}
	}
	if f.Ready == nil {
		f.Ready = map[string]time.Time{}
	}
	if f.Inv == nil {
		f.Inv = map[string]int{}
	}
}

// ---------- derived values ----------

// Raw is base + allocated points (what the point caps apply to).
func (f *Fighter) Raw(s Stat) int { return BaseStat(f.Race, f.Female, s) + f.Alloc[s] }

// Stat is the effective value including equipment.
func (f *Fighter) Stat(s Stat) int  { return f.Raw(s) + f.Gear[s] }
func (f *Fighter) MaxHP() float64   { return float64(f.Stat(StatHP)) * HPPerPoint }
func (f *Fighter) MaxMana() float64 { return float64(f.Stat(StatMana)) * ManaPerPoint }
func (f *Fighter) HPFrac() float64  { return f.HP / f.MaxHP() }

func (f *Fighter) HasElement(e Element) bool {
	for _, x := range f.Elems {
		if x == e {
			return true
		}
	}
	return false
}

// CanUse reports whether the spell is of the player's element and unlocked.
func (f *Fighter) CanUse(a *Ability) bool { return f.HasElement(a.Element) && f.Level >= a.Unlock }

func (f *Fighter) SpellLevel(a *Ability) int { return SpellLevel(f.Uses[a.Name]) }

// Weakness returns the human second-element penalty (%) for a spell.
func (f *Fighter) Weakness(a *Ability) float64 {
	if f.Race == Human && len(f.Elems) == 2 && a.Element == f.Elems[1] {
		return HumanWeakness(f.SpellLevel(a))
	}
	return 0
}

// ControlPenalty is the % extra mana cost and % less spell damage caused by
// mana − control: below 5 nothing, from 5 every point is 2% (max 25 points).
func (f *Fighter) ControlPenalty() float64 {
	diff := f.Stat(StatMana) - f.Stat(StatControl)
	if diff < 5 {
		return 0
	}
	if diff > 25 {
		diff = 25
	}
	return float64(diff) * 2
}

func (f *Fighter) Dead(now time.Time) bool {
	return !f.DeadUntil.IsZero() && now.Before(f.DeadUntil)
}

// Tick brings the lazily-computed state up to date: revival, regeneration,
// heal-over-time ticks and expired effects.
func (f *Fighter) Tick(now time.Time) {
	f.Init()
	if f.Updated.IsZero() {
		f.Updated = now
	}
	if !f.DeadUntil.IsZero() {
		if now.Before(f.DeadUntil) {
			f.Updated = now
			return
		}
		f.DeadUntil = time.Time{}
		f.HP, f.Mana = f.MaxHP(), f.MaxMana()
		f.Updated = now
	}
	if dt := now.Sub(f.Updated).Seconds(); dt > 0 {
		rate := BaseRegenPerMin * (1 + SpeedRegenPerPoint*float64(f.Stat(StatSpeed))) / 60
		f.HP = math.Min(f.MaxHP(), f.HP+rate*dt)
		f.Mana = math.Min(f.MaxMana(), f.Mana+rate*dt)
		f.Updated = now
	}
	kept := f.Effects[:0]
	for _, e := range f.Effects {
		if e.Type == EffHoT {
			for e.TicksLeft > 0 && !now.Before(e.Next) {
				f.HP = math.Min(f.MaxHP(), f.HP+e.Value)
				e.TicksLeft--
				e.Next = e.Next.Add(e.Every)
			}
		}
		if e.alive(now) {
			kept = append(kept, e)
		}
	}
	f.Effects = kept
	// drop mana-spend history older than the elf window
	i := 0
	for i < len(f.manaLog) && now.Sub(f.manaLog[i].t) > 10*time.Second {
		i++
	}
	f.manaLog = f.manaLog[i:]
}

// Die resets the fighter into the dead state (buffs are cleared on death).
func (f *Fighter) Die(now time.Time) {
	// Death penalty (open in the design doc; default): lose 10% of the gold and
	// 25% of the XP of the current level. Never de-levels; nothing is given to the killer.
	f.LostGold = f.Gold * DeathGoldLossPct / 100
	f.LostXP = f.XP * DeathXPLossPct / 100
	f.Gold -= f.LostGold
	f.XP -= f.LostXP
	f.HP = 0
	f.DeadUntil = now.Add(DeathSeconds * time.Second)
	f.Effects = nil
	f.Stacks = 0
	f.Updated = now
}

// GainXP adds XP and returns how many levels were gained.
func (f *Fighter) GainXP(x int64) int {
	if f.Level >= MaxLevel {
		return 0
	}
	f.XP += x
	gained := 0
	for f.Level < MaxLevel && f.XP >= XPNeeded(f.Level) {
		f.XP -= XPNeeded(f.Level)
		f.Level++
		f.Points += PointsPerLevel
		gained++
	}
	if f.Level >= MaxLevel {
		f.XP = 0
	}
	return gained
}

// ---------- effects helpers ----------

func (f *Fighter) addEffect(e *Effect) {
	for i, old := range f.Effects {
		if old.Key == e.Key {
			f.Effects[i] = e
			return
		}
	}
	f.Effects = append(f.Effects, e)
}

// sum adds the values of live effects of a type matching the attack filter.
func (f *Fighter) sum(t EffType, phys bool, now time.Time) float64 {
	s := 0.0
	for _, e := range f.Effects {
		if e.Type == t && e.alive(now) && e.Filter.match(phys) {
			s += e.Value
		}
	}
	return s
}

// consume spends one charge of every matching charge-based effect.
func (f *Fighter) consume(t EffType, phys bool, now time.Time, onlyOnMiss bool) {
	for _, e := range f.Effects {
		if e.Type != t || !e.alive(now) || e.Timed || !e.Filter.match(phys) {
			continue
		}
		if onlyOnMiss && !e.OnMiss {
			continue
		}
		e.Charges--
	}
}

func (f *Fighter) find(t EffType, now time.Time) *Effect {
	for _, e := range f.Effects {
		if e.Type == t && e.alive(now) {
			return e
		}
	}
	return nil
}

func (f *Fighter) logMana(now time.Time, amt float64) {
	if amt > 0 {
		f.manaLog = append(f.manaLog, manaEntry{now, amt})
	}
}

func (f *Fighter) recentMana(now time.Time) float64 {
	s := 0.0
	for _, m := range f.manaLog {
		if now.Sub(m.t) <= 10*time.Second {
			s += m.amt
		}
	}
	return s
}

// RemainingCD returns the seconds left on an ability key (0 = ready).
func (f *Fighter) RemainingCD(key string, now time.Time) float64 {
	if t, ok := f.Ready[key]; ok && t.After(now) {
		return t.Sub(now).Seconds()
	}
	return 0
}

// ResetCost is the gold price of respeccing: the first reset is free.
func (f *Fighter) ResetCost() int64 {
	if f.Resets == 0 {
		return 0
	}
	return int64(100 * f.Level)
}

// SpentPoints is the number of stat points currently allocated.
func (f *Fighter) SpentPoints() (n int) {
	for _, a := range f.Alloc {
		n += a
	}
	return
}

// ResetStats refunds every allocated point (equipment is unaffected).
func (f *Fighter) ResetStats() error {
	if f.SpentPoints() == 0 {
		return ErrNothingToReset
	}
	if f.Gold < f.ResetCost() {
		return ErrNoGold
	}
	f.Gold -= f.ResetCost()
	f.Points += f.SpentPoints()
	f.Alloc = [NumStats]int{}
	f.Resets++
	f.HP = math.Min(f.HP, f.MaxHP())
	f.Mana = math.Min(f.Mana, f.MaxMana())
	return nil
}
