package game

import (
	"errors"
	"sort"
	"time"
)

var (
	ErrBadItem        = errors.New("آیتم نامعتبر")
	ErrNoItem         = errors.New("این آیتم رو نداری")
	ErrSlot           = errors.New("این آیتم تو این اسلات جا نمی‌گیره")
	ErrSlotLevel      = errors.New("این اسلات هنوز باز نشده")
	ErrItemLevel      = errors.New("لولت برای این آیتم کافی نیست")
	ErrNoGold         = errors.New("طلای کافی نداری")
	ErrNothingToReset = errors.New("امتیازی خرج نکرده‌ای")
)

// RecalcGear recomputes the equipment bonuses from the equipped items.
func (f *Fighter) RecalcGear() {
	f.Gear = [NumStats]int{}
	f.WeaponDmg = 0
	for _, id := range f.Equipped {
		if it, ok := ItemByID(id); ok {
			for s := range f.Gear {
				f.Gear[s] += it.Stats[s]
			}
			f.WeaponDmg += float64(it.Weapon)
		}
	}
}

func (f *Fighter) AddItem(id string, n int) {
	f.Init()
	f.Inv[id] += n
}

func (f *Fighter) takeItem(id string) bool {
	if f.Inv[id] < 1 {
		return false
	}
	if f.Inv[id]--; f.Inv[id] == 0 {
		delete(f.Inv, id)
	}
	return true
}

// Equip moves an item from the bag into a slot (the old item goes back to the bag).
func (f *Fighter) Equip(slot int, id string) error {
	it, ok := ItemByID(id)
	switch {
	case !ok:
		return ErrBadItem
	case !SlotAccepts(slot, it.Kind):
		return ErrSlot
	case f.Level < SlotUnlock[slot]:
		return ErrSlotLevel
	case f.Level < it.ReqLevel:
		return ErrItemLevel
	case f.Inv[id] < 1:
		return ErrNoItem
	}
	f.takeItem(id)
	if old := f.Equipped[slot]; old != "" {
		f.AddItem(old, 1)
	}
	f.Equipped[slot] = id
	f.RecalcGear()
	return nil
}

func (f *Fighter) Unequip(slot int) bool {
	if slot < 0 || slot >= NumSlots || f.Equipped[slot] == "" {
		return false
	}
	f.AddItem(f.Equipped[slot], 1)
	f.Equipped[slot] = ""
	f.RecalcGear()
	return true
}

// Buy adds a common shop item and takes the gold.
func (f *Fighter) Buy(kind, tier int) (*Item, error) {
	if kind < 0 || kind >= NumKinds || tier < 1 || tier > NumTiers {
		return nil, ErrBadItem
	}
	it := ShopItem(kind, tier)
	if f.Level < it.ReqLevel {
		return nil, ErrItemLevel
	}
	if f.Gold < it.ShopPrice() {
		return nil, ErrNoGold
	}
	f.Gold -= it.ShopPrice()
	f.AddItem(it.ID, 1)
	return it, nil
}

// Sell sells one bag item for 40% of its value.
func (f *Fighter) Sell(id string) (int64, error) {
	it, ok := ItemByID(id)
	if !ok {
		return 0, ErrBadItem
	}
	if !f.takeItem(id) {
		return 0, ErrNoItem
	}
	f.Gold += it.SellPrice()
	return it.SellPrice(), nil
}

// BagFor lists bag item ids that fit a slot, best first.
func (f *Fighter) BagFor(slot int) []string {
	var ids []string
	for id := range f.Inv {
		if it, ok := ItemByID(id); ok && SlotAccepts(slot, it.Kind) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, _ := ItemByID(ids[i])
		b, _ := ItemByID(ids[j])
		if a.Tier != b.Tier {
			return a.Tier > b.Tier
		}
		return a.Rarity > b.Rarity
	})
	return ids
}

// Clone deep-copies the fighter so a fight can be simulated without side effects.
func (f *Fighter) Clone() *Fighter {
	c := *f
	c.Elems = append([]Element(nil), f.Elems...)
	c.Uses = make(map[string]int, len(f.Uses))
	for k, v := range f.Uses {
		c.Uses[k] = v
	}
	c.Ready = make(map[string]time.Time, len(f.Ready))
	for k, v := range f.Ready {
		c.Ready[k] = v
	}
	c.Effects = nil
	for _, e := range f.Effects {
		ec := *e
		c.Effects = append(c.Effects, &ec)
	}
	c.manaLog = append([]manaEntry(nil), f.manaLog...)
	return &c
}
