package bot

import (
	"fmt"
	"strconv"
	"strings"

	"ardian/internal/game"
	"ardian/internal/tg"
)

func fl(v float64) string { return game.FaI(int(v + 0.5)) }

func hpManaLines(f *game.Fighter) string {
	return fmt.Sprintf("❤️ HP: %s / %s\n💧 مانا: %s / %s", fl(f.HP), fl(f.MaxHP()), fl(f.Mana), fl(f.MaxMana()))
}

func elemList(f *game.Fighter) string {
	parts := make([]string, len(f.Elems))
	for i, e := range f.Elems {
		parts[i] = e.String()
	}
	return strings.Join(parts, " + ")
}

func sexName(female bool) string {
	if female {
		return "زن"
	}
	return "مرد"
}

func (b *Bot) viewMain(f *game.Fighter) view {
	f.Tick(b.now())
	status := ""
	if f.Dead(b.now()) {
		status = "\n💀 <b>مُرده‌ای</b> — " + fmtDur(f.DeadUntil.Sub(b.now()).Seconds()) + " تا بازگشت"
	}
	t := fmt.Sprintf("🏰 <b>منوی اصلی</b>\n\n%s <b>%s</b> — %s %s\n🔮 %s\n🎖 لول %s   ✨ XP: %s / %s\n%s\n🪙 طلا: %s%s",
		game.RaceEmoji[f.Race], esc(f.Name), game.RaceNames[f.Race], sexName(f.Female), elemList(f),
		game.FaI(f.Level), game.Fa(f.XP), game.Fa(game.XPNeeded(f.Level)), hpManaLines(f), game.Fa(f.Gold), status)
	kb := tg.Keyboard{
		{{Text: "📊 استت", Data: "st:view", Style: "success"}, {Text: "✨ تواناییها", Data: "ab:0", Style: "success"}},
		{{Text: "🎒 تجهیزات", Data: "eq:0", Style: "danger"}, {Text: "🛒 شاپ", Data: "sh:0", Style: "danger"}},
		{{Text: "⚔️ کلن", Data: "cl:0", Style: "primary"}, {Text: "🏆 لیدربرد", Data: "rk:level", Style: "primary"}},
		{{Text: "🏯 دانجن", Data: "dg:0"}, {Text: "🐗 شکار", Data: "hu:0"}},
		{{Text: "📖 راهنمایی", Data: "go:help"}},
	}
	return view{text: t, kb: kb, photo: "main"}
}

var backMain = []tg.Button{{Text: "🔙 بازگشت", Data: "go:main"}}

// ---------- stats ----------

func (b *Bot) draft(id int64) *[game.NumStats]int {
	d := b.drafts[id]
	if d == nil {
		d = new([game.NumStats]int)
		b.drafts[id] = d
	}
	return d
}

func draftUsed(d *[game.NumStats]int) (n int) {
	for _, v := range d {
		n += v
	}
	return
}

func (b *Bot) viewStats(f *game.Fighter) view {
	f.Tick(b.now())
	d := b.draft(f.ID)
	used := draftUsed(d)
	var sb strings.Builder
	fmt.Fprintf(&sb, "📊 <b>استت‌ها</b>\n\n🎖 لول: %s   ✨ XP: %s / %s\n%s\n🎯 امتیاز استت: <b>%s</b>   🪙 طلا: %s\n\n",
		game.FaI(f.Level), game.Fa(f.XP), game.Fa(game.XPNeeded(f.Level)), hpManaLines(f),
		game.FaI(f.Points-used), game.Fa(f.Gold))
	for s := game.Stat(0); s < game.NumStats; s++ {
		cur := f.Stat(s)
		line := fmt.Sprintf("%s %s: %s", game.StatEmoji[s], game.StatNames[s], game.FaI(cur))
		if d[s] > 0 {
			line += fmt.Sprintf(" ← <b>%s</b>", game.FaI(cur+d[s]))
		}
		sb.WriteString(line + "\n")
	}
	var kb tg.Keyboard
	if f.Points > 0 {
		for s := game.Stat(0); s < game.NumStats; s++ {
			i := strconv.Itoa(int(s))
			label := fmt.Sprintf("%s %s [%s]", game.StatEmoji[s], game.StatNames[s], game.FaI(f.Raw(s)+d[s]))
			// Written right-to-left in the spec: zero | name | +1 | −1 | +10
			// Telegram lays buttons out left-to-right, so the order is mirrored.
			kb = append(kb, []tg.Button{
				{Text: "+۱۰", Data: "st:+:" + i + ":10"},
				{Text: "−۱", Data: "st:-:" + i},
				{Text: "+۱", Data: "st:+:" + i + ":1"},
				{Text: label, Data: "st:nop"},
				{Text: "۰", Data: "st:0:" + i},
			})
		}
		kb = append(kb, []tg.Button{
			{Text: "✅ تایید", Data: "st:ok", Style: "success"},
			{Text: "❌ لغو", Data: "st:no", Style: "danger"},
		})
	}
	if f.SpentPoints() > 0 {
		cost := "رایگان"
		if c := f.ResetCost(); c > 0 {
			cost = game.Fa(c) + " 🪙"
		}
		kb = append(kb, []tg.Button{{Text: "♻️ ریست استت (" + cost + ")", Data: "st:rs"}})
	}
	if f.Race == game.Human && f.Level >= game.HumanBuyMinLevel && f.Bought < game.HumanBuyMax {
		kb = append(kb, []tg.Button{{
			Text: fmt.Sprintf("🛒 خرید ۱ امتیاز استت (%s 🪙)", game.Fa(game.HumanPointPrice(f.Bought+1))),
			Data: "st:buy",
		}})
	}
	return view{text: sb.String(), kb: append(kb, backMain)}
}

