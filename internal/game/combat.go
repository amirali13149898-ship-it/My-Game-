package game

import (
	"math"
	"math/rand/v2"
	"time"
)

// Rand is the randomness source (replaced by a fixed one in tests).
type Rand interface{ Float64() float64 }

type globalRand struct{}

func (globalRand) Float64() float64 { return rand.Float64() }

type Engine struct {
	R         Rand
	NoRewards bool // PvE simulations: no XP/gold from hits
}

func NewEngine() *Engine { return &Engine{R: globalRand{}} }

func (e *Engine) roll(pct float64) bool { return e.R.Float64()*100 < pct }

// Fail explains why an action was refused (nothing was spent).
type Fail int

const (
	FailNone Fail = iota
	FailSelfDead
	FailTargetDead
	FailCD
	FailMana
	FailLocked
	FailNotAvailable // race ability passive / not bought
)

// Result describes what an action did. It is plain data; the bot formats it.
type Result struct {
	Fail Fail
	Wait float64 // seconds (cooldown / death)
	Need float64 // mana needed

	Ab          *Ability // nil = physical attack
	Phys        bool
	RaceUse     bool
	ManaCost    int
	NewSpellLvl int // >0 when the spell reached a new upgrade level

	// attack
	Hit, Dodge, Crit bool
	Damage           int
	ElemMod          float64
	Reflected        int
	Killed           bool
	AttackerDied     bool
	LostGold, LostXP int64 // death penalty of the defender (or of the attacker if AttackerDied)
	XP, Gold         int64
	LevelsGained     int

	// support / race abilities
	Effect       *Effect
	Healed       float64
	ManaRestored float64
	Dodges       int // fox: number of free dodges
	Chance       float64
}

func (e *Engine) manaCost(f *Fighter, ab *Ability, now time.Time) float64 {
	c := float64(ab.Mana) * (1 + f.ControlPenalty()/100)
	red := math.Min(f.sum(EffManaCost, false, now), 90)
	return c * (1 - red/100)
}

func (e *Engine) spellCD(f *Fighter, ab *Ability, now time.Time) float64 {
	red := math.Min(f.sum(EffCDRed, false, now), 50)
	return ab.Cooldown * (1 - red/100)
}

// checkCommon validates liveness, cooldown, lock and mana for a spell.
func (e *Engine) checkSpell(f *Fighter, ab *Ability, now time.Time, r *Result, withGlobal bool) bool {
	if f.Dead(now) {
		r.Fail, r.Wait = FailSelfDead, f.DeadUntil.Sub(now).Seconds()
		return false
	}
	if !f.CanUse(ab) {
		r.Fail = FailLocked
		return false
	}
	wait := f.RemainingCD(ab.NormName, now)
	if withGlobal {
		wait = math.Max(wait, f.GlobalReady.Sub(now).Seconds())
	}
	if wait > 0 {
		r.Fail, r.Wait = FailCD, wait
		return false
	}
	if cost := e.manaCost(f, ab, now); f.Mana+1e-9 < cost {
		r.Fail, r.Need = FailMana, cost
		return false
	}
	return true
}

// spendSpell pays mana, starts the cooldown and counts the use.
func (e *Engine) spendSpell(f *Fighter, ab *Ability, now time.Time, r *Result) {
	cost := e.manaCost(f, ab, now)
	cd := e.spellCD(f, ab, now)
	f.Mana -= cost
	f.logMana(now, cost)
	r.ManaCost = int(math.Round(cost))
	f.consume(EffManaCost, false, now, false)
	f.consume(EffCDRed, false, now, false)
	f.Ready[ab.NormName] = now.Add(time.Duration(cd * float64(time.Second)))
	before := f.SpellLevel(ab)
	f.Uses[ab.Name]++
	if after := f.SpellLevel(ab); after > before {
		r.NewSpellLvl = after
	}
}

