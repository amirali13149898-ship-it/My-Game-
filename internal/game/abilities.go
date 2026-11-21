package game

// ---------- Ability model ----------

type Kind int

const (
	Offensive Kind = iota
	Support
	Defensive
)

var KindNames = [3]string{"تهاجمی", "پشتیبانی", "دفاعی"}

type EffType string

const (
	EffOutDmg   EffType = "out_dmg"    // attacker: +% damage
	EffOutCrit  EffType = "out_crit"   // attacker: +% crit chance
	EffOutAcc   EffType = "out_acc"    // attacker: +% accuracy
	EffInDmgRed EffType = "in_red"     // defender: -% damage taken
	EffInCrit   EffType = "in_critred" // defender: -% damage taken from crits
	EffInDodge  EffType = "in_dodge"   // defender: +% dodge
	EffReflect  EffType = "reflect"    // defender: % of damage returned
	EffManaCost EffType = "mana_cost"  // -% mana cost of next spells
	EffCDRed    EffType = "cd_red"     // -% cooldown of next spells
	EffHoT      EffType = "hot"        // heal over time
	EffInstant  EffType = "instant"    // instant HP / mana
	EffCDAll    EffType = "cd_all"     // reduce remaining cooldowns
	EffZetsu    EffType = "zetsu"      // dark elf: next physical may crit 2.5x
	EffKyubi    EffType = "kyubi"      // fox: next incoming attacks auto-miss
)

type Filter int

const (
	FAny Filter = iota
	FPhys
	FMagic
)

func (f Filter) match(phys bool) bool {
	return f == FAny || (f == FPhys && phys) || (f == FMagic && !phys)
}

// EffSpec describes the effect of a support/defensive ability.
// Value at upgrade level L = Base + Inc*(L-1).
type EffSpec struct {
	Type    EffType
	Base    float64
	Inc     float64
	Charges int     // number of next attacks/spells (0 when Timed)
	Dur     float64 // seconds, 0 = DefaultEffDur
	Filter  Filter
	OnMiss  bool // charge is consumed even when the attack misses
	Timed   bool // time-limited only, no charges
	Every   float64
	Ticks   int
	Mana    float64 // instant mana (Base is instant HP)
	ManaInc float64
}

type Ability struct {
	Name     string
	NormName string
	Element  Element
	Kind     Kind
	Unlock   int
	Cooldown float64
	Mana     int
	Dmg      float64 // offensive: center damage at upgrade level 1
	Inc      float64 // offensive: growth per level
	Fixed    bool    // offensive: no random range
	Eff      EffSpec
}

func (a *Ability) IsSpell() bool { return true }

var (
	allAbilities []*Ability
	byNorm       = map[string]*Ability{}
	byElement    [NumElements][]*Ability
)

// AbilityByName finds a spell by (unnormalized) text.
func AbilityByName(text string) *Ability { return byNorm[Norm(text)] }

// ElementAbilities returns all 20 abilities of an element ordered by unlock level.
func ElementAbilities(e Element) []*Ability { return byElement[e] }

type off struct {
	name   string
	unlock int
	cd     float64
	mana   int
	dmg    float64
	fixed  bool
	inc    float64
}

type sup struct {
	name   string
	unlock int
	cd     float64
	mana   int
	eff    EffSpec
}

func register(a *Ability) {
	a.NormName = Norm(a.Name)
	if _, dup := byNorm[a.NormName]; dup {
		panic("duplicate ability name: " + a.Name)
	}
	byNorm[a.NormName] = a
	allAbilities = append(allAbilities, a)
	byElement[a.Element] = append(byElement[a.Element], a)
}

