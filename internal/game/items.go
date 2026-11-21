package game

import (
	"fmt"
	"math"
	"sort"
)

// Items are generated from a compact id "k{kind}t{tier}r{rarity}" so there is
// no giant item table: 9 kinds × 10 tiers × 5 rarities = 450 items.
// NOTE: the design doc leaves item mechanics open; everything here is a
// data-driven default (see README "تصمیم‌های پیش‌فرض") that is easy to change.

const (
	NumSlots = 10
	NumKinds = 9
	NumTiers = 10
)

var SlotNames = [NumSlots]string{"ماسک", "کلاه", "زره", "شلوار", "دستکش", "کفش", "سلاح اصلی", "سلاح فرعی", "حلقه ۱", "حلقه ۲"}
var SlotEmoji = [NumSlots]string{"🎭", "🎩", "🛡", "👖", "🧤", "👢", "⚔️", "🔰", "💍", "💍"}
var ItemKindNames = [NumKinds]string{"ماسک", "کلاه", "زره", "شلوار", "دستکش", "کفش", "سلاح", "سپر/فرعی", "حلقه"}

// SlotUnlock is the player level at which a slot becomes usable.
var SlotUnlock = [NumSlots]int{25, 5, 1, 15, 20, 10, 1, 30, 40, 60}

// KindOfSlot maps a slot to the item kind it accepts (both rings share a kind).
func KindOfSlot(slot int) int {
	if slot >= NumKinds-1 {
		return NumKinds - 1
	}
	return slot
}

func SlotAccepts(slot, kind int) bool {
	return slot >= 0 && slot < NumSlots && KindOfSlot(slot) == kind
}

type Rarity int

const (
	Common Rarity = iota
	Uncommon
	Rare
	Epic
	Legendary
	NumRarities
)

var RarityNames = [NumRarities]string{"معمولی", "غیرمعمول", "کمیاب", "حماسی", "افسانه‌ای"}
var RarityEmoji = [NumRarities]string{"⚪", "🟢", "🔵", "🟣", "🟠"}
var rarityMul = [NumRarities]float64{1, 1.25, 1.6, 2.0, 2.6}

var itemNames = [NumKinds][NumTiers]string{
	{"نقاب پارچه‌ای", "ماسک چرمی", "ماسک مسی", "ماسک آهنی", "ماسک نقره‌ای", "ماسک شبح", "ماسک رعد", "ماسک سایه‌های کهن", "ماسک اژدهاپوش", "ماسک خدایان"},
	{"کلاه نمدی", "کلاه چرمی", "کلاه مسی", "کلاه آهنی", "کلاه جادوگر", "کلاه ستاره‌بین", "کلاه طوفان", "تاج فولاد", "تاج اژدها", "تاج کیهان"},
	{"پیراهن کتان", "زره چرمی", "زره مسی", "زره آهنی", "زره فولادی", "زره دیوپوش", "زره مینا", "زره اژدها", "زره نور", "زره بی‌پایان"},
	{"شلوار کتان", "شلوار چرمی", "ساق‌پوش مسی", "ساق‌پوش آهنی", "ساق‌پوش فولاد", "ساق‌پوش شبح", "ساق‌پوش مینا", "ساق‌پوش اژدها", "ساق‌پوش نور", "ساق‌پوش ازل"},
	{"دستکش پارچه‌ای", "دستکش چرمی", "دستکش مسی", "دستکش آهنی", "دستکش فولاد", "پنجهٔ شبح", "پنجهٔ رعد", "پنجهٔ اژدها", "پنجهٔ نور", "پنجهٔ خدایان"},
	{"صندل کتان", "چکمهٔ چرمی", "چکمهٔ مسی", "چکمهٔ آهنی", "چکمهٔ فولاد", "چکمهٔ باد", "چکمهٔ طوفان", "چکمهٔ اژدها", "چکمهٔ نور", "چکمهٔ بی‌پایان"},
	{"چوب‌دستی", "خنجر آهنی", "شمشیر کوتاه", "شمشیر فولاد", "تبر جنگی", "نیزهٔ باد", "شمشیر شعله", "تیغ ماه", "اژدها‌کُش", "شمشیر کیهان"},
	{"سپر چوبی", "سپر چرمی", "سپر مسی", "سپر آهنی", "سپر فولاد", "سپر برجی", "سپر مینا", "سپر اژدها", "سپر نور", "سپر ابدی"},
	{"حلقهٔ مس", "حلقهٔ برنج", "حلقهٔ نقره", "حلقهٔ طلا", "حلقهٔ یاقوت", "حلقهٔ زمرد", "حلقهٔ الماس", "حلقهٔ ستاره", "حلقهٔ آسمان", "حلقهٔ ازل"},
}

type Item struct {
	ID       string
	Kind     int
	Tier     int
	Rarity   Rarity
	Name     string
	ReqLevel int
	Stats    [NumStats]int
	Weapon   int // flat physical damage (main weapon only)
}

