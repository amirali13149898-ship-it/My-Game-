package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode/utf8"

	"ardian/internal/game"
	"ardian/internal/store"
	"ardian/internal/tg"
)

func logf(format string, a ...any) { log.Printf(format, a...) }

const (
	ClanCreateCost  = 1000
	ClanCreateLevel = 10
	clanPageSize    = 6
)

// ---------- leaderboard ----------

var rankTabs = []struct{ key, label string }{
	{"level", "🎖 لول"}, {"kills", "☠️ کشتار"}, {"gold", "🪙 طلا"}, {"clans", "⚔️ کلن‌ها"},
}

func medal(i int) string {
	switch i {
	case 0:
		return "🥇"
	case 1:
		return "🥈"
	case 2:
		return "🥉"
	}
	return game.FaI(i+1) + "."
}

func (b *Bot) viewRank(f *game.Fighter, tab string) view {
	var tabs []tg.Button
	for _, t := range rankTabs {
		label := t.label
		if t.key == tab {
			label = "▪️ " + label
		}
		tabs = append(tabs, tg.Button{Text: label, Data: "rk:" + t.key})
	}
	kb := tg.Keyboard{tabs[:2], tabs[2:], backMain}
	var sb strings.Builder
	if tab == "clans" {
		sb.WriteString("🏆 <b>برترین کلن‌ها</b> (مجموع لول)\n\n")
		cs, err := b.db.ListClans(10, 0)
		if err != nil || len(cs) == 0 {
			sb.WriteString("هنوز کلنی ساخته نشده.")
		}
		for i, c := range cs {
			fmt.Fprintf(&sb, "%s <b>%s</b> — %s عضو · مجموع لول %s · ☠️ %s\n", medal(i), esc(c.Name),
				game.FaI(c.Members), game.FaI(c.TotalLevel), game.FaI(c.Kills))
		}
		return view{text: sb.String(), kb: kb}
	}
	rows, err := b.db.TopPlayers(tab, 10)
	if err != nil {
		log.Printf("rank: %v", err)
	}
	titles := map[string]string{"level": "🏆 <b>برترین بازیکن‌ها</b> (لول)", "kills": "🏆 <b>بیشترین کشتار</b>", "gold": "🏆 <b>ثروتمندترین‌ها</b>"}
	sb.WriteString(titles[tab] + "\n\n")
	for i, r := range rows {
		var val string
		switch tab {
		case "kills":
			val = "☠️ " + game.FaI(r.Kills)
		case "gold":
			val = "🪙 " + game.Fa(r.Gold)
		default:
			val = "لول " + game.FaI(r.Level) + " · XP " + game.Fa(r.XP)
		}
		fmt.Fprintf(&sb, "%s <b>%s</b> — %s\n", medal(i), esc(r.Name), val)
	}
	if len(rows) == 0 {
		sb.WriteString("هنوز کسی نیست.")
	}
	if rank, err := b.db.PlayerRank(tab, store.RankRow{ID: f.ID, Level: f.Level, XP: f.XP, Kills: f.Kills, Gold: f.Gold}); err == nil {
		fmt.Fprintf(&sb, "\n📍 رتبهٔ تو: <b>%s</b>", game.FaI(rank))
	}
	return view{text: sb.String(), kb: kb}
}

func (b *Bot) cbRank(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	tab := p[0]
	if tab != "kills" && tab != "gold" && tab != "clans" {
		tab = "level"
	}
	return b.viewRank(f, tab)
}

// ---------- clans ----------

