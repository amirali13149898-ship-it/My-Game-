package bot

import (
	"fmt"
	"strconv"
	"strings"

	"ardian/internal/game"
	"ardian/internal/tg"
)

// ---------- item text ----------

func statsText(stats [game.NumStats]int, weapon float64) string {
	var parts []string
	if weapon > 0 {
		parts = append(parts, "+"+game.FaF(weapon, 0)+" دمیج سلاح")
	}
	for s := game.Stat(0); s < game.NumStats; s++ {
		if stats[s] != 0 {
			parts = append(parts, fmt.Sprintf("%+d %s", stats[s], game.StatNames[s]))
		}
	}
	return game.FaS(strings.Join(parts, " · "))
}

func itemLine(it *game.Item) string {
	return fmt.Sprintf("%s <b>%s</b> (%s) · لول %s\n    %s", game.RarityEmoji[it.Rarity], it.Name,
		game.RarityNames[it.Rarity], game.FaI(it.ReqLevel), statsText(it.Stats, float64(it.Weapon)))
}

func pairRows(btns []tg.Button) tg.Keyboard {
	var kb tg.Keyboard
	for i := 0; i < len(btns); i += 2 {
		kb = append(kb, btns[i:min(i+2, len(btns))])
	}
	return kb
}

// ---------- shop ----------

func (b *Bot) viewShop(f *game.Fighter) view {
	var btns []tg.Button
	for k := 0; k < game.NumKinds; k++ {
		btns = append(btns, tg.Button{Text: game.SlotEmoji[k] + " " + game.ItemKindNames[k], Data: "sh:k:" + strconv.Itoa(k)})
	}
	var kb tg.Keyboard
	for i := 0; i < len(btns); i += 3 {
		kb = append(kb, btns[i:min(i+3, len(btns))])
	}
	return view{text: fmt.Sprintf("🛒 <b>شاپ</b>\n🪙 طلا: %s\n\nآیتم‌های معمولی اینجا خریدنیه؛ آیتم‌های کمیاب‌تر فقط از شکار دراپ می‌شن.\nآیتم‌هایی که نیاز نداری رو تو منوی تجهیزات می‌فروشی (۴۰٪ ارزش).",
		game.Fa(f.Gold)), kb: append(kb, backMain)}
}

func (b *Bot) viewShopKind(f *game.Fighter, kind int) view {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🛒 <b>%s %s</b>   🪙 %s\n\n", game.SlotEmoji[kind], game.ItemKindNames[kind], game.Fa(f.Gold))
	var btns []tg.Button
	for t := 1; t <= game.NumTiers; t++ {
		it := game.ShopItem(kind, t)
		mark := "🔒"
		if f.Level >= it.ReqLevel {
			mark = "✅"
			btns = append(btns, tg.Button{
				Text: fmt.Sprintf("%s (%s🪙)", it.Name, game.Fa(it.ShopPrice())),
				Data: fmt.Sprintf("sh:b:%d:%d", kind, t),
			})
		}
		fmt.Fprintf(&sb, "%s %s · لول %s · %s🪙\n    %s\n", mark, it.Name, game.FaI(it.ReqLevel),
			game.Fa(it.ShopPrice()), statsText(it.Stats, float64(it.Weapon)))
	}
	return view{text: sb.String(), kb: append(pairRows(btns), []tg.Button{{Text: "🔙 بازگشت", Data: "sh:0"}})}
}

func (b *Bot) cbShop(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	num := func(i int) int {
		if i >= len(p) {
			return -1
		}
		n, err := strconv.Atoi(p[i])
		if err != nil {
			return -1
		}
		return n
	}
	switch p[0] {
	case "k":
		if k := num(1); k >= 0 && k < game.NumKinds {
			return b.viewShopKind(f, k)
		}
	case "b":
		k, t := num(1), num(2)
		it, err := f.Buy(k, t)
		if err != nil {
			return view{toast: err.Error(), alert: true, keep: true}
		}
		b.saveAll(f)
		v := b.viewShopKind(f, k)
		v.toast = "✅ " + it.Name + " خریده شد"
		return v
	}
	return b.viewShop(f)
}

// saveAll persists the player row and the inventory/equipment.
func (b *Bot) saveAll(f *game.Fighter) {
	b.save(f)
	if err := b.db.SaveGear(f); err != nil {
		logf("save gear %d: %v", f.ID, err)
	}
}

// ---------- equipment ----------

const maxBagShown = 10

func gearSummary(f *game.Fighter) string {
	s := statsText(f.Gear, f.WeaponDmg)
	if s == "" {
		return "—"
	}
	return s
}

