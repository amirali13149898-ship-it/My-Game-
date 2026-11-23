package bot

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"ardian/internal/tg"
)

// Default texts; the admin can override each one with /settext (stored in DB).
var defaultTexts = map[string]string{
	"welcome": "✦ <b>⚔️  به اردیان خوش اومدی!</b> ✦\n" + divider + "\n\n" +
		quote("اینجا یه بازی فانتزی PvP داخل گروه‌هاست. مهم نیست چه کارکتری هستی، مهم اینه چیکار کنی.") +
		"\n🛡 نژادت رو انتخاب کن، قوی شو و حریفات رو شکست بده!\n\n👇 <b>برای شروع یه کارکتر انتخاب کن</b>",
	"help": "✦ <b>📖  راهنما</b> ✦\n" + divider + "\n\n" +
		"⚔️ <b>حمله</b>\n" + quote("تو گروه روی پیام حریف ریپلای کن و فقط اسم ابیلیتی رو بنویس (مثلاً <code>فایربال</code>). برای حملهٔ فیزیکی بنویس <code>حمله</code>.") + "\n" +
		"✨ <b>جادوهای پشتیبانی</b>\nبدون ریپلای بنویس؛ روی خودت اثر می‌ذارن.\n\n" +
		"🧬 <b>توانایی نژاد</b>\nمثل کیوبی یا زتسو؛ با نوشتن اسمش فعال می‌شه.\n\n" +
		"⏱ <b>کولداون</b>\nبین هر دو حمله ۳ ثانیه و بین حملهٔ فیزیکی ۵ ثانیه صبر کن؛ هر جادو کولداون خودش رو داره.\n\n" +
		"📈 <b>پیشرفت</b>\nهر لول‌آپ ۵ امتیاز استت می‌ده؛ جادوها با استفاده ارتقا پیدا می‌کنن. استت‌ها رو می‌شه با طلا ریست کرد.\n\n" +
		"🐗🏯 <b>شکار و دانجن</b>\nبا هیولاها می‌جنگی و طلا، XP و آیتم می‌گیری.\n\n" +
		"💀 <b>مرگ</b>\n۵ دقیقه مُرده‌ای و ۱۰٪ طلا و ۲۵٪ XPِ لول فعلی رو از دست می‌دی.\n\n" +
		"⚠️ <i>حریف باید بازی رو استارت کرده باشه و بات تو گروه باشه.</i>",
}

func (b *Bot) text(key string) string {
	if v, ok := b.textCache[key]; ok {
		return v
	}
	v := defaultTexts[key]
	if t, ok, err := b.db.Text(key); err != nil {
		log.Printf("text %s: %v", key, err)
		return v // do not cache a failed lookup
	} else if ok {
		v = t
	}
	b.textCache[key] = v
	return v
}

func textKeys() string {
	ks := make([]string, 0, len(defaultTexts))
	for k := range defaultTexts {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return strings.Join(ks, ", ")
}

// adminText implements /settext and /gettext (admin only, private chat).
func (b *Bot) adminText(m *tg.Message, cmd, rest string) {
	reply := func(s string) { b.send(m.Chat.ID, s, nil, m.MessageID) }
	key, val := rest, ""
	if i := strings.IndexAny(rest, " \n"); i >= 0 {
		key, val = rest[:i], strings.TrimSpace(rest[i+1:])
	}
	if _, ok := defaultTexts[key]; !ok {
		reply("کلید معتبر نیست. کلیدها: <code>" + textKeys() + "</code>")
		return
	}
	if cmd == "gettext" {
		reply("<pre>" + esc(b.text(key)) + "</pre>")
		return
	}
	if val == "" {
		reply("فرمت: <code>/settext " + key + " متن جدید</code>")
		return
	}
	old, had, _ := b.db.Text(key)
	delete(b.textCache, key)
	if err := b.db.SetText(key, val); err != nil {
		reply("❌ ذخیره نشد: " + esc(err.Error()))
		return
	}
	// The text is HTML: verify Telegram accepts it, otherwise roll back.
	if _, err := b.tg.Send(m.Chat.ID, val, tg.SendOpts{}); err != nil {
		if had {
			b.db.SetText(key, old)
		} else {
			b.db.Exec(`DELETE FROM texts WHERE key=?`, key)
		}
		delete(b.textCache, key)
		reply(fmt.Sprintf("❌ متن HTML معتبر نیست و برگردونده شد:\n<code>%s</code>", esc(err.Error())))
		return
	}
	reply("✅ ذخیره شد (پیش‌نمایش بالاست)")
}
