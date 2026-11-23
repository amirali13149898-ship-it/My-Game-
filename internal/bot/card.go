package bot

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The stats card is a base picture (ASSETS_DIR/card_stats.jpg) with numbers
// drawn on it. Only ASCII text and digits are drawn (no Persian shaping
// needed): the labels are part of the picture itself. Glyphs come from a
// pre-rendered atlas, so this uses only the standard library.

//go:embed glyphs/*
var glyphFS embed.FS

type fontMeta struct {
	CW, CH, Cols, Asc int
	Adv               []int
}

type atlasFont struct {
	size int
	meta fontMeta
	mask *image.Alpha
}

var fonts []*atlasFont // sorted by size ascending

func init() {
	raw, err := glyphFS.ReadFile("glyphs/meta.json")
	if err != nil {
		panic(err)
	}
	var metas map[int]fontMeta
	if err := json.Unmarshal(raw, &metas); err != nil {
		panic(err)
	}
	for size, m := range metas {
		data, err := glyphFS.ReadFile(fmt.Sprintf("glyphs/g%d.png", size))
		if err != nil {
			panic(err)
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			panic(err)
		}
		a := image.NewAlpha(img.Bounds())
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				r, _, _, _ := img.At(x, y).RGBA()
				a.SetAlpha(x, y, color.Alpha{uint8(r >> 8)})
			}
		}
		fonts = append(fonts, &atlasFont{size, m, a})
	}
	sort.Slice(fonts, func(i, j int) bool { return fonts[i].size < fonts[j].size })
}

func (f *atlasFont) width(s string) int {
	w := 0
	for _, r := range s {
		if r >= 32 && r < 127 {
			w += f.meta.Adv[r-32]
		}
	}
	return w
}

func (f *atlasFont) draw(dst draw.Image, s string, x, top int, col color.Color) {
	for _, r := range s {
		if r < 32 || r >= 127 {
			continue
		}
		i := int(r - 32)
		cx, cy := (i%f.meta.Cols)*f.meta.CW, (i/f.meta.Cols)*f.meta.CH
		dr := image.Rect(x-2, top, x-2+f.meta.CW, top+f.meta.CH)
		draw.DrawMask(dst, dr, image.NewUniform(col), image.Point{}, f.mask, image.Point{cx, cy}, draw.Over)
		x += f.meta.Adv[i]
	}
}

// fit returns the biggest font (not above maxSize) in which s fits maxW.
func fitFont(s string, maxW, maxSize int) *atlasFont {
	best := fonts[0]
	for _, f := range fonts {
		if f.size <= maxSize && f.width(s) <= maxW {
			best = f
		}
	}
	return best
}

// ---------- layout (pixels of the 800-wide base picture) ----------

type box struct{ cx, cy, w, h int } // centre and size of a text area

type barBox struct{ x0, y0, x1, y1 int }

var cardLayout = struct {
	Name, Level, Gold box
	HP, Mana          barBox
}{
	Name:  box{402, 160, 150, 60},
	Level: box{577, 160, 104, 60},
	Gold:  box{229, 160, 104, 60},
	HP:    barBox{199, 239, 603, 282},
	Mana:  barBox{199, 317, 603, 364},
}

var (
	textCol   = color.NRGBA{250, 236, 200, 255}
	shadowCol = color.NRGBA{20, 8, 0, 235}
)

type cardData struct {
	Name          string
	Level         int
	Gold          int64
	HP, MaxHP     float64
	Mana, MaxMana float64
}

func (c cardData) key() string {
	return fmt.Sprintf("%s|%d|%d|%.0f|%.0f|%.0f|%.0f", c.Name, c.Level, c.Gold, c.HP, c.MaxHP, c.Mana, c.MaxMana)
}

// cardName keeps only the printable ASCII part of a name (decorations and
// Persian letters cannot be drawn), e.g. "꧁ MADARA ꧂" → "MADARA".
func cardName(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 {
			sb.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// asciiName reports whether the name can be drawn with the built-in glyphs.
func asciiName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 32 || r >= 127 {
			return false
		}
	}
	return true
}

func compact(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1e9)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 100_000:
		return fmt.Sprintf("%.0fK", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

func drawIn(dst draw.Image, b box, s string, maxSize int) {
	f := fitFont(s, b.w, maxSize)
	w := f.width(s)
	x := b.cx - w/2
	top := b.cy - f.meta.CH/2 + 2
	f.draw(dst, s, x+2, top+2, shadowCol)
	f.draw(dst, s, x, top, textCol)
}

func drawBar(dst draw.Image, b barBox, cur, max float64, label string) {
	if max <= 0 {
		max = 1
	}
	frac := cur / max
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	// darken the empty (right) part of the bar
	cut := b.x0 + int(float64(b.x1-b.x0)*frac+0.5)
	if cut < b.x1 {
		draw.Draw(dst, image.Rect(cut, b.y0, b.x1, b.y1), image.NewUniform(color.NRGBA{0, 0, 0, 175}), image.Point{}, draw.Over)
	}
	drawIn(dst, box{(b.x0 + b.x1) / 2, (b.y0 + b.y1) / 2, b.x1 - b.x0 - 20, b.y1 - b.y0}, label, 34)
}

// renderCard draws c onto a copy of base and returns the JPEG.
func renderCard(base image.Image, c cardData) ([]byte, error) {
	rgba := image.NewRGBA(base.Bounds())
	draw.Draw(rgba, rgba.Bounds(), base, base.Bounds().Min, draw.Src)
	if c.Name != "" {
		n := c.Name
		if fitFont(n, cardLayout.Name.w, 34).width(n) > cardLayout.Name.w {
			for len(n) > 1 && fitFont(n+"..", cardLayout.Name.w, 34).width(n+"..") > cardLayout.Name.w {
				n = n[:len(n)-1]
			}
			n += ".."
		}
		drawIn(rgba, cardLayout.Name, n, 34)
	}
	drawIn(rgba, cardLayout.Level, fmt.Sprint(c.Level), 56)
	drawIn(rgba, cardLayout.Gold, compact(c.Gold), 44)
	drawBar(rgba, cardLayout.HP, c.HP, c.MaxHP, fmt.Sprintf("%.0f / %.0f", c.HP, c.MaxHP))
	drawBar(rgba, cardLayout.Mana, c.Mana, c.MaxMana, fmt.Sprintf("%.0f / %.0f", c.Mana, c.MaxMana))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---------- bot side ----------

const cardBaseFile = "card_stats.jpg"

func (b *Bot) loadCardBase() {
	b.cardBase = nil
	if b.cfg.AssetsDir == "" {
		return
	}
	f, err := os.Open(filepath.Join(b.cfg.AssetsDir, cardBaseFile))
	if err != nil {
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		log.Printf("card base: %v", err)
		return
	}
	b.cardBase = img
}

// cardImage returns the rendered picture for a card (nil → send plain text).
func (b *Bot) cardImage(c *cardData) []byte {
	if c == nil || b.cardBase == nil {
		return nil
	}
	k := c.key()
	if v, ok := b.cardCache[k]; ok {
		return v
	}
	data, err := renderCard(b.cardBase, *c)
	if err != nil {
		log.Printf("render card: %v", err)
		return nil
	}
	if len(b.cardCache) >= 16 {
		b.cardCache = map[string][]byte{}
	}
	b.cardCache[k] = data
	return data
}
