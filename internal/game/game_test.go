package game

import (
	"testing"
	"time"
)

type fixed float64

func (f fixed) Float64() float64 { return float64(f) }

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func mk(id int64, r Race, el ...Element) *Fighter {
	return NewFighter(id, "x", r, false, el, t0)
}

func TestSpellLevelCurve(t *testing.T) {
	cases := map[int]int{0: 1, 4: 1, 5: 2, 15: 3, 224: 9, 225: 10, 9999: 10}
	for uses, want := range cases {
		if got := SpellLevel(uses); got != want {
			t.Errorf("uses %d: got %d want %d", uses, got, want)
		}
	}
}

func TestElementCycle(t *testing.T) {
	if ElementMod(Fire, Wind) != 10 || ElementMod(Wind, Earth) != 10 ||
		ElementMod(Earth, Water) != 10 || ElementMod(Water, Fire) != 10 {
		t.Fatal("winning edges wrong")
	}
	if ElementMod(Wind, Fire) != -10 || ElementMod(Fire, Fire) != 0 || ElementMod(Fire, Earth) != 0 {
		t.Fatal("losing/neutral edges wrong")
	}
	h := mk(1, Human, Wind, Earth) // fire beats wind(+10), neutral on earth(0)
	if m := elemMod(Fire, h); m != 5 {
		t.Fatalf("human average: got %v want 5", m)
	}
}

func TestAbilityTables(t *testing.T) {
	for e := Element(0); e < NumElements; e++ {
		list := ElementAbilities(e)
		if len(list) != 20 {
			t.Fatalf("%v has %d abilities", e, len(list))
		}
		n := map[Kind]int{}
		for _, a := range list {
			n[a.Kind]++
		}
		if n[Offensive] != 10 || n[Support] != 5 || n[Defensive] != 5 {
			t.Fatalf("%v kinds %v", e, n)
		}
	}
	if AbilityByName("فایـربال ") == nil || AbilityByName("فایر‌بال") == nil {
		t.Fatal("normalization failed")
	}
}

func TestPhysicalDamagePipeline(t *testing.T) {
	// R=0.5 → no random spread, no crit/dodge rolls succeed (0.5*100=50 ≥ chances).
	e := &Engine{R: fixed(0.5)}
	a, d := mk(1, Dwarf, Fire), mk(2, Human, Water, Fire) // dwarf power 10
	r := e.Attack(a, d, nil, t0)
	if !r.Hit || r.Crit {
		t.Fatalf("unexpected %+v", r)
	}
	// 10*20=200, defense 5*0.5=2.5% → 195
	if r.Damage != 195 {
		t.Fatalf("damage %d want 195", r.Damage)
	}
	if r.XP != 26 || r.Gold != 50 || a.Gold != 50 { // 25 XP +0.5%/luck point (luck 5)
		t.Fatalf("rewards %+v", r)
	}
	// cooldowns: second attack immediately → refused
	if r2 := e.Attack(a, d, nil, t0.Add(time.Second)); r2.Fail != FailCD {
		t.Fatalf("expected cooldown, got %+v", r2)
	}
	// 3s later physical is still on its 5s cooldown, a spell is not
	fb := AbilityByName("فایربال")
	if r3 := e.Attack(a, d, nil, t0.Add(3*time.Second)); r3.Fail != FailCD {
		t.Fatal("physical must wait 5s")
	}
	if r3 := e.Attack(a, d, fb, t0.Add(3*time.Second)); r3.Fail != FailNone {
		t.Fatalf("spell should work: %+v", r3)
	}
}

func TestDodgeAlwaysHitsWhenNotFaster(t *testing.T) {
	e := &Engine{R: fixed(0)} // every roll "succeeds"
	a, d := mk(1, Elf, Fire), mk(2, Dwarf, Water)
	// elf speed/perception/luck is higher than dwarf → dwarf never dodges
	r := e.Attack(d, a, nil, t0)
	if r.Dodge {
		t.Fatal("slower defender dodged")
	}
}

