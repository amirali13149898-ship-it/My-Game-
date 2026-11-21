package game

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// duel fights two fighters with the PvP engine using the same greedy action
// choice as the PvE simulator. Racial actives are not used (passives are).
func duel(e *Engine, a, b *Fighter, now time.Time) (winner int) {
	for t := 0; t <= 600; t++ {
		vt := now.Add(time.Duration(t) * time.Second)
		if ab, ok := e.pickAction(a, vt); ok {
			e.Attack(a, b, ab, vt)
			if b.Dead(vt) {
				return 1
			}
		}
		if ab, ok := e.pickAction(b, vt); ok {
			e.Attack(b, a, ab, vt)
			if a.Dead(vt) {
				return 2
			}
		}
	}
	return 0
}

// build creates a level-lvl character that spends all points by a profile.
func build(r Race, lvl int, prof string, el Element) *Fighter {
	f := NewFighter(1, "x", r, false, []Element{el}, t0)
	f.Level = lvl
	pts := PointsPerLevel * lvl
	w := map[string][NumStats]float64{
		"phys":  {3, 1, 4, 1, 1, 3, 1, 1, 1},
		"mage":  {2, 4, 1, 4, 1, 1, 1, 1, 1},
		"mixed": {2, 2, 2, 2, 1, 2, 1, 1, 1},
	}[prof]
	var sum float64
	for _, x := range w {
		sum += x
	}
	for s := Stat(0); s < NumStats; s++ {
		f.Alloc[s] = min(StatMax(s)-f.Raw(s), int(float64(pts)*w[s]/sum))
	}
	for _, ab := range ElementAbilities(el) {
		f.Uses[ab.Name] = 225
	}
	f.RaceTier = min(RaceMaxTier(r), (lvl-1)/10+1)
	f.HP, f.Mana = f.MaxHP(), f.MaxMana()
	return f
}

// TestRaceMatrix prints race and profile win rates at several levels. It is a
// balance dashboard (go test -run RaceMatrix -v), not a pass/fail gate.
func TestRaceMatrix(t *testing.T) {
	if os.Getenv("BALANCE") == "" {
		t.Skip("set BALANCE=1 to run")
	}
	e := NewEngine()
	for _, lvl := range []int{20, 60, 100} {
		var sb strings.Builder
		fmt.Fprintf(&sb, "\nlevel %d — race win rate vs all others (same profile mix)\n", lvl)
		for r := Race(0); r < NumRaces; r++ {
			wins, n := 0, 0
			for o := Race(0); o < NumRaces; o++ {
				if o == r {
					continue
				}
				for _, prof := range []string{"phys", "mage", "mixed"} {
					for i := 0; i < 20; i++ {
						a, b := build(r, lvl, prof, Fire), build(o, lvl, prof, Water)
						if r == Human {
							a.Elems = []Element{Fire, Earth}
						}
						if duel(e, a, b, t0) == 1 {
							wins++
						}
						n++
					}
				}
			}
			fmt.Fprintf(&sb, "  %-10s %3d%%\n", RaceNames[r], wins*100/n)
		}
		for _, pair := range [][2]string{{"phys", "mage"}, {"phys", "mixed"}, {"mage", "mixed"}} {
			wins, n := 0, 0
			for r := Race(0); r < NumRaces; r++ {
				for i := 0; i < 20; i++ {
					if duel(e, build(r, lvl, pair[0], Fire), build(r, lvl, pair[1], Water), t0) == 1 {
						wins++
					}
					n++
				}
			}
			fmt.Fprintf(&sb, "  %-5s vs %-5s: first wins %3d%%\n", pair[0], pair[1], wins*100/n)
		}
		t.Log(sb.String())
	}
}
