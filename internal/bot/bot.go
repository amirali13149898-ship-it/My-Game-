// Package bot wires the Telegram client, the database and the game engine.
package bot

import (
	"context"
	"fmt"
	"html"
	"image"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ardian/internal/game"
	"ardian/internal/store"
	"ardian/internal/tg"
)

type Config struct {
	Token       string
	AdminID     int64
	BackupDir   string
	BackupEvery time.Duration // 0 disables backups
	BackupKeep  int
	AssetsDir   string // folder with <key>.jpg|png pictures (optional)
}

type Bot struct {
	cfg         Config
	tg          *tg.Client
	db          *store.DB
	eng         *game.Engine
	now         func() time.Time
	cache       map[int64]*game.Fighter
	drafts      map[int64]*[game.NumStats]int // pending (unconfirmed) stat allocation
	lastBtn     map[int64]time.Time
	lastHunt    map[int64]time.Time
	awaiting    map[int64]string // pending free-text input (e.g. clan name)
	assets      map[string]asset
	seen        map[int64]time.Time // last activity per cached player (for eviction)
	textCache   map[string]string   // texts are read from the DB once
	photoCache  map[string]photoRow // picture lookups are read from the DB once
	lastGC      time.Time
	cardBase    image.Image
	cardCache   map[string][]byte
	dungeons    map[int64]*game.DungeonRun
	lastDungeon map[int64]time.Time
}

func New(cfg Config, db *store.DB) *Bot {
	return &Bot{
		cfg: cfg, tg: tg.New(cfg.Token), db: db, eng: game.NewEngine(), now: time.Now,
		cache: map[int64]*game.Fighter{}, drafts: map[int64]*[game.NumStats]int{},
		lastBtn: map[int64]time.Time{}, lastHunt: map[int64]time.Time{}, awaiting: map[int64]string{},
		assets: map[string]asset{}, seen: map[int64]time.Time{}, textCache: map[string]string{}, cardCache: map[string][]byte{}, photoCache: map[string]photoRow{}, dungeons: map[int64]*game.DungeonRun{}, lastDungeon: map[int64]time.Time{},
	}
}

// Run long-polls Telegram until ctx is cancelled. Updates are handled one at a
// time, so game state needs no locking.
func (b *Bot) Run(ctx context.Context) error {
	if err := b.tg.Init(); err != nil {
		return fmt.Errorf("getMe: %w", err)
	}
	log.Printf("bot @%s started", b.tg.Me.Username)
	b.loadAssets()
	b.loadCardBase()
	go b.backupLoop(ctx)
	offset := 0
	for ctx.Err() == nil {
		ups, err := b.tg.GetUpdates(offset, 50)
		if err != nil {
			log.Printf("getUpdates: %v", err)
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
			}
			continue
		}
		for _, u := range ups {
			offset = u.UpdateID + 1
			b.safeHandle(u)
		}
		b.gc()
	}
	return nil
}

const (
	idleEvict = time.Hour
	gcEvery   = 10 * time.Minute
)

// gc frees memory of idle players. Everything evicted is already persisted
// (state is saved after every action) and is reloaded from the DB on demand.
func (b *Bot) gc() {
	now := b.now()
	if now.Sub(b.lastGC) < gcEvery {
		return
	}
	b.lastGC = now
	for id, t := range b.seen {
		if now.Sub(t) < idleEvict || b.dungeons[id] != nil {
			continue
		}
		delete(b.seen, id)
		delete(b.cache, id)
		delete(b.drafts, id)
		delete(b.lastHunt, id)
		delete(b.lastDungeon, id)
		delete(b.awaiting, id)
	}
	for id, t := range b.lastBtn {
		if now.Sub(t) > time.Minute {
			delete(b.lastBtn, id)
		}
	}
}

func (b *Bot) safeHandle(u tg.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic handling update %d: %v", u.UpdateID, r)
		}
	}()
	switch {
	case u.CallbackQuery != nil:
		b.onCallback(u.CallbackQuery)
	case u.Message != nil:
		b.onMessage(u.Message)
	}
}

