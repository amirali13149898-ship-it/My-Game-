package bot

import (
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"ardian/internal/game"
	"ardian/internal/tg"
)

func newTestBot() *Bot {
	return &Bot{now: time.Now, cache: map[int64]*game.Fighter{}, drafts: map[int64]*[game.NumStats]int{},
		lastBtn: map[int64]time.Time{}, lastHunt: map[int64]time.Time{}, awaiting: map[int64]string{}, assets: map[string]asset{}, dungeons: map[int64]*game.DungeonRun{}, lastDungeon: map[int64]time.Time{}, eng: game.NewEngine()}
}

func balanced(s string) bool {
	for _, tag := range []string{"b", "i", "code", "pre"} {
		if strings.Count(s, "<"+tag+">") != strings.Count(s, "</"+tag+">") {
			return false
		}
	}
	return true
}

func TestViewsFitTelegramLimits(t *testing.T) {
	b := newTestBot()
	for r := game.Race(0); r < game.NumRaces; r++ {
		els := []game.Element{game.Fire}
		if r == game.Human {
			els = []game.Element{game.Fire, game.Wind}
		}
		f := game.NewFighter(int64(r)+1, "<Test & Name>", r, r%2 == 1, els, time.Now())
		f.Level = 100
		f.Points = 40
		views := map[string]view{"main": b.viewMain(f), "stats": b.viewStats(f), "race": b.viewAbilities(f, "r")}
		for i := range els {
			views["abil"+string(rune('0'+i))] = b.viewAbilities(f, string(rune('0'+i)))
		}
		for name, v := range views {
			if n := len(utf16.Encode([]rune(v.text))); n > 4096 || n == 0 {
				t.Errorf("%v/%s: %d UTF-16 units", r, name, n)
			}
			if !balanced(v.text) {
				t.Errorf("%v/%s: unbalanced HTML", r, name)
			}
			for _, row := range v.kb {
				for _, btn := range row {
					if len(btn.Data) > 64 {
						t.Errorf("callback data too long: %q", btn.Data)
					}
				}
			}
		}
	}
}

func TestStatDraft(t *testing.T) {
	b := newTestBot()
	f := game.NewFighter(7, "x", game.Human, false, []game.Element{game.Fire, game.Water}, time.Now())
	b.cache[7] = f
	u := &tg.User{ID: 7}
	b.cbStats(u, []string{"+", "2", "10"}) // only 5 points available
	if d := b.draft(7); d[game.StatPower] != 5 {
		t.Fatalf("draft %v", d)
	}
	if v := b.cbStats(u, []string{"+", "3", "1"}); v.toast == "" {
		t.Fatal("expected 'no points' toast")
	}
	b.cbStats(u, []string{"-", "2"})
	b.cbStats(u, []string{"0", "2"})
	if draftUsed(b.draft(7)) != 0 {
		t.Fatal("zero button failed")
	}
	// cap: control max is 125, others 100
	f.Alloc[game.StatLuck] = 95 // base 3 → 98
	f.Points = 20
	b.cbStats(u, []string{"+", "7", "10"})
	if f.Stat(game.StatLuck)+b.draft(7)[game.StatLuck] != 100 {
		t.Fatalf("luck cap broken: %v", b.draft(7))
	}
}

func TestAttackText(t *testing.T) {
	e := game.NewEngine()
	a := game.NewFighter(1, "A", game.Dwarf, false, []game.Element{game.Fire}, time.Now())
	d := game.NewFighter(2, "B", game.Human, false, []game.Element{game.Wind, game.Earth}, time.Now())
	res := e.Attack(a, d, game.AbilityByName("فایربال"), time.Now())
	s := attackText(a, d, res)
	if res.Fail != game.FailNone || !strings.Contains(s, "فایربال") || !balanced(s) {
		t.Fatalf("%+v\n%s", res, s)
	}
	t.Log("\n" + s)
}

func fits(t *testing.T, name string, v view) {
	t.Helper()
	if n := len(utf16.Encode([]rune(v.text))); n == 0 || n > 4096 {
		t.Errorf("%s: %d UTF-16 units", name, n)
	}
	if !balanced(v.text) {
		t.Errorf("%s: unbalanced HTML", name)
	}
	for _, row := range v.kb {
		for _, b := range row {
			if len(b.Data) > 64 {
				t.Errorf("%s: callback data too long %q", name, b.Data)
			}
		}
	}
}

func TestGearAndHuntViews(t *testing.T) {
	b := newTestBot()
	f := game.NewFighter(1, "<hero>", game.Human, false, []game.Element{game.Fire, game.Earth}, time.Now())
	f.Level = 100
	f.Gold = 1_000_000
	// a huge bag of weapons: every tier and rarity
	for tr := 1; tr <= game.NumTiers; tr++ {
		for r := game.Rarity(0); r < game.NumRarities; r++ {
			f.AddItem(game.ItemID(6, tr, r), 3)
		}
	}
	if err := f.Equip(6, game.ItemID(6, 10, game.Legendary)); err != nil {
		t.Fatal(err)
	}
	fits(t, "shop", b.viewShop(f))
	for k := 0; k < game.NumKinds; k++ {
		fits(t, "shopkind", b.viewShopKind(f, k))
	}
	fits(t, "equip", b.viewEquip(f))
	for s := 0; s < game.NumSlots; s++ {
		fits(t, "slot", b.viewSlot(f, s))
	}
	fits(t, "hunt", b.viewHunt(f))
	hr := game.NewEngine().Hunt(f, game.MonsterOptions(100)[0], time.Now())
	fits(t, "huntresult", view{text: huntText(f, hr)})
	if !strings.Contains(b.viewEquip(f).text, "سلاح اصلی") {
		t.Fatal("equip view missing slot")
	}
}

func TestDungeonStatsResetAndPhotoViews(t *testing.T) {
	b := newTestBot()
	f := game.NewFighter(5, "x", game.Dwarf, false, []game.Element{game.Fire}, time.Now())
	f.Level = 40
	f.Alloc[game.StatPower] = 30
	f.Gold = 5000
	b.cache[5] = f
	fits(t, "dungeon-entry", b.viewDungeon(f))
	b.dungeons[5] = game.NewDungeonRun(f.Level)
	fits(t, "dungeon-run", b.viewDungeon(f))
	delete(b.dungeons, 5)

	// reset flow (no DB needed for the confirm screen)
	u := &tg.User{ID: 5}
	if v := b.cbStats(u, []string{"rs"}); !strings.Contains(v.text, "ریست") {
		t.Fatalf("no confirm screen: %+v", v)
	}
	st := b.viewStats(f)
	found := false
	for _, row := range st.kb {
		for _, btn := range row {
			if btn.Data == "st:rs" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("reset button missing")
	}
	fits(t, "stats", st)

	// caption limit helper
	if !captionFits(strings.Repeat("a", 1024)) || captionFits(strings.Repeat("a", 1025)) {
		t.Fatal("captionFits wrong")
	}
	for r := 0; r < 9; r++ {
		if !validPhotoKey("race" + string(rune('0'+r))) {
			t.Fatalf("race%d key invalid", r)
		}
	}
}

func TestPenaltyText(t *testing.T) {
	if penaltyText(&game.Result{}) != "" {
		t.Fatal("empty penalty must print nothing")
	}
	if !strings.Contains(penaltyText(&game.Result{LostGold: 100, LostXP: 20}), "۱۰۰") {
		t.Fatal("penalty text")
	}
}
