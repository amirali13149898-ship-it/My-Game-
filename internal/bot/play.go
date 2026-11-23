package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ardian/internal/game"
	"ardian/internal/tg"
)

var physWords = map[string]bool{game.Norm("حمله"): true, game.Norm("قدرت"): true}

// onGroupText handles "ability name" messages written in a group.
func (b *Bot) onGroupText(m *tg.Message, text string) {
	norm := game.Norm(text)
	ab := game.AbilityByName(text)
	race, isRace := game.RaceAbilityByName(text)
	phys := physWords[norm]
	if ab == nil && !isRace && !phys {
		return // ordinary chat: no DB access at all
	}
	att := b.fighter(m.From.ID)
	if att == nil || (isRace && race != att.Race) {
		return
	}
	now := b.now()
	chat := m.Chat.ID

	switch {
	case isRace:
		res := b.eng.UseRace(att, now)
		if res.Fail != game.FailNone {
			b.notice(att, m, b.failText(att, nil, res, now))
			return
		}
		b.save(att)
		b.send(chat, raceUseText(att, res), nil, m.MessageID)

	case ab != nil && ab.Kind != game.Offensive:
		res := b.eng.Cast(att, ab, now)
		if res.Fail != game.FailNone {
			if res.Fail != game.FailLocked || att.HasElement(ab.Element) {
				b.notice(att, m, b.failText(att, nil, res, now))
			}
			return
		}
		b.afterSpell(att, ab)
		b.send(chat, castText(att, res), nil, m.MessageID)

	default: // physical attack or offensive spell: needs a target
		if ab != nil && !att.HasElement(ab.Element) {
			return
		}
		r := m.ReplyToMessage
		if r == nil || r.From == nil || r.From.IsBot || r.From.ID == att.ID {
			return
		}
		def := b.fighter(r.From.ID)
		if def == nil {
			return // the opponent has not started the game
		}
		res := b.eng.Attack(att, def, ab, now)
		if res.Fail != game.FailNone {
			b.notice(att, m, b.failText(att, def, res, now))
			return
		}
		if res.Killed {
			att.Kills++
		}
		if ab != nil {
			b.afterSpell(att, ab)
		}
		b.save(att, def)
		b.send(chat, attackText(att, def, res), nil, m.MessageID)
	}
}

// afterSpell persists the spell-use counter and the attacker state.
func (b *Bot) afterSpell(att *game.Fighter, ab *game.Ability) {
	if err := b.db.SaveUses(att.ID, ab.Name, att.Uses[ab.Name]); err != nil {
		log.Printf("save uses: %v", err)
	}
	b.save(att)
}

// notice shows a refusal message, but only if none was shown in the last
// 10 seconds (only the first premature attack gets a reply).
func (b *Bot) notice(att *game.Fighter, m *tg.Message, text string) {
	now := b.now()
	if text == "" || now.Sub(att.LastNotice) < 10*time.Second {
		return
	}
	att.LastNotice = now
	b.send(m.Chat.ID, text, nil, m.MessageID)
}

func (b *Bot) failText(att, def *game.Fighter, r *game.Result, now time.Time) string {
	switch r.Fail {
	case game.FailSelfDead:
		return "💀 مُرده‌ای! " + fmtDur(r.Wait) + " تا بازگشت"
	case game.FailTargetDead:
		return fmt.Sprintf("💀 %s مُرده؛ %s تا بازگشت", esc(def.Name), fmtDur(def.DeadUntil.Sub(now).Seconds()))
	case game.FailCD:
		return "⏳ " + fmtDur(r.Wait) + " صبر کن"
	case game.FailMana:
		return fmt.Sprintf("💧 مانا کافی نیست (نیاز: %s، داری: %s)", fl(r.Need), fl(att.Mana))
	case game.FailLocked:
		return "🔒 هنوز این جادو رو باز نکردی"
	case game.FailNotAvailable:
		return "🧬 این توانایی پسیوئه یا هنوز نخریدیش"
	}
	return ""
}

// ---------- result texts ----------

func hpOf(f *game.Fighter) string { return fl(f.HP) + "/" + fl(f.MaxHP()) }