func (b *Bot) cbStats(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	d := b.draft(f.ID)
	atoi := func(i int) int {
		if i >= len(p) {
			return -1
		}
		n, _ := strconv.Atoi(p[i])
		return n
	}
	switch p[0] {
	case "view":
	case "nop":
		return view{keep: true}
	case "+":
		s, n := atoi(1), atoi(2)
		if s < 0 || s >= game.NumStats || n < 1 {
			return view{keep: true}
		}
		n = min(n, f.Points-draftUsed(d), game.StatMax(game.Stat(s))-(f.Raw(game.Stat(s))+d[s]))
		if n <= 0 {
			if f.Points-draftUsed(d) <= 0 {
				return view{toast: "امتیاز استت نداری", keep: true}
			}
			return view{toast: "این استت به سقفش رسیده", keep: true}
		}
		d[s] += n
	case "-":
		if s := atoi(1); s >= 0 && s < game.NumStats && d[s] > 0 {
			d[s]--
		}
	case "0":
		if s := atoi(1); s >= 0 && s < game.NumStats {
			d[s] = 0
		}
	case "no":
		delete(b.drafts, f.ID)
		v := b.viewStats(f)
		v.toast = "لغو شد"
		return v
	case "ok":
		used := draftUsed(d)
		if used == 0 {
			return view{toast: "تغییری نداده‌ای", keep: true}
		}
		if used > f.Points {
			delete(b.drafts, f.ID)
			return view{toast: "امتیاز کافی نیست", alert: true, keep: true}
		}
		for s := range d {
			f.Alloc[s] += d[s]
		}
		f.Points -= used
		delete(b.drafts, f.ID)
		b.save(f)
		v := b.viewStats(f)
		v.toast = "✅ ثبت شد"
		return v
	case "rs": // ask for confirmation
		if f.SpentPoints() == 0 {
			return view{toast: "امتیازی خرج نکرده‌ای", keep: true}
		}
		cost := "رایگانه (بار اول)"
		if c := f.ResetCost(); c > 0 {
			cost = "هزینه: " + game.Fa(c) + " 🪙"
		}
		return view{text: fmt.Sprintf("♻️ <b>ریست استت</b>\n\nهمهٔ %s امتیاز خرج‌شده برمی‌گرده تا دوباره پخشش کنی. %s",
			game.FaI(f.SpentPoints()), cost), kb: tg.Keyboard{
			{{Text: "✅ ریست کن", Data: "st:rs2", Style: "success"}, {Text: "❌ لغو", Data: "st:view", Style: "danger"}}}}
	case "rs2":
		delete(b.drafts, f.ID)
		if err := f.ResetStats(); err != nil {
			return view{toast: err.Error(), alert: true, keep: true}
		}
		b.save(f)
		v := b.viewStats(f)
		v.toast = "♻️ امتیازها برگشت"
		return v
	case "buy":
		if v, ok := b.buyHumanPoint(f); !ok {
			return v
		}
	}
	return b.viewStats(f)
}

