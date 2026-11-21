package bot

import (
	"fmt"
	"math"
	"strings"

	"ardian/internal/game"
)

// fmtDur renders seconds as «X دقیقه و Y ثانیه» (rounded up).
func fmtDur(sec float64) string {
	s := int(math.Ceil(sec))
	if s < 1 {
		s = 1
	}
	if s >= 60 {
		if s%60 == 0 {
			return game.FaI(s/60) + " دقیقه"
		}
		return game.FaI(s/60) + " دقیقه و " + game.FaI(s%60) + " ثانیه"
	}
	return game.FaI(s) + " ثانیه"
}

func pct(v float64) string { return game.FaF(v, 1) + "٪" }

func scope(f game.Filter) string {
	switch f {
	case game.FPhys:
		return " فیزیکی"
	case game.FMagic:
		return " جادویی"
	}
	return ""
}

// effText describes an effect with its value already computed for the level.
func effText(s game.EffSpec, val float64) string {
	var what string
	switch s.Type {
	case game.EffOutDmg:
		what = "+" + pct(val) + " دمیج" + scope(s.Filter)
	case game.EffOutCrit:
		what = "+" + pct(val) + " شانس کریتیکال"
	case game.EffOutAcc:
		what = "+" + pct(val) + " دقت"
	case game.EffInDmgRed:
		what = "−" + pct(val) + " دمیج دریافتی" + scope(s.Filter)
	case game.EffInCrit:
		what = "−" + pct(val) + " دمیج کریتیکال دریافتی"
	case game.EffInDodge:
		what = "+" + pct(val) + " جاخالی"
	case game.EffReflect:
		what = pct(val) + " از دمیج" + scope(s.Filter) + " به مهاجم برمی‌گرده"
	case game.EffManaCost:
		return "−" + pct(val) + " مصرف مانای " + game.FaI(s.Charges) + " جادوی بعدی"
	case game.EffCDRed:
		return "−" + pct(math.Min(val, 50)) + " کولداون " + game.FaI(s.Charges) + " جادوی بعدی"
	case game.EffCDAll:
		return "−" + pct(val) + " از کولداون باقیماندهٔ بقیهٔ ابیلیتی‌ها"
	case game.EffHoT:
		return fmt.Sprintf("هر %s ثانیه %s HP، به مدت %s", game.FaF(s.Every, 1), game.FaF(val, 1), fmtDur(s.Dur))
	case game.EffInstant:
		var parts []string
		if s.Base > 0 {
			parts = append(parts, "+"+game.FaF(val, 0)+" HP")
		}
		if s.Mana > 0 {
			parts = append(parts, "+"+game.FaF(val, 0)+" مانا")
		}
		return strings.Join(parts, " و ")
	}
	if s.Timed {
		return what + " به مدت " + fmtDur(s.Dur)
	}
	return what + " برای " + game.FaI(s.Charges) + " حملهٔ بعدی"
}

// describeAbility is the one-line description at an upgrade level.
func describeAbility(ab *game.Ability, lvl int, weak float64) string {
	k := 1 - weak/100
	if ab.Kind == game.Offensive {
		c := (ab.Dmg + ab.Inc*float64(lvl-1)) * k
		if ab.Fixed {
			return "💥 دمیج ثابت " + game.FaF(c, 0)
		}
		r := float64(lvl) * k
		return fmt.Sprintf("💥 دمیج %s تا %s", game.FaF(c-r, 0), game.FaF(c+r, 0))
	}
	s := ab.Eff
	val := (s.Base + s.Inc*float64(lvl-1)) * k
	if s.Type == game.EffInstant && s.Base == 0 { // mana-only instant
		val = (s.Mana + s.ManaInc*float64(lvl-1)) * k
	}
	return "🔸 " + effText(s, val)
}

// ---------- race abilities ----------

