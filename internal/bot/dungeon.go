package bot

import (
	"fmt"
	"strings"
	"time"

	"ardian/internal/game"
	"ardian/internal/tg"
)

const dungeonCooldown = 10 * time.Minute

func lootText(l *game.DungeonLoot, f *game.Fighter) string {
	s := fmt.Sprintf("🎁 +%s XP · +%s 🪙", game.Fa(l.XP), game.Fa(l.Gold))
	if l.Levels > 0 {
		s += fmt.Sprintf("\n🎉 به لول %s رسیدی! (+%s امتیاز استت)", game.FaI(f.Level), game.FaI(l.Levels*game.PointsPerLevel))
	}
	for _, id := range l.Drops {
		if it, ok := game.ItemByID(id); ok {
			s += "\n🧰 " + itemLine(it)
		}
	}
	return s
}

func (b *Bot) viewDungeon(f *game.Fighter) view {
	f.Tick(b.now())
	var sb strings.Builder
	if run := b.dungeons[f.ID]; run != nil {
		next := run.NextSpec()
		fmt.Fprintf(&sb, "🏯 <b>%s</b>\nطبقهٔ %s از %s پاک شده · %s\n\n💰 غنیمت فعلی: %s XP · %s 🪙 · %s آیتم\n%s\n\nبعدی: %s <b>%s</b>%s",
			game.DungeonName(run.Tier), game.FaI(run.Cleared), game.FaI(run.Total()), hpOnly(f),
			game.Fa(run.XP), game.Fa(run.Gold), game.FaI(len(run.Drops)), hpManaLines(f),
			game.ElementEmoji[next.Element()], next.Name, map[bool]string{true: " 👹 (باس)"}[next.Boss])
		return view{text: sb.String(), kb: tg.Keyboard{
			{{Text: "⏭ طبقهٔ بعد", Data: "dg:next", Style: "danger"}, {Text: "💰 خروج با غنیمت", Data: "dg:out", Style: "success"}},
			backMain}}
	}
	tier := game.DungeonTier(f.Level)
	fmt.Fprintf(&sb, "🏯 <b>دانجن: %s</b>\n<i>%s طبقه + باس. HP و مانا بین طبقه‌ها بازیابی نمی‌شه. بعد از هر طبقه می‌تونی با غنیمت خارج بشی؛ اگه بمیری غنیمت از دست می‌ره و جریمهٔ مرگ اعمال می‌شه.</i>\n\n",
		game.DungeonName(tier), game.FaI(game.DungeonFloors))
	fmt.Fprintf(&sb, "📋 شرط ورود: لول %s · حداقل ۵۰٪ HP · ۱۰ دقیقه فاصله بین دو دانجن\n%s",
		game.FaI(game.DungeonMinLevel), hpManaLines(f))
	if w := b.lastDungeon[f.ID].Add(dungeonCooldown).Sub(b.now()); w > 0 {
		fmt.Fprintf(&sb, "\n\n⏳ %s تا دانجن بعدی", fmtDur(w.Seconds()))
	}
	return view{text: sb.String(), kb: tg.Keyboard{{{Text: "🚪 ورود", Data: "dg:in", Style: "danger"}}, backMain}}
}

func hpOnly(f *game.Fighter) string { return "❤️ " + fl(f.HP) + "/" + fl(f.MaxHP()) }