// ---------- state helpers ----------

func (b *Bot) fighter(id int64) *game.Fighter {
	b.seen[id] = b.now()
	if f, ok := b.cache[id]; ok {
		return f
	}
	f, err := b.db.LoadFighter(id)
	if err != nil {
		log.Printf("load %d: %v", id, err)
		return nil
	}
	if f != nil {
		b.cache[id] = f
	}
	return f
}

func (b *Bot) save(fs ...*game.Fighter) {
	for _, f := range fs {
		if err := b.db.SaveFighter(f); err != nil {
			log.Printf("save %d: %v", f.ID, err)
		}
	}
}

func esc(s string) string { return html.EscapeString(s) }

func displayName(u *tg.User) string {
	n := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if n == "" {
		n = u.Username
	}
	if n == "" {
		n = fmt.Sprint(u.ID)
	}
	if r := []rune(n); len(r) > 24 {
		n = string(r[:24])
	}
	return n
}

// ---------- messages ----------

func (b *Bot) onMessage(m *tg.Message) {
	if m.From == nil || m.From.IsBot {
		return
	}
	if len(m.Photo) > 0 && m.Chat.Type == "private" && m.From.ID == b.cfg.AdminID &&
		strings.HasPrefix(m.Caption, "/setphoto") {
		b.adminPhoto(m, "setphoto", strings.TrimSpace(strings.TrimPrefix(m.Caption, "/setphoto")))
		return
	}
	if m.Text == "" {
		return
	}
	text := strings.TrimSpace(m.Text)
	if strings.HasPrefix(text, "/") {
		b.onCommand(m, text)
		return
	}
	if m.Chat.Type != "private" {
		b.onGroupText(m, text)
		return
	}
	b.onPrivateText(m, text)
}

func (b *Bot) onCommand(m *tg.Message, text string) {
	tok, rest := text, ""
	if i := strings.IndexAny(text, " \n"); i >= 0 {
		tok, rest = text[:i], strings.TrimSpace(text[i+1:])
	}
	cmd := strings.ToLower(tok[1:])
	if at := strings.Index(cmd, "@"); at >= 0 {
		if !strings.EqualFold(cmd[at+1:], b.tg.Me.Username) {
			return // addressed to another bot
		}
		cmd = cmd[:at]
	}
	switch cmd {
	case "start", "menu":
		b.cmdStart(m)
	case "help":
		b.send(m.Chat.ID, b.text("help"), nil, m.MessageID)
	case "setphoto", "photos":
		if m.From.ID == b.cfg.AdminID && m.Chat.Type == "private" {
			b.adminPhoto(m, cmd, rest)
		}
	case "settext", "gettext":
		if m.From.ID == b.cfg.AdminID && m.Chat.Type == "private" {
			b.adminText(m, cmd, rest)
		}
	}
}

func (b *Bot) send(chat int64, text string, kb tg.Keyboard, replyTo int) {
	if _, err := b.tg.Send(chat, text, tg.SendOpts{ReplyTo: replyTo, Keyboard: kb}); err != nil {
		log.Printf("send to %d: %v", chat, err)
	}
}

func (b *Bot) cmdStart(m *tg.Message) {
	if m.Chat.Type != "private" {
		kb := tg.Keyboard{{{Text: "🎮 باز کردن بات", URL: "https://t.me/" + b.tg.Me.Username + "?start=g"}}}
		b.send(m.Chat.ID, "برای ساخت کارکتر و دیدن منو، بات رو تو پیوی باز کن 👇", kb, m.MessageID)
		return
	}
	f := b.fighter(m.From.ID)
	b.sendView(m.Chat.ID, b.viewStart(f))
}

// ---------- callbacks ----------

// view is what a callback handler wants shown.
type view struct {
	text  string
	kb    tg.Keyboard
	toast string
	alert bool
	keep  bool      // only answer the callback, do not touch the message
	photo string    // picture key (optional)
	card  *cardData // numbers drawn on the stats base picture (optional)
}