func raceTierEffect(f *game.Fighter, tier int) string {
	switch f.Race {
	case game.Elf:
		v := game.ElfTier(tier)
		return fmt.Sprintf("برمی‌گردونه %s مانا + %s از مانای مصرفی ۱۰ ثانیهٔ اخیر (سقف %s) · کولداون %s",
			game.FaF(v.Fixed, 0), pct(v.Pct), game.FaF(v.Cap, 0), fmtDur(game.ElfCooldown))
	case game.DarkElf:
		eff := min(f.Level, tier*10)
		ch, cd := game.DarkElfVals(eff)
		return fmt.Sprintf("حملهٔ فیزیکی بعدی با شانس %s کریتیکال ۲٫۵x می‌شه · کولداون %s", pct(ch), fmtDur(cd))
	case game.Dwarf:
		v := game.DwarfTier(tier)
		return fmt.Sprintf("%s از دمیج %s ضربهٔ بعدی به مهاجم برمی‌گرده · کولداون %s", pct(v.Reflect), game.FaI(v.Hits), fmtDur(v.CD))
	case game.Angel:
		l := min(f.Level/5+1, tier)
		a, b, c := game.AngelBoosts(l)
		return fmt.Sprintf("ضریب دراپ سه آیتم کمیاب‌تر: ×%s / ×%s / ×%s (لول اسکیل %s)",
			game.FaF(a, 3), game.FaF(b, 3), game.FaF(c, 3), game.FaI(l))
	case game.Tiger:
		mid, low := game.TigerVals(tier)
		return fmt.Sprintf("HP بین ۲۵–۵۰٪: +%s دمیج · زیر ۲۵٪: +%s دمیج (فیزیکی و جادو، کولداون ۸ ثانیه)", pct(mid), pct(low))
	case game.Dragon:
		d, t := game.DragonVals(tier)
		return fmt.Sprintf("وقتی HP ≤ ۲۰٪: +%s دمیج زدن و +%s دمیج دریافتی", pct(d), pct(t))
	case game.Wolf:
		return fmt.Sprintf("هر ضربهٔ موفق یه استک (تا %s)؛ هر استک +%s دمیج. فقط با مرگ صفر می‌شه",
			game.FaI(game.WolfMaxStacks), pct(game.WolfPerStack(tier)))
	case game.Fox:
		n, cd := game.FoxTier(tier)
		return fmt.Sprintf("%s حملهٔ بعدی حریف رایگان جاخالی می‌شه · کولداون %s", game.FaI(n), fmtDur(cd))
	}
	return ""
}

func (b *Bot) raceText(f *game.Fighter) string {
	if f.Race == game.Human {
		s := "👤 <b>انسان</b>\n\nدو عنصر داری؛ عنصر دوم ضعیف‌تره (از ۳۰٪ تا ۱۰٪ با ارتقای جادو).\n"
		s += fmt.Sprintf("از لول %s می‌تونی تا %s امتیاز استت بخری (از منوی استت).\nخریدشده: %s",
			game.FaI(game.HumanBuyMinLevel), game.FaI(game.HumanBuyMax), game.FaI(f.Bought))
		return s
	}
	kind := "پسیو"
	if game.RaceActive(f.Race) {
		kind = "فعال — با نوشتن اسمش تو گروه"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "🧬 <b>%s</b> — %s\n(%s)\n\n", game.RaceAbilityName[f.Race], f.Race, kind)
	fmt.Fprintf(&sb, "📍 لول فعلی: %s / %s\n%s", game.FaI(f.RaceTier), game.FaI(game.RaceMaxTier(f.Race)), raceNowText(f))
	if next := f.RaceTier + 1; next <= game.RaceMaxTier(f.Race) {
		fmt.Fprintf(&sb, "\n\n⏭ لول %s: %s\n🪙 قیمت: %s · از لول بازیکن %s",
			game.FaI(next), raceTierEffect(f, next), game.Fa(game.RaceTierPrice(f.Race, next)),
			game.FaI(game.RaceTierMinLevel(f.Race, next)))
	}
	return sb.String()
}

func raceNowText(f *game.Fighter) string {
	if f.RaceTier < 1 {
		return "هنوز نخریدی."
	}
	return raceTierEffect(f, f.RaceTier)
}
