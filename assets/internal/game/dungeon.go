package game

import "time"

// Dungeon: a solo run through 5 floors and a boss. HP and mana carry over
// from floor to floor; after every floor the player may leave with the loot
// collected so far or continue. Dying loses the pending loot (and costs the
// normal death penalty). The design doc only says "anime dungeon, undecided",
// so everything here is a default.

const (
	DungeonFloors   = 5 // plus the boss floor
	DungeonMinLevel = 10
)

var dungeonNames = [NumTiers]string{
	"غار گابلین‌ها", "جنگل نجواگر", "خرابهٔ فراموش‌شده", "معدن ترول‌ها", "مرداب جادوگران",
	"قلعهٔ یخی", "برج طوفان", "دخمهٔ لیچ", "آشیانهٔ اژدها", "تالار خدایان تاریکی",
}
var bossNames = [NumTiers]string{
	"پادشاه گابلین", "روح جنگل", "فرمانروای خرابه", "شاه ترول‌ها", "ملکهٔ مرداب",
	"ارباب یخ", "سلطان طوفان", "لیچ بزرگ", "مادر اژدها", "امپراتور تاریکی",
}

func DungeonName(tier int) string { return dungeonNames[clampTier(tier, NumTiers)-1] }
func DungeonTier(level int) int   { return clampTier((level-1)/10+1, NumTiers) }

type DungeonRun struct {
	Tier    int
	Cleared int // floors beaten (the boss is floor DungeonFloors+1)
	XP      int64
	Gold    int64
	Drops   []string
}

func NewDungeonRun(level int) *DungeonRun { return &DungeonRun{Tier: DungeonTier(level)} }

func (r *DungeonRun) Total() int { return DungeonFloors + 1 }

// NextSpec is the monster of the next floor.
func (r *DungeonRun) NextSpec() MonsterSpec {
	f := r.Cleared // 0-based floor index
	if f >= DungeonFloors {
		return MonsterSpec{Tier: r.Tier, Index: 0, Name: bossNames[r.Tier-1], Level: r.Tier * 10, Scale: 1.6, Boss: true}
	}
	i := f % 3
	return MonsterSpec{Tier: r.Tier, Index: i, Name: monsterNames[r.Tier-1][i],
		Level: max(1, r.Tier*10-5+f*2), Scale: 1 + 0.12*float64(f)}
}

type FloorResult struct {
	Fail     Fail
	Floor    int // 1-based number of the fought floor
	Hunt     *HuntResult
	Boss     bool
	Finished bool         // run ended (boss beaten, lost, or timed out)
	Loot     *DungeonLoot // set when the run ended with a payout
}

type DungeonLoot struct {
	XP, Gold int64
	Levels   int
	Drops    []string
}

// DungeonFloor fights the next floor of the run.
func (e *Engine) DungeonFloor(p *Fighter, run *DungeonRun, now time.Time) *FloorResult {
	spec := run.NextSpec()
	hr := e.simulate(p, spec, now)
	fr := &FloorResult{Fail: hr.Fail, Floor: run.Cleared + 1, Hunt: hr, Boss: spec.Boss}
	if hr.Fail != FailNone {
		return fr
	}
	switch {
	case hr.Win:
		mult := 1 + 0.25*float64(run.Cleared)
		if spec.Boss {
			mult = 4
		}
		run.XP += int64(float64(spec.XPReward()) * mult)
		run.Gold += int64(float64(spec.GoldReward()) * mult)
		rolls := 1
		if spec.Boss {
			rolls = 3
		}
		for i := 0; i < rolls; i++ {
			if id := RollDrop(DropTable(spec.Tier, spec.Index), p.Stat(StatLuck), angelBoost(p), e.R); id != "" {
				run.Drops = append(run.Drops, id)
			}
		}
		run.Cleared++
		if spec.Boss {
			fr.Finished = true
			fr.Loot = e.DungeonClaim(p, run)
		}
	case hr.Died:
		fr.Finished = true // pending loot is lost
	default: // nobody won in time: retreat with what we have
		fr.Finished = true
		fr.Loot = e.DungeonClaim(p, run)
	}
	return fr
}

// DungeonClaim pays the pending loot of the run.
func (e *Engine) DungeonClaim(p *Fighter, run *DungeonRun) *DungeonLoot {
	l := &DungeonLoot{XP: luckXP(p, run.XP), Gold: run.Gold, Drops: run.Drops}
	p.Gold += l.Gold
	l.Levels = p.GainXP(l.XP)
	for _, id := range l.Drops {
		p.AddItem(id, 1)
	}
	run.XP, run.Gold, run.Drops = 0, 0, nil
	return l
}