func hitScore(f *Fighter) float64 {
	return float64(f.Stat(StatSpeed))*0.2 + float64(f.Stat(StatPerception))*0.6 + float64(f.Stat(StatLuck))*0.2
}

// elemMod: average over the defender's elements (humans have two).
func elemMod(att Element, def *Fighter) float64 {
	if len(def.Elems) == 0 {
		return 0
	}
	s := 0.0
	for _, d := range def.Elems {
		s += ElementMod(att, d)
	}
	return s / float64(len(def.Elems))
}

// raceDealtBonus is the % damage bonus coming from racial passives.
func raceDealtBonus(f *Fighter, now time.Time) (bonus float64, usedRastsu bool) {
	switch f.Race {
	case Tiger:
		if now.Before(f.RastsuReady) {
			return 0, false
		}
		mid, low := TigerVals(f.RaceTier)
		switch frac := f.HPFrac(); {
		case frac < 0.25:
			return low, true
		case frac <= 0.5:
			return mid, true
		}
	case Dragon:
		if f.HPFrac() <= 0.2 {
			d, _ := DragonVals(f.RaceTier)
			return d, false
		}
	case Wolf:
		return float64(f.Stacks) * WolfPerStack(f.RaceTier), false
	}
	return 0, false
}

// Attack performs a physical attack (ab == nil) or an offensive spell.
func (e *Engine) Attack(att, def *Fighter, ab *Ability, now time.Time) *Result {
	att.Tick(now)
	def.Tick(now)
	r := &Result{Ab: ab, Phys: ab == nil}
	if ab != nil && ab.Kind != Offensive {
		r.Fail = FailLocked
		return r
	}
	if att.Dead(now) {
		r.Fail, r.Wait = FailSelfDead, att.DeadUntil.Sub(now).Seconds()
		return r
	}
	if def.Dead(now) {
		r.Fail = FailTargetDead
		return r
	}
	if ab == nil {
		if wait := math.Max(att.GlobalReady.Sub(now).Seconds(), att.PhysReady.Sub(now).Seconds()); wait > 0 {
			r.Fail, r.Wait = FailCD, wait
			return r
		}
	} else if !e.checkSpell(att, ab, now, r, true) {
		return r
	}

	// ---- pay ----
	lvl, weak, pen := 1, 0.0, 0.0
	if ab != nil {
		lvl, weak, pen = att.SpellLevel(ab), att.Weakness(ab), att.ControlPenalty()
		e.spendSpell(att, ab, now, r)
	} else {
		att.PhysReady = now.Add(PhysCDSec * time.Second)
	}
	att.GlobalReady = now.Add(GlobalCDSec * time.Second)

	// ---- 1. base damage ----
	var dmg float64
	power := float64(att.Stat(StatPower))*PhysPerPower + att.WeaponDmg // weapon: flat damage
	if ab == nil {
		dmg = power * (1 + (e.R.Float64()*2-1)*PhysSpread)
	} else {
		center := ab.Dmg + ab.Inc*float64(lvl-1)
		if !ab.Fixed {
			center += (e.R.Float64()*2 - 1) * float64(lvl)
		}
		center += power * MagicPowerShare
		dmg = center * (1 - weak/100) * (1 - pen/100)
	}

	// ---- 2. buffs and racial passives ----
	bonus := att.sum(EffOutDmg, r.Phys, now)
	rb, usedRastsu := raceDealtBonus(att, now)
	bonus += rb
	if usedRastsu {
		att.RastsuReady = now.Add(8 * time.Second)
	}
	dmg *= 1 + bonus/100

	// ---- 3. element cycle (magic only) ----
	if ab != nil {
		r.ElemMod = elemMod(ab.Element, def)
		dmg *= 1 + r.ElemMod/100
	}

	// ---- 4. dodge ----
	if k := def.find(EffKyubi, now); k != nil {
		k.Charges--
		r.Dodge = true
	} else {
		dodge := math.Max(0, hitScore(def)-hitScore(att)) +
			def.sum(EffInDodge, r.Phys, now) - att.sum(EffOutAcc, r.Phys, now)
		dodge = math.Max(0, math.Min(dodge, 95))
		r.Dodge = e.roll(dodge)
	}
	r.Hit = !r.Dodge

	// ---- 5. critical ----
	critMult := 1.0
	natural, forced := false, false
	if r.Hit {
		chance := math.Min(10, float64(att.Stat(StatCrit))*CritPerPoint) + att.sum(EffOutCrit, r.Phys, now)
		natural = e.roll(chance)
	}
	if z := att.find(EffZetsu, now); z != nil && r.Phys {
		forced = r.Hit && e.roll(z.Value)
		z.Charges-- // consumed by the attack even if dodged
	}
	switch {
	case natural:
		critMult = CritMult // a natural crit keeps the full multiplier
	case forced:
		critMult = ZetsuMult
	}
	r.Crit = natural || forced
	dmg *= critMult

	// ---- 6. defense ----
	reflectPct := 0.0
	if r.Hit {
		red := float64(def.Stat(StatDefense))*DefensePerPoint + def.sum(EffInDmgRed, r.Phys, now)
		dmg *= 1 - math.Min(red, MaxDefenseRed)/100
		if r.Crit {
			dmg *= 1 - math.Min(def.sum(EffInCrit, r.Phys, now), MaxDefenseRed)/100
		}
		if def.Race == Dragon && def.HPFrac() <= 0.2 {
			_, taken := DragonVals(def.RaceTier)
			dmg *= 1 + taken/100
		}
		reflectPct = def.sum(EffReflect, r.Phys, now)
		r.Damage = int(math.Round(dmg))
		if r.Damage < 1 {
			r.Damage = 1
		}
	}

	// ---- charges ----
	att.consume(EffOutDmg, r.Phys, now, !r.Hit)
	att.consume(EffOutCrit, r.Phys, now, !r.Hit)
	att.consume(EffOutAcc, r.Phys, now, !r.Hit)
	def.consume(EffInDodge, r.Phys, now, false)
	if r.Hit {
		def.consume(EffInDmgRed, r.Phys, now, false)
		def.consume(EffInCrit, r.Phys, now, false)
		def.consume(EffReflect, r.Phys, now, false)
	}

	if !r.Hit {
		return r
	}

	// ---- apply ----
	def.HP -= float64(r.Damage)
	if def.HP <= 0 {
		def.Die(now)
		r.Killed = true
		r.LostGold, r.LostXP = def.LostGold, def.LostXP
	}
	if att.Race == Wolf && att.Stacks < WolfMaxStacks {
		att.Stacks++
	}
	if reflectPct > 0 {
		r.Reflected = int(math.Round(float64(r.Damage) * reflectPct / 100))
		if r.Reflected > 0 {
			att.HP -= float64(r.Reflected)
			if att.HP <= 0 {
				att.Die(now)
				r.AttackerDied = true
				r.LostGold, r.LostXP = att.LostGold, att.LostXP
			}
		}
	}

	// ---- rewards ----
	if !r.AttackerDied && !e.NoRewards {
		xp, gold := int64(XPHit), int64(GoldHit)
		if r.Killed {
			xp, gold = XPKill, GoldKill
		}
		xp = int64(math.Round(float64(xp) * (1 + 0.005*float64(att.Stat(StatLuck)))))
		att.Gold += gold
		r.XP, r.Gold = xp, gold
		r.LevelsGained = att.GainXP(xp)
	}
	return r
}