func TestKillAndRevive(t *testing.T) {
	e := &Engine{R: fixed(0.5)}
	a, d := mk(1, Dragon, Fire), mk(2, Human, Water)
	d.HP = 10
	r := e.Attack(a, d, nil, t0)
	if !r.Killed || r.XP != 101 || r.Gold != 200 {
		t.Fatalf("kill result %+v", r)
	}
	if !d.Dead(t0.Add(299 * time.Second)) {
		t.Fatal("should still be dead")
	}
	d.Tick(t0.Add(301 * time.Second))
	if d.Dead(t0.Add(301*time.Second)) || d.HP != d.MaxHP() || d.Mana != d.MaxMana() {
		t.Fatal("should be revived with full hp/mana")
	}
	ghost := mk(9, Human, Fire)
	ghost.Die(t0)
	if r2 := e.Attack(ghost, a, nil, t0.Add(10*time.Second)); r2.Fail != FailSelfDead {
		t.Fatal("dead player attacked")
	}
}

func TestLevelUp(t *testing.T) {
	f := mk(1, Human, Fire)
	if n := f.GainXP(350); n != 2 || f.Level != 3 || f.XP != 50 || f.Points != 15 {
		t.Fatalf("level %d xp %d points %d (+%d)", f.Level, f.XP, f.Points, n)
	}
}

func TestBuffChargesAndDefense(t *testing.T) {
	e := &Engine{R: fixed(0.5)}
	a := mk(1, Human, Fire)
	a.Level = 15
	d := mk(2, Human, Fire)
	a.Mana = 1000
	raito := AbilityByName("فایر رایتو")
	if r := e.Cast(a, raito, t0); r.Fail != FailNone || r.Effect.Value != 10 {
		t.Fatalf("cast %+v", r)
	}
	r1 := e.Attack(a, d, nil, t0)
	r2 := e.Attack(a, d, nil, t0.Add(5*time.Second))
	r3 := e.Attack(a, d, nil, t0.Add(10*time.Second))
	if !(r1.Damage > r3.Damage && r2.Damage > r3.Damage) {
		t.Fatalf("2 charged attacks expected: %d %d %d", r1.Damage, r2.Damage, r3.Damage)
	}
}

func TestZetsuAndKyubi(t *testing.T) {
	e := &Engine{R: fixed(0.0)} // all rolls succeed
	de := NewFighter(1, "d", DarkElf, false, []Element{Fire}, t0)
	de.RaceTier = 10
	de.Level = 100
	tg := mk(2, Dwarf, Water)
	if r := e.UseRace(de, t0); r.Fail != FailNone || r.Chance != 100 {
		t.Fatalf("zetsu %+v", r)
	}
	if r := e.Attack(de, tg, nil, t0); !r.Crit {
		t.Fatalf("expected forced crit: %+v", r)
	}
	fox := mk(3, Fox, Fire)
	e.UseRace(fox, t0)
	if r := e.Attack(tg, fox, nil, t0); !r.Dodge || r.Hit {
		t.Fatal("kyubi must dodge")
	}
}

func TestRegenAndHoT(t *testing.T) {
	f := mk(1, Human, Fire)
	f.HP, f.Mana = 0.0+1, 0
	f.Tick(t0.Add(60 * time.Second)) // speed 4 → 50*1.12 per minute
	if f.HP < 56 || f.HP > 58 || f.Mana < 55 || f.Mana > 57 {
		t.Fatalf("regen hp %v mana %v", f.HP, f.Mana)
	}
}

func TestRaceTables(t *testing.T) {
	var total int64
	for k := 2; k <= 10; k++ {
		total += RaceTierPrice(Elf, k)
	}
	if total != 13500 {
		t.Fatalf("elf total %d", total)
	}
	total = 0
	for k := 1; k <= 10; k++ {
		total += RaceTierPrice(DarkElf, k)
	}
	if total != 27500 {
		t.Fatalf("dark elf total %d", total)
	}
	total = 0
	for n := 1; n <= 100; n++ {
		total += HumanPointPrice(n)
	}
	if total != 30000 {
		t.Fatalf("human total %d", total)
	}
	if c, cd := DarkElfVals(50); c != 50 || cd != 64 {
		t.Fatalf("zetsu lvl50: %v %v", c, cd)
	}
	if r, s, th := AngelBoosts(20); r != 10 || s != 5 || th != 2.5 {
		t.Fatalf("angel %v %v %v", r, s, th)
	}
}

