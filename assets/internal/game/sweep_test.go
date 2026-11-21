package game

import (
	"os"
	"testing"
)

func TestSweepShare(t *testing.T) {
	if os.Getenv("BALANCE") == "" {
		t.Skip()
	}
	e := NewEngine()
	old := MagicPowerShare
	defer func() { MagicPowerShare = old }()
	for _, share := range []float64{0.3, 0.6, 1.0, 1.5} {
		MagicPowerShare = share
		for _, lvl := range []int{20, 60, 100} {
			res := map[string]int{}
			for _, pair := range [][2]string{{"phys", "mage"}, {"phys", "mixed"}, {"mage", "mixed"}} {
				w, n := 0, 0
				for r := Race(0); r < NumRaces; r++ {
					for i := 0; i < 15; i++ {
						if duel(e, build(r, lvl, pair[0], Fire), build(r, lvl, pair[1], Water), t0) == 1 {
							w++
						}
						n++
					}
				}
				res[pair[0]+">"+pair[1]] = w * 100 / n
			}
			t.Logf("share %.1f lvl %3d  phys>mage %3d%%  phys>mixed %3d%%  mage>mixed %3d%%", share, lvl,
				res["phys>mage"], res["phys>mixed"], res["mage>mixed"])
		}
	}
}