// buyHumanPoint buys one stat point (human racial: from level 50, max 100).
func (b *Bot) buyHumanPoint(f *game.Fighter) (view, bool) {
	switch {
	case f.Race != game.Human || f.Level < game.HumanBuyMinLevel:
		return view{toast: "هنوز نمی‌تونی بخری", keep: true}, false
	case f.Bought >= game.HumanBuyMax:
		return view{toast: "سقف خرید پر شده", keep: true}, false
	}
	price := game.HumanPointPrice(f.Bought + 1)
	if f.Gold < price {
		return view{toast: "طلای کافی نداری", alert: true, keep: true}, false
	}
	f.Gold -= price
	f.Bought++
	f.Points++
	b.save(f)
	return view{}, true
}

// ---------- abilities ----------

func (b *Bot) viewAbilities(f *game.Fighter, tab string) view {
	f.Tick(b.now())
	var kb tg.Keyboard
	var tabs []tg.Button
	for i, e := range f.Elems {
		tabs = append(tabs, tg.Button{Text: e.String(), Data: "ab:" + strconv.Itoa(i)})
	}
	tabs = append(tabs, tg.Button{Text: "🧬 نژاد", Data: "ab:r"})
	kb = append(kb, tabs)

	var sb strings.Builder
	if tab == "r" {
		sb.WriteString(b.raceText(f))
		if btn, ok := raceBuyButton(f); ok {
			kb = append(kb, []tg.Button{btn})
		}
	} else {
		idx, _ := strconv.Atoi(tab)
		if idx < 0 || idx >= len(f.Elems) {
			idx = 0
		}
		el := f.Elems[idx]
		fmt.Fprintf(&sb, "✨ <b>جادوهای %s</b>\n<i>تهاجمی: ریپلای + اسم | پشتیبانی/دفاعی: فقط اسم</i>\n", el)
		if idx == 1 {
			fmt.Fprintf(&sb, "⚠️ عنصر دوم ضعیف‌تره (از ۳۰٪ تا ۱۰٪ بسته به لول جادو)\n")
		}
		sb.WriteString("\n")
		for _, ab := range game.ElementAbilities(el) {
			sb.WriteString(abilityLine(f, ab) + "\n")
		}
	}
	return view{text: sb.String(), kb: append(kb, backMain)}
}

func abilityLine(f *game.Fighter, ab *game.Ability) string {
	kindEmoji := [3]string{"⚔️", "🔰", "🛡"}[ab.Kind]
	if f.Level < ab.Unlock {
		return fmt.Sprintf("🔒 %s <i>(لول %s)</i>", ab.Name, game.FaI(ab.Unlock))
	}
	lvl := f.SpellLevel(ab)
	line := fmt.Sprintf("%s <b>%s</b> · ارتقا %s · 💧%s · ⏱%ss", kindEmoji, ab.Name,
		game.FaI(lvl), game.FaI(ab.Mana), game.FaF(ab.Cooldown, 2))
	if lvl < 10 {
		line += fmt.Sprintf(" (%s استفادهٔ دیگه)", game.FaI(game.UsesToNext(f.Uses[ab.Name])))
	}
	return line + "\n    " + describeAbility(ab, lvl, f.Weakness(ab))
}

func raceBuyButton(f *game.Fighter) (tg.Button, bool) {
	if f.Race == game.Human {
		return tg.Button{}, false
	}
	next := f.RaceTier + 1
	if next > game.RaceMaxTier(f.Race) {
		return tg.Button{}, false
	}
	label := fmt.Sprintf("🛒 خرید لول %s — %s 🪙", game.FaI(next), game.Fa(game.RaceTierPrice(f.Race, next)))
	if f.Level < game.RaceTierMinLevel(f.Race, next) {
		label = fmt.Sprintf("🔒 لول %s (از لول بازیکن %s)", game.FaI(next), game.FaI(game.RaceTierMinLevel(f.Race, next)))
	}
	return tg.Button{Text: label, Data: "ab:buy"}, true
}

func (b *Bot) cbAbilities(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	if p[0] != "buy" {
		return b.viewAbilities(f, p[0])
	}
	next := f.RaceTier + 1
	switch {
	case f.Race == game.Human || next > game.RaceMaxTier(f.Race):
		return view{toast: "چیزی برای خرید نیست", keep: true}
	case f.Level < game.RaceTierMinLevel(f.Race, next):
		return view{toast: "لول بازیکنت کافی نیست", alert: true, keep: true}
	}
	price := game.RaceTierPrice(f.Race, next)
	if f.Gold < price {
		return view{toast: "طلای کافی نداری", alert: true, keep: true}
	}
	f.Gold -= price
	f.RaceTier = next
	b.save(f)
	v := b.viewAbilities(f, "r")
	v.toast = "✅ خریدی!"
	return v
}