func TestItemsRoundTripAndEquip(t *testing.T) {
	for k := 0; k < NumKinds; k++ {
		for tr := 1; tr <= NumTiers; tr++ {
			for r := Rarity(0); r < NumRarities; r++ {
				id := ItemID(k, tr, r)
				it, ok := ItemByID(id)
				if !ok || it.ID != id || it.Name == "" {
					t.Fatalf("bad item %s", id)
				}
			}
		}
	}
	if _, ok := ItemByID("k9t1r0"); ok {
		t.Fatal("out-of-range kind accepted")
	}
	f := mk(1, Human, Fire)
	f.Gold = 10_000
	it, err := f.Buy(6, 1) // starter weapon
	if err != nil || f.Gold != 10_000-it.ShopPrice() {
		t.Fatalf("buy: %v", err)
	}
	if err := f.Equip(7, it.ID); err != ErrSlot {
		t.Fatalf("weapon in off-hand slot: %v", err)
	}
	if err := f.Equip(6, it.ID); err != nil || f.WeaponDmg != 25 || f.Inv[it.ID] != 0 {
		t.Fatalf("equip: %v dmg=%v", err, f.WeaponDmg)
	}
	if _, err := f.Buy(6, 5); err != ErrItemLevel {
		t.Fatalf("level gate: %v", err)
	}
	// ring slot 2 unlocks at 60
	ring, _ := f.Buy(8, 1)
	if err := f.Equip(9, ring.ID); err != ErrSlotLevel {
		t.Fatalf("slot gate: %v", err)
	}
	// weapon damage adds to physical attacks
	e := &Engine{R: fixed(0.5)}
	a, d := mk(1, Dwarf, Fire), mk(2, Human, Water)
	base := e.Attack(a, d, nil, t0).Damage
	a2, d2 := mk(1, Dwarf, Fire), mk(2, Human, Water)
	a2.AddItem(it.ID, 1)
	a2.Equip(6, it.ID)
	if got := e.Attack(a2, d2, nil, t0).Damage; got <= base {
		t.Fatalf("weapon had no effect: %d vs %d", got, base)
	}
	if p, err := f.Sell(ring.ID); err != nil || p != ring.SellPrice() {
		t.Fatalf("sell: %v", err)
	}
}

func TestDropRollAndAngelBoost(t *testing.T) {
	tbl := DropTable(3, 0)
	if len(tbl) != 12 {
		t.Fatalf("table size %d", len(tbl))
	}
	// R=0.0 → every entry succeeds, one random pick; R close to 1 → nothing
	if id := RollDrop(tbl, 0, [3]float64{1, 1, 1}, fixed(0)); id == "" {
		t.Fatal("expected a drop")
	}
	if id := RollDrop(tbl, 0, [3]float64{1, 1, 1}, fixed(0.999)); id != "" {
		t.Fatal("unexpected drop")
	}
	// boosted rarest entry: 0.2% × 10 = 2% → a 1.5% roll now succeeds only for the boosted entries
	only := []Drop{{ID: ItemID(0, 1, Legendary), P: 0.2, R: Legendary}}
	if RollDrop(only, 0, [3]float64{1, 1, 1}, fixed(0.01)) != "" {
		t.Fatal("unboosted should miss")
	}
	if RollDrop(only, 0, [3]float64{10, 5, 2.5}, fixed(0.01)) == "" {
		t.Fatal("boosted should hit")
	}
}

func TestHuntDoesNotMutateOnClone(t *testing.T) {
	e := &Engine{R: fixed(0.5)}
	p := mk(1, Dwarf, Fire)
	p.Alloc[StatPower] = 5
	p.Alloc[StatMana] = 0
	opts := MonsterOptions(1)
	if len(opts) == 0 {
		t.Fatal("no monsters")
	}
	hr := e.Hunt(p, opts[0], t0)
	if hr.Fail != FailNone || (!hr.Win && !hr.Died && !hr.Fled) {
		t.Fatalf("%+v", hr)
	}
	if hr.Win && (p.Gold != hr.Gold || p.Gold == 0) {
		t.Fatalf("gold not paid: %+v", hr)
	}
	if hr.Died && !p.Dead(t0) {
		t.Fatal("player should be dead")
	}
}