func (b *Bot) cbDungeon(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	now := b.now()
	fail := func(s string) view { return view{toast: s, alert: true, keep: true} }
	run := b.dungeons[f.ID]
	switch p[0] {
	case "in":
		f.Tick(now)
		switch {
		case run != nil:
		case f.Level < game.DungeonMinLevel:
			return fail(fmt.Sprintf("برای دانجن لول %s لازمه", game.FaI(game.DungeonMinLevel)))
		case f.Dead(now):
			return fail("💀 مُرده‌ای! " + fmtDur(f.DeadUntil.Sub(now).Seconds()) + " تا بازگشت")
		case f.HPFrac() < 0.5:
			return fail("برای ورود حداقل ۵۰٪ HP لازمه")
		case now.Sub(b.lastDungeon[f.ID]) < dungeonCooldown:
			return fail("⏳ " + fmtDur((dungeonCooldown - now.Sub(b.lastDungeon[f.ID])).Seconds()) + " صبر کن")
		default:
			b.dungeons[f.ID] = game.NewDungeonRun(f.Level)
			return b.dungeonFight(f)
		}
		return b.dungeonFight(f)
	case "next":
		if run == nil {
			return b.viewDungeon(f)
		}
		return b.dungeonFight(f)
	case "out":
		if run == nil {
			return b.viewDungeon(f)
		}
		delete(b.dungeons, f.ID)
		b.lastDungeon[f.ID] = now
		if run.Cleared == 0 {
			v := b.viewDungeon(f)
			v.toast = "بدون جنگ بیرون اومدی"
			return v
		}
		l := b.eng.DungeonClaim(f, run)
		b.saveAll(f)
		return view{text: "🏯 <b>با غنیمت خارج شدی</b>\n\n" + lootText(l, f), kb: tg.Keyboard{{{Text: "🔙 دانجن", Data: "dg:0"}}}}
	}
	return b.viewDungeon(f)
}

func (b *Bot) dungeonFight(f *game.Fighter) view {
	now := b.now()
	run := b.dungeons[f.ID]
	fr := b.eng.DungeonFloor(f, run, now)
	if fr.Fail != game.FailNone {
		delete(b.dungeons, f.ID)
		return view{toast: "💀 مُرده‌ای!", alert: true, keep: true}
	}
	hr := fr.Hunt
	for _, name := range hr.UsesChanged {
		b.db.SaveUses(f.ID, name, f.Uses[name])
	}
	b.saveAll(f)

	var sb strings.Builder
	boss := ""
	if fr.Boss {
		boss = " 👹"
	}
	fmt.Fprintf(&sb, "🏯 <b>%s</b> — طبقهٔ %s از %s%s\n%s %s (لول %s) · ⏱ %s\n🩸 %s   💔 %s\n",
		game.DungeonName(run.Tier), game.FaI(fr.Floor), game.FaI(run.Total()), boss,
		game.ElementEmoji[hr.Spec.Element()], hr.Spec.Name, game.FaI(hr.Spec.Level), fmtDur(float64(hr.Seconds)),
		game.FaI(hr.Dealt), game.FaI(hr.Taken))
	var kb tg.Keyboard
	switch {
	case hr.Died:
		fmt.Fprintf(&sb, "\n💀 <b>کشته شدی!</b> غنیمت از دست رفت.\n💸 جریمه: −%s 🪙 · −%s XP", game.Fa(hr.LostGold), game.Fa(hr.LostXP))
	case hr.Win && fr.Finished:
		fmt.Fprintf(&sb, "\n🏆 <b>باس رو شکست دادی و دانجن تموم شد!</b>\n%s\n\n%s", hpManaLines(f), lootText(fr.Loot, f))
	case hr.Win:
		fmt.Fprintf(&sb, "\n✅ طبقه پاک شد!\n%s\n💰 غنیمت: %s XP · %s 🪙 · %s آیتم", hpManaLines(f),
			game.Fa(run.XP), game.Fa(run.Gold), game.FaI(len(run.Drops)))
		kb = tg.Keyboard{{{Text: "⏭ طبقهٔ بعد", Data: "dg:next", Style: "danger"}, {Text: "💰 خروج با غنیمت", Data: "dg:out", Style: "success"}}}
	default:
		fmt.Fprintf(&sb, "\n🏃 زمان تموم شد و کسی نبرد رو نبرد؛ با غنیمت عقب‌نشینی کردی.\n\n%s", lootText(fr.Loot, f))
	}
	if fr.Finished {
		delete(b.dungeons, f.ID)
		b.lastDungeon[f.ID] = now
		kb = tg.Keyboard{{{Text: "🔙 دانجن", Data: "dg:0"}}}
	}
	return view{text: sb.String(), kb: kb}
}
