package bot

import (
	"fmt"
	"strconv"
	"strings"

	"ardian/internal/game"
	"ardian/internal/tg"
)

type raceGroup struct {
	key, label string
	races      []game.Race
}

var raceGroups = []raceGroup{
	{"h", "👤 انسان", []game.Race{game.Human}},
	{"e", "🧝 الف‌ها", []game.Race{game.Elf, game.DarkElf}},
	{"d", "🪨 دورف", []game.Race{game.Dwarf}},
	{"a", "👼 فرشته", []game.Race{game.Angel}},
	{"b", "🐾 نیمه‌ها", []game.Race{game.Tiger, game.Dragon, game.Wolf, game.Fox}},
}

func groupOf(r game.Race) raceGroup {
	for _, g := range raceGroups {
		for _, x := range g.races {
			if x == r {
				return g
			}
		}
	}
	return raceGroups[0]
}

var raceBlurb = [game.NumRaces]string{
	game.Human:   "دو عنصر انتخاب می‌کنه (عنصر دوم ضعیف‌تره) و از لول ۵۰ می‌تونه امتیاز استت بخره.",
	game.Elf:     "«شینرا ایبوکی»: مانا برمی‌گردونه.",
	game.DarkElf: "«زتسو»: حملهٔ فیزیکی بعدی با شانسی کریتیکال می‌شه.",
	game.Dwarf:   "«کوکتسو»: بخشی از دمیج دریافتی رو به مهاجم برمی‌گردونه.",
	game.Angel:   "«رایکو»: شانس دراپ آیتم‌های کمیاب رو چند برابر می‌کنه.",
	game.Tiger:   "«راستسو»: وقتی HP پایینه دمیجش بیشتر می‌شه.",
	game.Dragon:  "«ریوکتسو»: وقتی HP ≤ ۲۰٪ هم دمیج زدن هم دمیج دریافتی بالا می‌ره.",
	game.Wolf:    "«شینگو»: هر ضربهٔ موفق یه استک می‌ده و دمیج رو بالا می‌بره.",
	game.Fox:     "«کیوبی»: حمله‌های بعدی حریف رایگان جاخالی می‌شن.",
}

func statLine(r game.Race, female bool) string {
	parts := make([]string, game.NumStats)
	for s := game.Stat(0); s < game.NumStats; s++ {
		parts[s] = game.StatEmoji[s] + game.FaI(game.BaseStat(r, female, s))
	}
	return strings.Join(parts, " ")
}

func (b *Bot) viewStart(f *game.Fighter) view {
	if f != nil {
		return b.viewMain(f)
	}
	return view{text: b.text("welcome"), photo: "welcome", kb: tg.Keyboard{{
		{Text: "📖 راهنمایی", Data: "go:help", Style: "primary"},  // left
		{Text: "🧙 انتخاب کارکتر", Data: "c:g", Style: "primary"}, // right
	}}}
}