// TestHuntBalance prints win rates; it asserts only broad sanity.
func TestHuntBalance(t *testing.T) {
	type build struct {
		name  string
		race  Race
		lvl   int
		alloc map[Stat]int
	}
	builds := []build{
		{"L1 dwarf power", Dwarf, 1, map[Stat]int{StatPower: 5}},
		{"L1 human power", Human, 1, map[Stat]int{StatPower: 5}},
		{"L30 elf", Elf, 30, map[Stat]int{StatPower: 60, StatHP: 40, StatDefense: 30, StatControl: 20}},
		{"L60 dwarf", Dwarf, 60, map[Stat]int{StatPower: 90, StatHP: 90, StatDefense: 60, StatControl: 30}},
	}
	for _, b := range builds {
		for _, idx := range []int{0, 1, 2} {
			wins, n := 0, 100
			for i := 0; i < n; i++ {
				e := NewEngine()
				f := NewFighter(1, "p", b.race, false, []Element{Fire}, t0)
				f.Level = b.lvl
				for s, v := range b.alloc {
					f.Alloc[s] = v
				}
				f.HP, f.Mana = f.MaxHP(), f.MaxMana()
				opts := MonsterOptions(b.lvl)
				spec := opts[len(opts)-1] // hardest offered
				spec.Index = idx
				spec.Name = monsterNames[spec.Tier-1][idx]
				if e.Hunt(f, spec, t0).Win {
					wins++
				}
			}
			o := MonsterOptions(b.lvl)
			w := o[len(o)-1]
			t.Logf("%-16s vs %-14s (tier %d, lvl %2d): %3d%% wins", b.name, monsterNames[w.Tier-1][idx], w.Tier, w.Level, wins)
		}
	}
}

func TestDeathPenaltyAndReset(t *testing.T) {
	f := mk(1, Human, Fire)
	f.Gold, f.XP = 1000, 80
	f.Die(t0)
	if f.Gold != 900 || f.XP != 60 || f.LostGold != 100 || f.LostXP != 20 {
		t.Fatalf("penalty: gold %d xp %d", f.Gold, f.XP)
	}
	g := mk(2, Human, Fire)
	if err := g.ResetStats(); err != ErrNothingToReset {
		t.Fatalf("empty reset: %v", err)
	}
	g.Alloc[StatPower], g.Points = 5, 0
	if err := g.ResetStats(); err != ErrNoGold {
		t.Fatalf("reset must not be free: %v", err)
	}
	g.Gold = 100 // level 1 costs 100
	if err := g.ResetStats(); err != nil || g.Points != 5 || g.Alloc[StatPower] != 0 || g.Resets != 1 || g.Gold != 0 {
		t.Fatalf("reset: %v %+v", err, g)
	}
	g.Alloc[StatPower], g.Points, g.Level = 5, 0, 7
	if err := g.ResetStats(); err != ErrNoGold {
		t.Fatalf("paid reset without gold: %v", err)
	}
	g.Gold = 700
	if err := g.ResetStats(); err != nil || g.Gold != 0 {
		t.Fatalf("paid reset: %v gold %d", err, g.Gold)
	}
}

func TestDungeonRun(t *testing.T) {
	e := &Engine{R: fixed(0.5)}
	p := NewFighter(1, "p", Dwarf, false, []Element{Fire}, t0)
	p.Level = 12
	p.Alloc[StatPower], p.Alloc[StatHP], p.Alloc[StatDefense] = 60, 60, 60
	p.HP, p.Mana = p.MaxHP(), p.MaxMana()
	run := NewDungeonRun(p.Level)
	if run.Tier != 2 || run.Total() != 6 {
		t.Fatalf("run %+v", run)
	}
	now := t0
	var last *FloorResult
	for i := 0; i < 6 && (last == nil || !last.Finished); i++ {
		last = e.DungeonFloor(p, run, now)
		now = now.Add(time.Minute)
		if last.Hunt.Win && last.Hunt.Taken < 0 {
			t.Fatal("negative damage")
		}
	}
	if last == nil || !last.Finished {
		t.Fatal("run never finished")
	}
	if last.Hunt.Died {
		if p.Gold != 0 && run.Gold == 0 { // loot lost, but penalty applied to existing gold only
			t.Fatal("inconsistent")
		}
	} else if last.Loot == nil || p.Gold < last.Loot.Gold {
		t.Fatalf("no payout: %+v", last)
	}
	// floors scale up and the boss is last
	r2 := NewDungeonRun(50)
	r2.Cleared = DungeonFloors
	if !r2.NextSpec().Boss || r2.NextSpec().Name != bossNames[4] {
		t.Fatal("boss floor wrong")
	}
}