func (b *Bot) viewClan(f *game.Fighter) view {
	if f.ClanID == 0 {
		t := fmt.Sprintf("⚔️ <b>کلن</b>\n\nتو عضو هیچ کلنی نیستی.\nساخت کلن: لول %s و %s 🪙 (فعلاً %s 🪙 داری).",
			game.FaI(ClanCreateLevel), game.Fa(ClanCreateCost), game.Fa(f.Gold))
		return view{text: t, kb: tg.Keyboard{
			{{Text: "🆕 ساخت کلن", Data: "cl:new"}, {Text: "📜 لیست کلن‌ها", Data: "cl:list:0"}},
			backMain,
		}}
	}
	c, err := b.db.Clan(f.ClanID)
	if err != nil || c == nil {
		f.ClanID = 0
		return b.viewClan(f)
	}
	members, _ := b.db.ClanMembers(c.ID)
	var sb strings.Builder
	fmt.Fprintf(&sb, "⚔️ <b>کلن %s</b>\n👥 %s/%s عضو · مجموع لول %s · ☠️ %s\n\n", esc(c.Name),
		game.FaI(c.Members), game.FaI(store.ClanMaxMembers), game.FaI(c.TotalLevel), game.FaI(c.Kills))
	for _, m := range members {
		crown := "▫️"
		if m.ID == c.LeaderID {
			crown = "👑"
		}
		fmt.Fprintf(&sb, "%s %s — لول %s\n", crown, esc(m.Name), game.FaI(m.Level))
	}
	var kb tg.Keyboard
	if c.LeaderID == f.ID {
		kb = append(kb, []tg.Button{{Text: "👢 اخراج عضو", Data: "cl:kick"}, {Text: "🗑 انحلال کلن", Data: "cl:dis"}})
	} else {
		kb = append(kb, []tg.Button{{Text: "🚪 خروج از کلن", Data: "cl:leave"}})
	}
	return view{text: sb.String(), kb: append(kb, backMain)}
}

func (b *Bot) viewClanList(page int) view {
	total, _ := b.db.ClanCount()
	if page < 0 || page*clanPageSize >= max(total, 1) {
		page = 0
	}
	cs, _ := b.db.ListClans(clanPageSize, page*clanPageSize)
	var sb strings.Builder
	sb.WriteString("📜 <b>کلن‌ها</b>\n\n")
	if len(cs) == 0 {
		sb.WriteString("هنوز کلنی ساخته نشده.")
	}
	var kb tg.Keyboard
	for _, c := range cs {
		fmt.Fprintf(&sb, "• <b>%s</b> — %s/%s عضو · مجموع لول %s\n", esc(c.Name), game.FaI(c.Members),
			game.FaI(store.ClanMaxMembers), game.FaI(c.TotalLevel))
		kb = append(kb, []tg.Button{{Text: "➕ پیوستن به " + c.Name, Data: "cl:j:" + strconv.FormatInt(c.ID, 10)}})
	}
	var nav []tg.Button
	if page > 0 {
		nav = append(nav, tg.Button{Text: "◀️", Data: fmt.Sprintf("cl:list:%d", page-1)})
	}
	if (page+1)*clanPageSize < total {
		nav = append(nav, tg.Button{Text: "▶️", Data: fmt.Sprintf("cl:list:%d", page+1)})
	}
	if nav != nil {
		kb = append(kb, nav)
	}
	return view{text: sb.String(), kb: append(kb, []tg.Button{{Text: "🔙 بازگشت", Data: "cl:0"}})}
}

func (b *Bot) cacheClan(id, clan int64) {
	if cf, ok := b.cache[id]; ok {
		cf.ClanID = clan
	}
}