// Cast uses a support/defensive spell on the caster.
func (e *Engine) Cast(f *Fighter, ab *Ability, now time.Time) *Result {
	f.Tick(now)
	r := &Result{Ab: ab}
	if !e.checkSpell(f, ab, now, r, false) {
		return r
	}
	lvl, weak := f.SpellLevel(ab), f.Weakness(ab)
	e.spendSpell(f, ab, now, r)

	s := ab.Eff
	val := (s.Base + s.Inc*float64(lvl-1)) * (1 - weak/100)
	switch s.Type {
	case EffInstant:
		hp := val
		mana := (s.Mana + s.ManaInc*float64(lvl-1)) * (1 - weak/100)
		r.Healed = math.Min(hp, f.MaxHP()-f.HP)
		r.ManaRestored = math.Min(mana, f.MaxMana()-f.Mana)
		f.HP += r.Healed
		f.Mana += r.ManaRestored
	case EffCDAll:
		for k, t := range f.Ready {
			if k == ab.NormName || !t.After(now) {
				continue
			}
			rem := math.Round(t.Sub(now).Seconds()*(1-val/100)*100) / 100
			f.Ready[k] = now.Add(time.Duration(rem * float64(time.Second)))
		}
		r.Effect = &Effect{Name: ab.Name, Type: s.Type, Value: val}
	default:
		if s.Type == EffCDRed {
			val = math.Min(val, 50)
		}
		dur := s.Dur
		if dur == 0 {
			dur = DefaultEffDur
		}
		ef := &Effect{
			Key: ab.NormName, Name: ab.Name, Type: s.Type, Value: val, Filter: s.Filter,
			Charges: s.Charges, OnMiss: s.OnMiss, Timed: s.Timed,
			Expires: now.Add(time.Duration(dur * float64(time.Second))),
		}
		if s.Type == EffHoT {
			ef.Timed = true
			ef.Every = time.Duration(s.Every * float64(time.Second))
			ef.TicksLeft = s.Ticks
			ef.Next = now.Add(ef.Every)
		}
		f.addEffect(ef)
		r.Effect = ef
	}
	return r
}