func (b *Bot) viewEquip(f *game.Fighter) view {
	var sb strings.Builder
	sb.WriteString("🎒 <b>تجهیزات</b>\n\n")
	var btns []tg.Button
	for s := 0; s < game.NumSlots; s++ {
		label := game.SlotEmoji[s] + " " + game.SlotNames[s]
		switch {
		case f.Level < game.SlotUnlock[s]:
			fmt.Fprintf(&sb, "🔒 %s <i>(باز شدن: لول %s)</i>\n", label, game.FaI(game.SlotUnlock[s]))
		case f.Equipped[s] == "":
			fmt.Fprintf(&sb, "▫️ %s: —\n", label)
		default:
			it, _ := game.ItemByID(f.Equipped[s])
			fmt.Fprintf(&sb, "%s: %s %s\n", label, game.RarityEmoji[it.Rarity], it.Name)
		}
		if f.Level >= game.SlotUnlock[s] {
			btns = append(btns, tg.Button{Text: label, Data: "eq:s:" + strconv.Itoa(s)})
		}
	}
	fmt.Fprintf(&sb, "\n📈 بونوس کل: %s", gearSummary(f))
	return view{text: sb.String(), kb: append(pairRows(btns), backMain)}
}

func (b *Bot) viewSlot(f *game.Fighter, slot int) view {
	var sb strings.Builder
	var kb tg.Keyboard
	fmt.Fprintf(&sb, "%s <b>%s</b>   🪙 %s\n\n", game.SlotEmoji[slot], game.SlotNames[slot], game.Fa(f.Gold))
	if id := f.Equipped[slot]; id != "" {
		it, _ := game.ItemByID(id)
		sb.WriteString("پوشیده:\n" + itemLine(it) + "\n\n")
		kb = append(kb, []tg.Button{{Text: "❎ درآوردن " + it.Name, Data: fmt.Sprintf("eq:u:%d", slot)}})
	} else {
		sb.WriteString("چیزی نپوشیدی.\n\n")
	}
	bag := f.BagFor(slot)
	extra := 0
	if len(bag) > maxBagShown { // keep the message under Telegram's 4096 limit
		extra, bag = len(bag)-maxBagShown, bag[:maxBagShown]
	}
	if len(bag) == 0 {
		sb.WriteString("🎒 تو کوله‌ات آیتمی برای این اسلات نیست.")
	} else {
		sb.WriteString("🎒 کوله‌پشتی:\n")
	}
	for _, id := range bag {
		it, _ := game.ItemByID(id)
		fmt.Fprintf(&sb, "%s ×%s\n", itemLine(it), game.FaI(f.Inv[id]))
		kb = append(kb, []tg.Button{
			{Text: "✅ پوشیدن " + it.Name, Data: fmt.Sprintf("eq:e:%d:%s", slot, id)},
			{Text: fmt.Sprintf("💰 %s", game.Fa(it.SellPrice())), Data: fmt.Sprintf("eq:x:%d:%s", slot, id)},
		})
	}
	if extra > 0 {
		fmt.Fprintf(&sb, "\n… و %s آیتم دیگه (اول چندتا رو بپوش یا بفروش)", game.FaI(extra))
	}
	return view{text: sb.String(), kb: append(kb, []tg.Button{{Text: "🔙 بازگشت", Data: "eq:0"}})}
}

func (b *Bot) cbEquip(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	slot := -1
	if len(p) > 1 {
		if n, err := strconv.Atoi(p[1]); err == nil && n >= 0 && n < game.NumSlots {
			slot = n
		}
	}
	if p[0] == "0" || slot < 0 {
		return b.viewEquip(f)
	}
	toast := ""
	switch p[0] {
	case "s":
	case "e":
		if len(p) < 3 {
			return view{keep: true}
		}
		if err := f.Equip(slot, p[2]); err != nil {
			return view{toast: err.Error(), alert: true, keep: true}
		}
		b.saveAll(f)
		toast = "✅ پوشیدی"
	case "u":
		if f.Unequip(slot) {
			b.saveAll(f)
			toast = "درآوردی"
		}
	case "x":
		if len(p) < 3 {
			return view{keep: true}
		}
		gold, err := f.Sell(p[2])
		if err != nil {
			return view{toast: err.Error(), alert: true, keep: true}
		}
		b.saveAll(f)
		toast = "💰 فروختی: +" + game.Fa(gold)
	}
	v := b.viewSlot(f, slot)
	v.toast = toast
	return v
}
