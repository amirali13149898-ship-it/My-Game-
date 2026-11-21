package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ardian/internal/game"
	"ardian/internal/tg"
)

const huntCooldown = 15 * time.Second

func (b *Bot) viewHunt(f *game.Fighter) view {
	var sb strings.Builder
	sb.WriteString("🐗 <b>شکار</b>\n<i>نبرد خودکار با هیولا؛ منبع طلا، XP و آیتم‌های کمیاب (حالت آزمایشی)</i>\n\n")
	var kb tg.Keyboard
	for _, m := range game.MonsterOptions(f.Level) {
		fmt.Fprintf(&sb, "%s <b>%s</b> — %s · لول %s\n    🎁 ~%s XP · ~%s 🪙 + شانس دراپ\n",
			game.ElementEmoji[m.Element()], m.Name, game.ArchetypeNames[m.Index], game.FaI(m.Level),
			game.Fa(m.XPReward()), game.Fa(m.GoldReward()))
		kb = append(kb, []tg.Button{{Text: "⚔️ " + m.Name, Data: fmt.Sprintf("hu:go:%d:%d", m.Tier, m.Index)}})
	}
	return view{text: sb.String(), kb: append(kb, backMain)}
}

func (b *Bot) cbHunt(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	if p[0] != "go" || len(p) < 3 {
		return b.viewHunt(f)
	}
	tier, _ := strconv.Atoi(p[1])
	idx, _ := strconv.Atoi(p[2])
	var spec *game.MonsterSpec
	for _, m := range game.MonsterOptions(f.Level) {
		if m.Tier == tier && m.Index == idx {
			m := m
			spec = &m
		}
	}
	now := b.now()
	switch {
	case spec == nil:
		return b.viewHunt(f)
	case now.Sub(b.lastHunt[f.ID]) < huntCooldown:
		return view{toast: "⏳ " + fmtDur((huntCooldown - now.Sub(b.lastHunt[f.ID])).Seconds()) + " استراحت کن", keep: true}
	}
	hr := b.eng.Hunt(f, *spec, now)
	if hr.Fail == game.FailSelfDead {
		return view{toast: "💀 مُرده‌ای! " + fmtDur(f.DeadUntil.Sub(now).Seconds()) + " تا بازگشت", alert: true, keep: true}
	}
	b.lastHunt[f.ID] = now
	for _, name := range hr.UsesChanged {
		b.db.SaveUses(f.ID, name, f.Uses[name])
	}
	b.saveAll(f)
	kb := tg.Keyboard{{{Text: "🔁 دوباره", Data: fmt.Sprintf("hu:go:%d:%d", tier, idx)}, {Text: "🔙 شکار", Data: "hu:0"}}}
	return view{text: huntText(f, hr), kb: kb}
}

func huntText(f *game.Fighter, hr *game.HuntResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🐗 <b>%s</b> (لول %s) — ⏱ %s\n", hr.Spec.Name, game.FaI(hr.Spec.Level), fmtDur(float64(hr.Seconds)))
	switch {
	case hr.Win:
		sb.WriteString("✅ <b>پیروز شدی!</b>\n")
	case hr.Died:
		fmt.Fprintf(&sb, "💀 <b>کشته شدی...</b> ۵ دقیقه دیگه برمی‌گردی.\n💸 جریمه: −%s 🪙 · −%s XP\n", game.Fa(hr.LostGold), game.Fa(hr.LostXP))
	default:
		sb.WriteString("🏃 زمان تموم شد و هیچ‌کس نبرد رو نبرد.\n")
	}
	fmt.Fprintf(&sb, "🩸 دمیج زدی: %s   💔 دمیج خوردی: %s\n%s\n", game.FaI(hr.Dealt), game.FaI(hr.Taken), hpManaLines(f))
	if hr.Win {
		fmt.Fprintf(&sb, "\n🎁 +%s XP · +%s 🪙", game.Fa(hr.XP), game.Fa(hr.Gold))
		if hr.Levels > 0 {
			fmt.Fprintf(&sb, "\n🎉 به لول %s رسیدی! (+%s امتیاز استت)", game.FaI(f.Level), game.FaI(hr.Levels*game.PointsPerLevel))
		}
		if it, ok := game.ItemByID(hr.Drop); ok {
			fmt.Fprintf(&sb, "\n🧰 دراپ: %s", itemLine(it))
		}
	}
	return sb.String()
}