func attackText(att, def *game.Fighter, r *game.Result) string {
	var sb strings.Builder
	a, d := "<b>"+esc(att.Name)+"</b>", "<b>"+esc(def.Name)+"</b>"
	if r.Ab == nil {
		fmt.Fprintf(&sb, "⚔️ %s به %s حمله کرد!\n\n", a, d)
	} else {
		fmt.Fprintf(&sb, "%s %s با <b>%s</b> (ارتقا %s) به %s زد!\n\n", game.ElementEmoji[r.Ab.Element], a, r.Ab.Name,
			game.FaI(att.SpellLevel(r.Ab)), d)
	}
	if r.Dodge {
		fmt.Fprintf(&sb, "💨 %s جاخالی داد!", d)
	} else {
		if r.Crit {
			sb.WriteString("💥💥 <b>ضربهٔ کریتیکال!</b> 💥💥\n")
		}
		fmt.Fprintf(&sb, "🩸 دمیج: <b>%s</b> 🔥", game.FaI(r.Damage))
		if r.ElemMod != 0 {
			fmt.Fprintf(&sb, "  🔄 چرخهٔ عنصری: %s", signed(r.ElemMod))
		}
		fmt.Fprintf(&sb, "\n❤️ %s  %s  %s", d, bar(def.HP, def.MaxHP(), 6, "🟥"), hpOf(def))
		if r.Reflected > 0 {
			fmt.Fprintf(&sb, "\n🪞 %s دمیج به %s برگشت (❤️ %s)", game.FaI(r.Reflected), a, hpOf(att))
		}
	}
	if r.ManaCost > 0 {
		fmt.Fprintf(&sb, "\n💧 −%s مانا (%s)", game.FaI(r.ManaCost), fl(att.Mana))
	}
	switch {
	case r.Killed:
		fmt.Fprintf(&sb, "\n\n☠️ <b>%s کشته شد!</b> ۵ دقیقه دیگه برمی‌گرده.%s", d, penaltyText(r))
	case r.AttackerDied:
		fmt.Fprintf(&sb, "\n☠️ %s با بازتاب دمیج مُرد!%s", a, penaltyText(r))
	}
	if r.XP > 0 {
		fmt.Fprintf(&sb, "\n🎁 +%s XP · +%s 🪙", game.Fa(r.XP), game.Fa(r.Gold))
	}
	if r.LevelsGained > 0 {
		fmt.Fprintf(&sb, "\n🎉 %s به لول %s رسید! (+%s امتیاز استت)", a, game.FaI(att.Level),
			game.FaI(r.LevelsGained*game.PointsPerLevel))
	}
	if r.NewSpellLvl > 0 {
		fmt.Fprintf(&sb, "\n📈 %s به ارتقای %s رسید!", r.Ab.Name, game.FaI(r.NewSpellLvl))
	}
	return sb.String()
}

func signed(v float64) string {
	if v > 0 {
		return "+" + pct(v)
	}
	return "−" + pct(-v)
}

func castText(f *game.Fighter, r *game.Result) string {
	a := "<b>" + esc(f.Name) + "</b>"
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s <b>%s</b> رو فعال کرد", game.ElementEmoji[r.Ab.Element], a, r.Ab.Name)
	lvl := f.SpellLevel(r.Ab)
	if r.NewSpellLvl > 0 {
		lvl = r.NewSpellLvl
	}
	switch {
	case r.Healed > 0 || r.ManaRestored > 0 || r.Ab.Eff.Type == game.EffInstant:
		sb.WriteString(":")
		if r.Healed > 0 {
			fmt.Fprintf(&sb, " 💚 +%s HP", fl(r.Healed))
		}
		if r.ManaRestored > 0 {
			fmt.Fprintf(&sb, " 💧 +%s مانا", fl(r.ManaRestored))
		}
	default:
		sb.WriteString("\n🔸 " + describeAbility(r.Ab, lvl, f.Weakness(r.Ab))[len("🔸 "):])
	}
	if r.ManaCost > 0 {
		fmt.Fprintf(&sb, "\n💧 −%s مانا (%s)", game.FaI(r.ManaCost), fl(f.Mana))
	}
	if r.NewSpellLvl > 0 {
		fmt.Fprintf(&sb, "\n📈 %s به ارتقای %s رسید!", r.Ab.Name, game.FaI(r.NewSpellLvl))
	}
	return sb.String()
}

func raceUseText(f *game.Fighter, r *game.Result) string {
	a := "<b>" + esc(f.Name) + "</b>"
	name := game.RaceAbilityName[f.Race]
	switch f.Race {
	case game.Elf:
		return fmt.Sprintf("🧝 %s «%s» رو فعال کرد: 💧 +%s مانا (%s)", a, name, fl(r.ManaRestored), fl(f.Mana))
	case game.DarkElf:
		return fmt.Sprintf("🌑 %s «%s» رو فعال کرد: حملهٔ فیزیکی بعدی با شانس %s کریتیکال می‌شه!", a, name, pct(r.Chance))
	case game.Dwarf:
		return fmt.Sprintf("🪨 %s «%s» رو فعال کرد: %s از دمیج %s ضربهٔ بعدی برمی‌گرده!", a, name,
			pct(r.Effect.Value), game.FaI(r.Effect.Charges))
	case game.Fox:
		return fmt.Sprintf("🦊 %s «%s» رو فعال کرد: %s حملهٔ بعدی حریف رایگان جاخالی می‌شه!", a, name, game.FaI(r.Dodges))
	}
	return ""
}

// penaltyText is the death-penalty line (empty when the dead had nothing to lose).
func penaltyText(r *game.Result) string {
	if r.LostGold == 0 && r.LostXP == 0 {
		return ""
	}
	return fmt.Sprintf("\n💸 جریمهٔ مرگ: −%s 🪙 · −%s XP", game.Fa(r.LostGold), game.Fa(r.LostXP))
}