func ItemID(kind, tier int, r Rarity) string { return fmt.Sprintf("k%dt%dr%d", kind, tier, int(r)) }

// ItemByID rebuilds an item from its id; ok=false for malformed ids.
func ItemByID(id string) (*Item, bool) {
	var k, t, r int
	if n, err := fmt.Sscanf(id, "k%dt%dr%d", &k, &t, &r); n != 3 || err != nil {
		return nil, false
	}
	if k < 0 || k >= NumKinds || t < 1 || t > NumTiers || r < 0 || r >= int(NumRarities) || ItemID(k, t, Rarity(r)) != id {
		return nil, false
	}
	return MakeItem(k, t, Rarity(r)), true
}

func MakeItem(kind, tier int, r Rarity) *Item {
	m, t := rarityMul[r], float64(tier)
	p := max(1, int(math.Round(1.2*t*m)))
	s := max(1, int(math.Round(0.6*t*m)))
	it := &Item{ID: ItemID(kind, tier, r), Kind: kind, Tier: tier, Rarity: r, ReqLevel: 1 + (tier-1)*10,
		Name: itemNames[kind][tier-1]}
	switch kind {
	case 0: // mask
		it.Stats[StatPerception], it.Stats[StatLuck] = p, s
	case 1: // hat
		it.Stats[StatMana], it.Stats[StatControl] = p, s
	case 2: // chest
		it.Stats[StatHP], it.Stats[StatDefense] = p, s
	case 3: // pants
		it.Stats[StatDefense], it.Stats[StatSpeed] = p, s
	case 4: // gloves
		it.Stats[StatPower], it.Stats[StatCrit] = p, s
	case 5: // boots
		it.Stats[StatSpeed], it.Stats[StatPerception] = p, s
	case 6: // main weapon: flat damage + a little power
		it.Weapon = int(math.Round(25 * math.Pow(t, 1.4) * m))
		it.Stats[StatPower] = s
	case 7: // off-hand
		it.Stats[StatDefense], it.Stats[StatHP] = p, s
	case 8: // ring
		it.Stats[StatLuck], it.Stats[StatCrit] = p, s
	}
	return it
}

func (it *Item) DisplayName() string {
	return RarityEmoji[it.Rarity] + " " + it.Name
}

var kindPriceFactor = [NumKinds]float64{1.5, 1.5, 1.5, 1.5, 1.5, 1.5, 3, 2, 2}

// Value is the base worth in gold; shop price for common items.
func (it *Item) Value() int64 {
	v := 40 * float64(it.Tier*it.Tier) * kindPriceFactor[it.Kind] * rarityMul[it.Rarity] * rarityMul[it.Rarity]
	return int64(math.Round(v/10)) * 10
}

func (it *Item) ShopPrice() int64 { return it.Value() }
func (it *Item) SellPrice() int64 { return max(1, it.Value()*40/100) }

// ShopItem returns the common item sold for a kind and tier.
func ShopItem(kind, tier int) *Item { return MakeItem(kind, tier, Common) }

// ---------- drops ----------

type Drop struct {
	ID string
	P  float64 // percent
	R  Rarity
}

// DropTable is the loot table of a monster (tier 1..10, index 0..2).
// Each table has 3 item kinds; the three rarest entries are the ones that
// the angel racial (Raiko) boosts.
func DropTable(tier, mi int) []Drop {
	var kinds []int
	for k := 0; k < NumKinds; k++ {
		if (k+mi)%3 == 0 {
			kinds = append(kinds, k)
		}
	}
	var t []Drop
	add := func(k int, r Rarity, p float64) { t = append(t, Drop{ItemID(k, tier, r), p, r}) }
	for i, k := range kinds {
		add(k, Common, 8)
		add(k, Uncommon, 4)
		add(k, Rare, 1.5)
		if i >= 1 {
			add(k, Epic, 0.6)
		}
		if i == 2 {
			add(k, Legendary, 0.2)
		}
	}
	return t
}

// RollDrop rolls every entry independently; if several succeed one is picked
// at random (max 1 item per drop). Boosts multiply the chance of the three
// rarest entries; luck adds +0.5% (relative) per point to rare-or-better ones.
func RollDrop(table []Drop, luck int, boost [3]float64, r Rand) string {
	t := append([]Drop(nil), table...)
	sort.SliceStable(t, func(i, j int) bool { return t[i].P < t[j].P })
	var won []string
	for i := range t {
		p := t[i].P
		if i < 3 {
			p *= boost[i]
		}
		if t[i].R >= Rare {
			p *= 1 + 0.005*float64(luck)
		}
		if r.Float64()*100 < p {
			won = append(won, t[i].ID)
		}
	}
	if len(won) == 0 {
		return ""
	}
	return won[int(r.Float64()*float64(len(won)))%len(won)]
}
