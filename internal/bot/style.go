package bot

import (
	"fmt"
	"strings"

	"ardian/internal/game"
)

// Text styling helpers. Telegram has no font sizes, so "big" text is made of
// bold, headers with dividers, quotes and emoji bars.

const divider = "▬▬▬▬▬▬▬▬▬▬▬▬"

// header is a title banner: ✦ ▬▬▬ 🏰 <b>title</b> ▬▬▬ ✦ style.
func header(emoji, title string) string {
	return fmt.Sprintf("✦ <b>%s  %s</b> ✦\n%s\n", emoji, title, divider)
}

// quote renders a note as a block quote (a visually separate, indented box).
func quote(s string) string { return "<blockquote>" + s + "</blockquote>" }

// bar draws an emoji progress bar, e.g. 🟥🟥🟥⬛⬛ (n cells).
func bar(cur, max float64, n int, full string) string {
	if max <= 0 {
		max = 1
	}
	k := int(cur/max*float64(n) + 0.5)
	if k < 0 {
		k = 0
	}
	if k > n {
		k = n
	}
	if k == 0 && cur > 0 {
		k = 1
	}
	return strings.Repeat(full, k) + strings.Repeat("⬛", n-k)
}

func hpManaLines(f *game.Fighter) string {
	return fmt.Sprintf("❤️ <b>HP</b>  %s  %s / %s\n💧 <b>مانا</b>  %s  %s / %s",
		bar(f.HP, f.MaxHP(), 8, "🟥"), fl(f.HP), fl(f.MaxHP()),
		bar(f.Mana, f.MaxMana(), 8, "🟦"), fl(f.Mana), fl(f.MaxMana()))
}

func xpLine(f *game.Fighter) string {
	need := float64(game.XPNeeded(f.Level))
	return fmt.Sprintf("✨ <b>XP</b>  %s  %s / %s", bar(float64(f.XP), need, 8, "🟨"), game.Fa(f.XP), game.Fa(game.XPNeeded(f.Level)))
}
