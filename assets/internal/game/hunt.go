package game

import (
	"math"
	"time"
)

// Hunting is a simple PvE mode that exists mainly as the source of items
// ("هیولا/دراپ" is still open in the design doc; this is a default that can be
// replaced). The fight is simulated second by second on clones of the
// fighters, using the very same combat engine as PvP.

const HuntMaxSeconds = 240

type MonsterSpec struct {
	Tier  int // 1..10
	Index int // 0 brute, 1 mage, 2 rogue
	Name  string
	Level int
	Scale float64 // stat multiplier (0 = 1); bosses and deeper dungeon floors use > 1
	Boss  bool
}

var monsterNames = [NumTiers][3]string{
	{"اسلایم سبز", "گابلین جنگلی", "موش غارنشین"},
	{"گرگ وحشی", "جادوگر خرابه", "راهزن جاده"},
	{"اسکلت نگهبان", "شمن گابلین", "سایهٔ شبگرد"},
	{"ترول کوهستان", "ساحرهٔ مرداب", "دزد دریایی"},
	{"گولم سنگی", "جن آتشین", "آدمکش نقاب‌دار"},
	{"مینوتور", "شیطان یخ", "شکارچی ارواح"},
	{"غول طوفان", "لیچ تاریک", "تیغ‌زن سایه"},
	{"هیدرا", "فرمانروای ارواح", "شبح مرگ"},
	{"اژدهای جوان", "ارکان‌مایج", "جنگجوی بی‌صورت"},
	{"اژدهای باستان", "خدای تاریکی", "پادشاه سایه‌ها"},
}

var ArchetypeNames = [3]string{"🗡 جنگجو", "🔮 جادوگر", "🏹 چابک"}

// weights over HP, Mana, Power, Control, Speed, Defense, Perception, Luck, Crit
var archetypeWeights = [3][NumStats]float64{
	{3, 1, 4, 1, 1, 3, 1, 0, 0},
	{2, 4, 1, 4, 1, 1, 1, 0, 0},
	{1, 1, 3, 1, 4, 1, 3, 2, 2},
}

// MonsterOptions returns the three monsters offered to a player of a level
// (tier below, same tier, tier above).
func MonsterOptions(level int) []MonsterSpec {
	pt := (level-1)/10 + 1
	var out []MonsterSpec
	for _, dt := range []int{-1, 0, 1} {
		t := pt + dt
		if t < 1 || t > NumTiers {
			continue
		}
		for i := 0; i < 3; i++ {
			out = append(out, MonsterSpec{Tier: t, Index: i, Name: monsterNames[t-1][i], Level: max(1, t*10-5)})
		}
	}
	// keep the list short: one archetype per tier, rotating with the level
	var short []MonsterSpec
	for n, s := range out {
		if n%3 == (level/3)%3 {
			short = append(short, s)
		}
	}
	return short
}

func (m MonsterSpec) Element() Element { return Element((m.Tier + m.Index) % NumElements) }

// XPReward / GoldReward of a won hunt before luck.
func (m MonsterSpec) XPReward() int64   { return int64(30 + 5*m.Level) }
func (m MonsterSpec) GoldReward() int64 { return int64(40 + 10*m.Level) }

func (m MonsterSpec) Build(now time.Time) *Fighter {
	f := NewFighter(-int64(m.Tier*10+m.Index+1), m.Name, Human, false, []Element{m.Element()}, now)
	f.Level = m.Level
	// early monsters are weaker than a player of their level, late ones stronger
	pts := float64(PointsPerLevel*m.Level) * (0.2 + 0.12*float64(m.Tier))
	if m.Scale > 0 {
		pts *= m.Scale
	}
	var sum float64
	w := archetypeWeights[m.Index]
	for _, x := range w {
		sum += x
	}
	for s := Stat(0); s < NumStats; s++ {
		room := StatMax(s) - f.Raw(s)
		f.Alloc[s] = min(room, int(math.Round(pts*w[s]/sum)))
	}
	f.Points = 0
	spellLvl := min(10, 1+m.Level/10)
	for _, ab := range ElementAbilities(m.Element()) {
		f.Uses[ab.Name] = 5 * spellLvl * (spellLvl - 1) / 2
	}
	f.HP, f.Mana = f.MaxHP(), f.MaxMana()
	return f
}