// UseRace triggers the active ability of the player's race.
func (e *Engine) UseRace(f *Fighter, now time.Time) *Result {
	f.Tick(now)
	r := &Result{RaceUse: true}
	if f.Dead(now) {
		r.Fail, r.Wait = FailSelfDead, f.DeadUntil.Sub(now).Seconds()
		return r
	}
	if !RaceActive(f.Race) || f.RaceTier < 1 {
		r.Fail = FailNotAvailable
		return r
	}
	if w := f.RemainingCD(RaceKey, now); w > 0 {
		r.Fail, r.Wait = FailCD, w
		return r
	}
	dur := now.Add(DefaultEffDur * time.Second)
	var cd float64
	switch f.Race {
	case Elf:
		v := ElfTier(f.RaceTier)
		amt := math.Min(v.Cap, v.Fixed+v.Pct/100*f.recentMana(now))
		r.ManaRestored = math.Min(amt, f.MaxMana()-f.Mana)
		f.Mana += r.ManaRestored
		cd = ElfCooldown
	case DarkElf:
		eff := f.Level
		if m := f.RaceTier * 10; m < eff {
			eff = m
		}
		chance, c := DarkElfVals(eff)
		cd, r.Chance = c, chance
		r.Effect = &Effect{Key: RaceKey, Name: RaceAbilityName[DarkElf], Type: EffZetsu,
			Value: chance, Filter: FPhys, Charges: 1, Expires: dur}
		f.addEffect(r.Effect)
	case Dwarf:
		v := DwarfTier(f.RaceTier)
		cd = v.CD
		r.Effect = &Effect{Key: RaceKey, Name: RaceAbilityName[Dwarf], Type: EffReflect,
			Value: v.Reflect, Charges: v.Hits, Expires: dur}
		f.addEffect(r.Effect)
	case Fox:
		n, c := FoxTier(f.RaceTier)
		cd, r.Dodges = c, n
		r.Effect = &Effect{Key: RaceKey, Name: RaceAbilityName[Fox], Type: EffKyubi,
			Charges: n, Expires: dur}
		f.addEffect(r.Effect)
	}
	f.Ready[RaceKey] = now.Add(time.Duration(cd * float64(time.Second)))
	return r
}