func (b *Bot) onCallback(cb *tg.CallbackQuery) {
	if cb.Message == nil || cb.Message.Chat.Type != "private" {
		b.tg.Answer(cb.ID, "منو فقط تو پیوی بات کار می‌کنه", true)
		return
	}
	now := b.now()
	if now.Sub(b.lastBtn[cb.From.ID]) < 700*time.Millisecond { // anti-spam
		b.tg.Answer(cb.ID, "", false)
		return
	}
	b.lastBtn[cb.From.ID] = now
	delete(b.awaiting, cb.From.ID) // leaving a prompt cancels it

	p := strings.Split(cb.Data, ":")
	var v view
	switch p[0] {
	case "c":
		v = b.cbCreate(&cb.From, p[1:])
	case "go":
		v = b.cbNav(&cb.From, p[1:])
	case "st":
		v = b.cbStats(&cb.From, p[1:])
	case "ab":
		v = b.cbAbilities(&cb.From, p[1:])
	case "sh":
		v = b.cbShop(&cb.From, p[1:])
	case "eq":
		v = b.cbEquip(&cb.From, p[1:])
	case "rk":
		v = b.cbRank(&cb.From, p[1:])
	case "cl":
		v = b.cbClan(&cb.From, p[1:])
	case "hu":
		v = b.cbHunt(&cb.From, p[1:])
	case "dg":
		v = b.cbDungeon(&cb.From, p[1:])
	default:
		v = view{keep: true}
	}
	if err := b.tg.Answer(cb.ID, v.toast, v.alert); err != nil {
		log.Printf("answer: %v", err)
	}
	if v.keep || v.text == "" {
		return
	}
	if v.photo == "" {
		v.photo = sectionPhoto[p[0]]
	}
	b.present(cb.Message, v)
}

// default picture of each menu section (views may override with their own key)
var sectionPhoto = map[string]string{"st": "stats", "ab": "abilities", "sh": "shop", "eq": "equip",
	"rk": "rank", "cl": "clan", "hu": "hunt", "dg": "dungeon", "c": "create"}

func (b *Bot) cbNav(u *tg.User, p []string) view {
	f := b.fighter(u.ID)
	switch p[0] {
	case "start":
		return b.viewStart(f)
	case "help":
		return view{text: b.text("help"), photo: "help", kb: tg.Keyboard{{{Text: "🔙 بازگشت", Data: "go:start"}}}}
	case "main":
		if f == nil {
			return b.viewStart(nil)
		}
		return b.viewMain(f)
	case "soon":
		return view{toast: "🚧 این بخش به‌زودی اضافه می‌شه", keep: true}
	}
	return view{keep: true}
}

// ---------- backups ----------

func (b *Bot) backupLoop(ctx context.Context) {
	if b.cfg.BackupEvery <= 0 {
		return
	}
	if !b.db.CanBackup() {
		log.Printf("backups: skipped (Postgres/Supabase keeps its own backups)")
		return
	}
	t := time.NewTicker(b.cfg.BackupEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.backupOnce()
		}
	}
}

func (b *Bot) backupOnce() {
	if err := os.MkdirAll(b.cfg.BackupDir, 0o755); err != nil {
		log.Printf("backup mkdir: %v", err)
		return
	}
	dest := filepath.Join(b.cfg.BackupDir, "ardian-"+time.Now().UTC().Format("20060102-150405")+".db")
	if err := b.db.Backup(dest); err != nil {
		log.Printf("backup: %v", err)
		return
	}
	log.Printf("backup written: %s", dest)
	if b.cfg.AdminID != 0 { // off-site copy: the Railway volume is not a backup by itself
		if err := b.tg.SendDocument(b.cfg.AdminID, dest, "💾 بکاپ دیتابیس"); err != nil {
			log.Printf("backup upload: %v", err)
		}
	}
	files, _ := filepath.Glob(filepath.Join(b.cfg.BackupDir, "ardian-*.db"))
	sort.Strings(files)
	keep := b.cfg.BackupKeep
	if keep < 1 {
		keep = 10
	}
	for len(files) > keep {
		os.Remove(files[0])
		files = files[1:]
	}
}
