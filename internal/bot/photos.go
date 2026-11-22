package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"

	"ardian/internal/tg"
)

// Pictures are plain images WITHOUT numbers (art only); the bot writes every
// number in the caption. Every picture has a key. A picture comes either from
//   - a file ASSETS_DIR/<key>.jpg|png (uploaded once, then cached by content
//     hash + Telegram file_id, re-uploaded only when the file changes), or
//   - the admin sending the photo with the caption "/setphoto <key>".
// Without a picture the view is simply text, so the game works with no art.

var photoKeys = func() []string {
	ks := []string{"welcome", "help", "create", "main", "stats", "abilities", "equip", "shop", "clan", "rank", "hunt", "dungeon"}
	for r := 0; r < 9; r++ {
		ks = append(ks, fmt.Sprintf("race%d", r))
	}
	return ks
}()

func validPhotoKey(k string) bool {
	for _, x := range photoKeys {
		if x == k {
			return true
		}
	}
	return false
}

type asset struct{ path, hash string }

type photoRow struct {
	fid, hash string
	ok        bool
}

const captionLimit = 1024 // UTF-16 code units

func captionFits(s string) bool { return len(utf16.Encode([]rune(s))) <= captionLimit }

func (b *Bot) loadAssets() {
	b.assets = map[string]asset{}
	if b.cfg.AssetsDir == "" {
		return
	}
	for _, ext := range []string{"jpg", "jpeg", "png"} {
		files, _ := filepath.Glob(filepath.Join(b.cfg.AssetsDir, "*."+ext))
		for _, f := range files {
			key := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
			if strings.HasPrefix(key, "card_") { // base pictures for drawn cards, loaded separately
				continue
			}
			if !validPhotoKey(key) {
				log.Printf("assets: ignoring %s (unknown key; valid: %s)", f, strings.Join(photoKeys, ", "))
				continue
			}
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			sum := sha256.Sum256(data)
			b.assets[key] = asset{path: f, hash: hex.EncodeToString(sum[:])}
		}
	}
	log.Printf("assets: %d pictures found", len(b.assets))
}

// photoRef resolves a key to a cached file_id or a local file to upload.
func (b *Bot) photoRef(key string) (fileID, path string) {
	if key == "" {
		return
	}
	a, hasAsset := b.assets[key]
	row, cached := b.photoCache[key]
	if !cached {
		fid, hash, ok, err := b.db.Photo(key)
		if err != nil {
			log.Printf("photo %s: %v", key, err)
		} else {
			row = photoRow{fid, hash, ok}
			b.photoCache[key] = row // one DB lookup per key, not one per menu render
		}
	}
	fid, hash, ok := row.fid, row.hash, row.ok
	if ok && (!hasAsset || hash == "" || hash == a.hash) {
		return fid, ""
	}
	if hasAsset {
		return "", a.path
	}
	return "", ""
}

// sendView sends a new message for a view (photo + caption when available).
func (b *Bot) sendView(chat int64, v view) {
	if img := b.cardImage(v.card); img != nil && captionFits(v.text) {
		if _, err := b.tg.SendPhotoBytes(chat, img, v.text, v.kb); err == nil {
			return
		} else {
			log.Printf("sendPhoto card: %v (falling back)", err)
		}
	}
	if fid, path := b.photoRef(v.photo); (fid != "" || path != "") && captionFits(v.text) {
		msg, err := b.tg.SendPhoto(chat, fid, path, v.text, v.kb)
		if err == nil {
			if path != "" {
				b.rememberUpload(v.photo, msg)
			}
			return
		}
		log.Printf("sendPhoto %s: %v (falling back to text)", v.photo, err)
	}
	b.send(chat, v.text, v.kb, 0)
}

func (b *Bot) rememberUpload(key string, msg *tg.Message) {
	if fid := msg.BestPhoto(); fid != "" {
		if err := b.db.SetPhoto(key, fid, b.assets[key].hash); err != nil {
			log.Printf("save photo %s: %v", key, err)
		}
		delete(b.photoCache, key)
	}
}

// present replaces an existing menu message with a view. Text → text and
// photo → photo are edited in place; switching between them sends a new
// message and deletes the old one.
func (b *Bot) present(old *tg.Message, v view) {
	chat := old.Chat.ID
	if img := b.cardImage(v.card); img != nil && captionFits(v.text) {
		if len(old.Photo) > 0 {
			if err := b.tg.EditPhotoBytes(chat, old.MessageID, img, v.text, v.kb); err != nil {
				log.Printf("present card: %v", err)
			}
			return
		}
		b.sendView(chat, v)
		if derr := b.tg.Delete(chat, old.MessageID); derr != nil {
			log.Printf("delete: %v", derr)
		}
		return
	}
	fid, path := b.photoRef(v.photo)
	wantPhoto := (fid != "" || path != "") && captionFits(v.text)
	hasPhoto := len(old.Photo) > 0
	var err error
	switch {
	case !wantPhoto && !hasPhoto:
		err = b.tg.Edit(chat, old.MessageID, v.text, v.kb)
	case wantPhoto && hasPhoto && path == "":
		err = b.tg.EditPhoto(chat, old.MessageID, fid, v.text, v.kb)
	default:
		b.sendView(chat, v)
		if derr := b.tg.Delete(chat, old.MessageID); derr != nil {
			log.Printf("delete: %v", derr)
		}
	}
	if err != nil {
		log.Printf("present: %v", err)
	}
}

// ---------- admin ----------

// adminPhoto handles "/setphoto <key>" (photo caption, or a reply to a photo)
// and "/photos".
func (b *Bot) adminPhoto(m *tg.Message, cmd, rest string) {
	reply := func(s string) { b.send(m.Chat.ID, s, nil, m.MessageID) }
	if cmd == "photos" {
		cached, _ := b.db.PhotoKeys()
		set := map[string]bool{}
		for _, k := range cached {
			set[k] = true
		}
		var sb strings.Builder
		sb.WriteString("🖼 <b>عکس‌ها</b>\n")
		keys := append([]string(nil), photoKeys...)
		sort.Strings(keys)
		for _, k := range keys {
			st := "—"
			switch _, hasAsset := b.assets[k]; {
			case set[k]:
				st = "✅"
			case hasAsset:
				st = "📁 (آپلود می‌شه)"
			}
			fmt.Fprintf(&sb, "<code>%s</code> %s\n", k, st)
		}
		sb.WriteString("\nبرای ست کردن: عکس رو با کپشن <code>/setphoto کلید</code> بفرست.")
		reply(sb.String())
		return
	}
	key := strings.Fields(rest + " ")[0]
	if !validPhotoKey(key) {
		reply("کلید معتبر نیست. کلیدها: <code>" + strings.Join(photoKeys, ", ") + "</code>")
		return
	}
	fid := m.BestPhoto()
	if fid == "" && m.ReplyToMessage != nil {
		fid = m.ReplyToMessage.BestPhoto()
	}
	if fid == "" {
		reply("عکس رو همراه کپشن /setphoto بفرست یا روی یه عکس ریپلای کن.")
		return
	}
	if err := b.db.SetPhoto(key, fid, ""); err != nil {
		reply("❌ " + esc(err.Error()))
		return
	}
	delete(b.photoCache, key)
	reply("✅ عکس <code>" + key + "</code> ذخیره شد.")
}