func (b *Bot) cbCreate(u *tg.User, p []string) view {
	if f := b.fighter(u.ID); f != nil {
		return b.viewMain(f)
	}
	back := []tg.Button{{Text: "🔙 بازگشت", Data: "go:start"}}
	atoi := func(i int) int {
		if i >= len(p) {
			return -1
		}
		n, err := strconv.Atoi(p[i])
		if err != nil {
			return -1
		}
		return n
	}
	validRace := func(n int) bool { return n >= 0 && n < game.NumRaces }
	validEl := func(n int) bool { return n >= 0 && n < game.NumElements }

	switch p[0] {
	case "g": // race group list, or a group's races
		if len(p) == 1 {
			var kb tg.Keyboard
			var row []tg.Button
			for _, g := range raceGroups {
				row = append(row, tg.Button{Text: g.label, Data: "c:g:" + g.key})
				if len(row) == 2 {
					kb, row = append(kb, row), nil
				}
			}
			if row != nil {
				kb = append(kb, row)
			}
			return view{text: "🧬 <b>نژادت رو انتخاب کن</b>", kb: append(kb, back)}
		}
		for _, g := range raceGroups {
			if g.key != p[1] {
				continue
			}
			if len(g.races) == 1 {
				return b.viewGender(g.races[0])
			}
			var row []tg.Button
			for _, r := range g.races {
				row = append(row, tg.Button{Text: r.String(), Data: fmt.Sprintf("c:r:%d", r)})
			}
			kb := tg.Keyboard{}
			for i := 0; i < len(row); i += 2 {
				kb = append(kb, row[i:min(i+2, len(row))])
			}
			return view{text: "🧬 <b>" + g.label + "</b> — کدوم؟", kb: append(kb, []tg.Button{{Text: "🔙 بازگشت", Data: "c:g"}})}
		}
	case "r":
		if r := atoi(1); validRace(r) {
			return b.viewGender(game.Race(r))
		}
	case "s": // gender chosen → first element
		r, fem := atoi(1), atoi(2)
		if validRace(r) && (fem == 0 || fem == 1) {
			return b.viewElement(game.Race(r), fem == 1, -1)
		}
	case "e": // first element chosen
		r, fem, e1 := atoi(1), atoi(2), atoi(3)
		if validRace(r) && (fem == 0 || fem == 1) && validEl(e1) {
			if game.Race(r) == game.Human {
				return b.viewElement(game.Human, fem == 1, e1)
			}
			return b.finishCreate(u, game.Race(r), fem == 1, []game.Element{game.Element(e1)})
		}
	case "E": // human second element chosen
		r, fem, e1, e2 := atoi(1), atoi(2), atoi(3), atoi(4)
		if game.Race(r) == game.Human && (fem == 0 || fem == 1) && validEl(e1) && validEl(e2) && e1 != e2 {
			return b.finishCreate(u, game.Human, fem == 1, []game.Element{game.Element(e1), game.Element(e2)})
		}
	}
	return view{keep: true}
}

func (b *Bot) viewGender(r game.Race) view {
	t := fmt.Sprintf("%s\n<i>%s</i>\n\n🧔 <b>مرد</b>\n%s\n\n👩 <b>زن</b>\n%s\n\n<b>جنسیت؟</b>",
		r, raceBlurb[r], statLine(r, false), statLine(r, true))
	kb := tg.Keyboard{
		{{Text: "👩 زن", Data: fmt.Sprintf("c:s:%d:1", r)}, {Text: "🧔 مرد", Data: fmt.Sprintf("c:s:%d:0", r)}},
		{{Text: "🔙 بازگشت", Data: "c:g"}},
	}
	return view{text: t, kb: kb, photo: fmt.Sprintf("race%d", r)}
}

// viewElement shows the element choice. first == -1 → choose the (first)
// element, otherwise choose the human's weaker second element.
func (b *Bot) viewElement(r game.Race, female bool, first int) view {
	fem := 0
	if female {
		fem = 1
	}
	title := "🔮 <b>عنصر جادوت رو انتخاب کن</b>"
	if first >= 0 {
		title = fmt.Sprintf("🔮 عنصر اول: %s\n<b>حالا عنصر دوم (ضعیف‌تر) رو انتخاب کن</b>", game.Element(first))
	} else if r == game.Human {
		title += "\n<i>انسان‌ها دو عنصر دارن؛ اولی کامل، دومی ضعیف‌تر.</i>"
	}
	var row []tg.Button
	for e := 0; e < game.NumElements; e++ {
		if e == first {
			continue
		}
		data := fmt.Sprintf("c:e:%d:%d:%d", r, fem, e)
		if first >= 0 {
			data = fmt.Sprintf("c:E:%d:%d:%d:%d", r, fem, first, e)
		}
		row = append(row, tg.Button{Text: game.Element(e).String(), Data: data})
	}
	kb := tg.Keyboard{}
	for i := 0; i < len(row); i += 2 {
		kb = append(kb, row[i:min(i+2, len(row))])
	}
	return view{text: title, kb: append(kb, []tg.Button{{Text: "🔙 بازگشت", Data: "c:r:" + strconv.Itoa(int(r))}})}
}

func (b *Bot) finishCreate(u *tg.User, r game.Race, female bool, elems []game.Element) view {
	now := b.now()
	f := game.NewFighter(u.ID, displayName(u), r, female, elems, now)
	if err := b.db.CreateFighter(f, now); err != nil {
		return view{toast: "❌ خطا در ساخت کارکتر، دوباره امتحان کن", alert: true, keep: true}
	}
	b.cache[u.ID] = f
	v := b.viewMain(f)
	v.toast = "✅ کارکترت ساخته شد!"
	return v
}