func (b *Bot) cbClan(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	if f == nil {
		return b.viewStart(nil)
	}
	fail := func(msg string) view { return view{toast: msg, alert: true, keep: true} }
	var c *store.Clan
	if f.ClanID != 0 {
		c, _ = b.db.Clan(f.ClanID)
	}
	switch p[0] {
	case "new":
		switch {
		case f.ClanID != 0:
			return fail("تو الان تو یه کلن هستی")
		case f.Level < ClanCreateLevel:
			return fail(fmt.Sprintf("برای ساخت کلن لول %s لازمه", game.FaI(ClanCreateLevel)))
		case f.Gold < ClanCreateCost:
			return fail("طلای کافی نداری")
		}
		b.awaiting[f.ID] = "clan"
		return view{text: fmt.Sprintf("🆕 اسم کلن رو تو یه پیام بفرست (%s تا %s حرف).",
			game.FaI(store.ClanMinName), game.FaI(store.ClanMaxName)),
			kb: tg.Keyboard{{{Text: "❌ لغو", Data: "cl:0"}}}}
	case "list":
		page := 0
		if len(p) > 1 {
			page, _ = strconv.Atoi(p[1])
		}
		return b.viewClanList(page)
	case "j":
		if f.ClanID != 0 || len(p) < 2 {
			return fail("تو الان تو یه کلن هستی")
		}
		id, _ := strconv.ParseInt(p[1], 10, 64)
		if err := b.db.JoinClan(f.ID, id); err != nil {
			return fail(err.Error())
		}
		f.ClanID = id
		v := b.viewClan(f)
		v.toast = "✅ به کلن پیوستی"
		return v
	case "leave":
		if c != nil && c.LeaderID == f.ID {
			return fail("رهبر نمی‌تونه خارج بشه؛ کلن رو منحل کن")
		}
		if err := b.db.LeaveClan(f.ID); err != nil {
			return fail("خطا")
		}
		f.ClanID = 0
	case "kick":
		if c == nil || c.LeaderID != f.ID {
			return fail("فقط رهبر")
		}
		ms, _ := b.db.ClanMembers(c.ID)
		var kb tg.Keyboard
		for _, m := range ms {
			if m.ID != f.ID {
				kb = append(kb, []tg.Button{{Text: "👢 " + m.Name, Data: "cl:k:" + strconv.FormatInt(m.ID, 10)}})
			}
		}
		if kb == nil {
			return fail("عضوی برای اخراج نیست")
		}
		return view{text: "👢 کدوم عضو اخراج بشه؟", kb: append(kb, []tg.Button{{Text: "🔙 بازگشت", Data: "cl:0"}})}
	case "k":
		if c == nil || c.LeaderID != f.ID || len(p) < 2 {
			return fail("فقط رهبر")
		}
		id, _ := strconv.ParseInt(p[1], 10, 64)
		if id == f.ID {
			return fail("خودت رو نمی‌تونی اخراج کنی")
		}
		ms, _ := b.db.ClanMembers(c.ID)
		for _, m := range ms {
			if m.ID == id {
				b.db.LeaveClan(id)
				b.cacheClan(id, 0)
				v := b.viewClan(f)
				v.toast = "👢 اخراج شد"
				return v
			}
		}
		return fail("این عضو دیگه تو کلن نیست")
	case "dis":
		if c == nil || c.LeaderID != f.ID {
			return fail("فقط رهبر")
		}
		return view{text: fmt.Sprintf("🗑 کلن <b>%s</b> برای همیشه منحل بشه؟", esc(c.Name)), kb: tg.Keyboard{
			{{Text: "✅ بله، منحل کن", Data: "cl:dis2", Style: "danger"}, {Text: "❌ نه", Data: "cl:0", Style: "success"}}}}
	case "dis2":
		if c == nil || c.LeaderID != f.ID {
			return fail("فقط رهبر")
		}
		ids, err := b.db.DeleteClan(c.ID)
		if err != nil {
			return fail("خطا")
		}
		for _, id := range ids {
			b.cacheClan(id, 0)
		}
		f.ClanID = 0
		v := b.viewClan(f)
		v.toast = "کلن منحل شد"
		return v
	}
	return b.viewClan(f)
}

// onPrivateText handles free-text input the menus are waiting for.
func (b *Bot) onPrivateText(m *tg.Message, text string) {
	if b.awaiting[m.From.ID] != "clan" {
		return
	}
	f := b.fighter(m.From.ID)
	if f == nil {
		return
	}
	reply := func(s string) { b.send(m.Chat.ID, s, nil, m.MessageID) }
	name := strings.Join(strings.Fields(text), " ")
	switch n := utf8.RuneCountInString(name); {
	case n < store.ClanMinName || n > store.ClanMaxName:
		reply(fmt.Sprintf("⚠️ اسم باید %s تا %s حرف باشه؛ دوباره بفرست.", game.FaI(store.ClanMinName), game.FaI(store.ClanMaxName)))
		return
	case f.ClanID != 0 || f.Level < ClanCreateLevel || f.Gold < ClanCreateCost:
		delete(b.awaiting, f.ID)
		reply("الان نمی‌تونی کلن بسازی.")
		return
	}
	id, err := b.db.CreateClan(name, f.ID, b.now())
	if err != nil {
		reply("⚠️ " + esc(err.Error()) + "؛ یه اسم دیگه بفرست.")
		return
	}
	delete(b.awaiting, f.ID)
	f.ClanID = id
	f.Gold -= ClanCreateCost
	b.save(f)
	v := b.viewClan(f)
	b.send(m.Chat.ID, "✅ کلن ساخته شد!\n\n"+v.text, v.kb, 0)
}