func addElement(el Element, offs []off, sups []sup, defs []sup) {
	for _, o := range offs {
		inc := o.inc
		if inc == 0 {
			inc = OffIncPerLevel
		}
		register(&Ability{Name: o.name, Element: el, Kind: Offensive, Unlock: o.unlock,
			Cooldown: o.cd, Mana: o.mana, Dmg: o.dmg, Inc: inc, Fixed: o.fixed})
	}
	for _, s := range sups {
		register(&Ability{Name: s.name, Element: el, Kind: Support, Unlock: s.unlock,
			Cooldown: s.cd, Mana: s.mana, Eff: s.eff})
	}
	for _, s := range defs {
		register(&Ability{Name: s.name, Element: el, Kind: Defensive, Unlock: s.unlock,
			Cooldown: s.cd, Mana: s.mana, Eff: s.eff})
	}
	// order by unlock level (stable)
	list := byElement[el]
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].Unlock < list[j-1].Unlock; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func init() {
	// ---------------- Fire ----------------
	addElement(Fire,
		[]off{
			{"فایربال", 1, 5, 40, 20, false, 0},
			{"فایر ارو", 5, 10, 60, 60, false, 0},
			{"فایر ویپو", 10, 15, 70, 90, false, 0},
			{"فوریمو سوراشو", 25, 30, 100, 120, false, 0},
			{"فایر رانسو", 30, 35, 120, 150, false, 0},
			{"فوریمو باستو", 45, 50, 150, 180, false, 0},
			{"فایر سوتومو", 60, 65, 190, 210, false, 0},
			{"فوریمو دوراگون", 75, 80, 230, 260, false, 0},
			{"اینفرنو دوزخ", 80, 85, 250, 290, false, 0},
			{"فوریمو آپوکاریپوسو", 95, 200, 300, 300, true, 20},
		},
		[]sup{
			{"فایر رایتو", 15, 20, 80, EffSpec{Type: EffOutDmg, Base: 10, Inc: 2, Charges: 2, OnMiss: true}},
			{"ووموسو", 35, 40, 130, EffSpec{Type: EffOutCrit, Base: 7, Inc: 2, Charges: 1}},
			{"فوریمو انچانتو", 50, 55, 160, EffSpec{Type: EffOutDmg, Base: 5, Inc: 2, Charges: 3, Filter: FPhys}},
			{"فوریمو ههیسوتو", 65, 70, 200, EffSpec{Type: EffCDRed, Base: 5, Inc: 5, Charges: 3}},
			{"فنیکوسو بورسو", 85, 90, 270, EffSpec{Type: EffHoT, Base: 30, Inc: 5, Every: 2, Ticks: 30, Dur: 60}},
		},
		[]sup{
			{"فایر گادو", 20, 25, 90, EffSpec{Type: EffInDmgRed, Base: 10, Inc: 2, Charges: 2}},
			{"فوریمو باریا", 40, 45, 140, EffSpec{Type: EffInCrit, Base: 7, Inc: 2, Charges: 1}},
			{"هیتو شیرودو", 55, 60, 170, EffSpec{Type: EffReflect, Base: 4, Inc: 2, Charges: 1, Filter: FPhys}},
			{"فایر وورو", 70, 75, 210, EffSpec{Type: EffInDmgRed, Base: 7, Inc: 2, Charges: 2, Filter: FMagic}},
			{"فنیکوسو ایجیسو", 90, 100, 290, EffSpec{Type: EffInDmgRed, Base: 44, Inc: 4, Charges: 1}},
		})

	// ---------------- Water ----------------
	addElement(Water,
		[]off{
			{"واتر بورو", 1, 4, 32, 16, false, 0},
			{"واتر ارو", 5, 8, 48, 48, false, 0},
			{"واتر ویپو", 10, 12, 56, 72, false, 0},
			{"آکوا کاتا", 25, 24, 80, 96, false, 0},
			{"ووتا رانسو", 30, 28, 96, 120, false, 0},
			{"آکوا جتو", 45, 40, 120, 144, false, 0},
			{"واتر بومو", 60, 52, 152, 168, false, 0},
			{"تایداری ویبو", 75, 64, 184, 208, false, 0},
			{"ووتا دوراگون", 80, 68, 200, 232, false, 0},
			{"مهیروسوتورومو", 95, 160, 240, 240, true, 16},
		},
		[]sup{
			{"واتر هیرو", 15, 16, 64, EffSpec{Type: EffInstant, Base: 150, Inc: 20}},
			{"کورنجینگو", 35, 32, 104, EffSpec{Type: EffInstant, Mana: 130, ManaInc: 20}},
			{"آکوا بورسو", 50, 44, 128, EffSpec{Type: EffManaCost, Base: 10, Inc: 2, Charges: 3}},
			{"میسوتو ورو", 65, 56, 160, EffSpec{Type: EffInDodge, Base: 6, Inc: 2, Charges: 3}},
			{"رایفو فونتو", 85, 72, 216, EffSpec{Type: EffHoT, Base: 48, Inc: 8, Every: 3, Ticks: 15, Dur: 45}},
		},
		[]sup{
			{"ووتا شیرودو", 20, 20, 72, EffSpec{Type: EffInDmgRed, Base: 20, Inc: 3, Charges: 1}},
			{"آکوا باریا", 40, 36, 112, EffSpec{Type: EffInDodge, Base: 10, Inc: 2, Charges: 1}},
			{"بابورو دومو", 55, 48, 136, EffSpec{Type: EffReflect, Base: 4, Inc: 2, Charges: 1, Filter: FMagic}},
			{"واتر وورو", 70, 60, 168, EffSpec{Type: EffInDmgRed, Base: 7, Inc: 2, Charges: 2, Filter: FPhys}},
			{"تایداری ایجیسو", 90, 80, 232, EffSpec{Type: EffInDmgRed, Base: 22, Inc: 2, Charges: 2}},
		})

	// ---------------- Earth ----------------
	addElement(Earth,
		[]off{
			{"آسو بورو", 1, 6, 48, 24, false, 0},
			{"آسو آرو", 5, 12, 72, 72, false, 0},
			{"سوتون ویپو", 10, 18, 84, 108, false, 0},
			{"آسو سوپایکو", 25, 36, 120, 144, false, 0},
			{"روککو رانسو", 30, 42, 144, 180, false, 0},
			{"سوتون کیانون", 45, 60, 180, 216, false, 0},
			{"روککو باستو", 60, 78, 228, 252, false, 0},
			{"آسو دوراگون", 75, 96, 276, 312, false, 0},
			{"ساندو سوتومو", 80, 102, 300, 348, false, 0},
			{"آسوکوئیکو", 95, 240, 360, 360, true, 24},
		},
		[]sup{
			{"سوتون سوکین", 15, 24, 96, EffSpec{Type: EffInDmgRed, Base: 5, Inc: 1, Timed: true, Dur: 60}},
			{"آسو سوتورنگوسو", 35, 48, 156, EffSpec{Type: EffOutDmg, Base: 4, Inc: 1, Timed: true, Dur: 60}},
			{"روتو بایندو", 50, 66, 192, EffSpec{Type: EffOutAcc, Base: 6, Inc: 2, Charges: 3}},
			{"فاتیریتی", 65, 84, 240, EffSpec{Type: EffInstant, Base: 360, Inc: 40, Mana: 120, ManaInc: 20}},
			{"گایا بورسو", 85, 108, 324, EffSpec{Type: EffHoT, Base: 72, Inc: 12, Every: 4, Ticks: 15, Dur: 60}},
		},
		[]sup{
			{"آسو وورو", 20, 30, 108, EffSpec{Type: EffInDmgRed, Base: 8, Inc: 2, Charges: 3, Filter: FPhys}},
			{"روککو شیرودو", 40, 54, 168, EffSpec{Type: EffInCrit, Base: 8, Inc: 2, Charges: 2}},
			{"سوتون باریا", 55, 72, 204, EffSpec{Type: EffReflect, Base: 5, Inc: 2, Charges: 1}},
			{"ساندو ورو", 70, 90, 252, EffSpec{Type: EffInDmgRed, Base: 8, Inc: 2, Charges: 2, Filter: FMagic}},
			{"گایا ایجیسو", 90, 120, 348, EffSpec{Type: EffInDmgRed, Base: 18, Inc: 2, Charges: 3}},
		})

	// ---------------- Wind ----------------
	addElement(Wind,
		[]off{
			{"ویندو بورو", 1, 3, 24, 12, false, 0},
			{"ویندو آرو", 5, 6, 36, 36, false, 0},
			{"ویندو ویپو", 10, 9, 42, 54, false, 0},
			{"ویندو کاتا", 25, 18, 60, 72, false, 0},
			{"ویندو رانسو", 30, 21, 72, 90, false, 0},
			{"اآ بارتو", 45, 30, 90, 108, false, 0},
			{"ویندو بومو", 60, 39, 114, 126, false, 0},
			{"تنپسوتو", 75, 48, 138, 156, false, 0},
			{"ویندو دوراگون", 80, 51, 150, 174, false, 0},
			{"هاریکن", 95, 120, 180, 180, true, 12},
		},
		[]sup{
			{"ویندو سوتپو", 15, 12, 48, EffSpec{Type: EffInDodge, Base: 5, Inc: 2, Timed: true, Dur: 60}},
			{"فزا فورو", 35, 24, 78, EffSpec{Type: EffOutCrit, Base: 4, Inc: 2, Charges: 2}},
			{"اآ بورسو", 50, 33, 96, EffSpec{Type: EffOutDmg, Base: 5, Inc: 2, Charges: 3, Filter: FMagic}},
			{"تیرو ویندو", 65, 42, 120, EffSpec{Type: EffCDAll, Base: 10}},
			{"سوکای بورسو", 85, 54, 162, EffSpec{Type: EffHoT, Base: 18, Inc: 3, Every: 1, Ticks: 30, Dur: 30}},
		},
		[]sup{
			{"ویندو شیرودو", 20, 15, 54, EffSpec{Type: EffInDmgRed, Base: 6, Inc: 2, Charges: 2}},
			{"اآ باریا", 40, 27, 84, EffSpec{Type: EffInCrit, Base: 4, Inc: 2, Charges: 2}},
			{"گاسوتو وورو", 55, 36, 102, EffSpec{Type: EffInDodge, Base: 5, Inc: 2, Charges: 4}},
			{"سایکورون گادو", 70, 45, 126, EffSpec{Type: EffReflect, Base: 3, Inc: 2, Charges: 2}},
			{"سوتومو ایجیسو", 90, 60, 174, EffSpec{Type: EffInDmgRed, Base: 13, Inc: 1, Charges: 2}},
		})
}