type HuntResult struct {
	Fail        Fail
	Spec        MonsterSpec
	Win, Died   bool
	Fled        bool // nobody won within the time limit
	Seconds     int
	Dealt       int
	Taken       int
	XP, Gold    int64
	Levels      int
	Drop        string
	UsesChanged []string
	LostGold    int64
	LostXP      int64
}

func physExpected(f *Fighter) float64 { return float64(f.Stat(StatPower))*PhysPerPower + f.WeaponDmg }

func spellExpected(f *Fighter, ab *Ability) float64 {
	lvl := f.SpellLevel(ab)
	c := ab.Dmg + ab.Inc*float64(lvl-1) + physExpected(f)*MagicPowerShare
	return c * (1 - f.Weakness(ab)/100) * (1 - f.ControlPenalty()/100)
}

// pickAction chooses the offensive action with the highest expected damage
// that is currently usable (nil ability = physical attack).
func (e *Engine) pickAction(f *Fighter, now time.Time) (*Ability, bool) {
	f.Tick(now)
	if f.GlobalReady.After(now) {
		return nil, false
	}
	var best *Ability
	bestD, ok := -1.0, false
	if !f.PhysReady.After(now) {
		bestD, ok = physExpected(f), true
	}
	for _, el := range f.Elems {
		for _, ab := range ElementAbilities(el) {
			if ab.Kind != Offensive || !f.CanUse(ab) || f.RemainingCD(ab.NormName, now) > 0 || f.Mana < e.manaCost(f, ab, now) {
				continue
			}
			if d := spellExpected(f, ab); d > bestD {
				best, bestD, ok = ab, d, true
			}
		}
	}
	return best, ok
}

// simulate fights the monster on clones. The player keeps HP/mana, spell-use
// progress and (when dead) the death penalty; buffs and cooldowns are discarded.
// It pays no rewards.
func (e *Engine) simulate(p *Fighter, spec MonsterSpec, now time.Time) *HuntResult {
	hr := &HuntResult{Spec: spec}
	p.Tick(now)
	if p.Dead(now) {
		hr.Fail = FailSelfDead
		return hr
	}
	sim := &Engine{R: e.R, NoRewards: true}
	pc := p.Clone()
	pc.Effects = nil
	m := spec.Build(now)

	for t := 0; t <= HuntMaxSeconds; t++ {
		vt := now.Add(time.Duration(t) * time.Second)
		hr.Seconds = t
		if ab, ok := sim.pickAction(pc, vt); ok {
			hr.Dealt += sim.Attack(pc, m, ab, vt).Damage
			if m.Dead(vt) {
				hr.Win = true
				break
			}
		}
		if ab, ok := sim.pickAction(m, vt); ok {
			hr.Taken += sim.Attack(m, pc, ab, vt).Damage
			if pc.Dead(vt) {
				hr.Died = true
				break
			}
		}
	}
	hr.Fled = !hr.Win && !hr.Died

	for name, n := range pc.Uses {
		if p.Uses[name] != n {
			hr.UsesChanged = append(hr.UsesChanged, name)
		}
	}
	p.Uses = pc.Uses
	p.Mana, p.HP, p.Updated = pc.Mana, pc.HP, now
	if hr.Died {
		p.Die(now)
		hr.LostGold, hr.LostXP = p.LostGold, p.LostXP
	}
	return hr
}

func angelBoost(p *Fighter) [3]float64 {
	boost := [3]float64{1, 1, 1}
	if p.Race == Angel && p.RaceTier >= 1 {
		boost[0], boost[1], boost[2] = AngelBoosts(min(p.Level/5+1, p.RaceTier))
	}
	return boost
}

func luckXP(p *Fighter, xp int64) int64 {
	return int64(math.Round(float64(xp) * (1 + 0.005*float64(p.Stat(StatLuck)))))
}

// Hunt simulates a fight and pays the rewards of a win.
func (e *Engine) Hunt(p *Fighter, spec MonsterSpec, now time.Time) *HuntResult {
	hr := e.simulate(p, spec, now)
	if hr.Win {
		hr.XP = luckXP(p, spec.XPReward())
		hr.Gold = spec.GoldReward()
		p.Gold += hr.Gold
		hr.Levels = p.GainXP(hr.XP)
		if id := RollDrop(DropTable(spec.Tier, spec.Index), p.Stat(StatLuck), angelBoost(p), e.R); id != "" {
			hr.Drop = id
			p.AddItem(id, 1)
		}
	}
	return hr
}
