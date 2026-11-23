package bot

import (
	"testing"
	"time"

	"ardian/internal/game"
)

func TestStatsCaptionFitsPhoto(t *testing.T) {
	b := newTestBot()
	f := game.NewFighter(1, "علی", game.Human, false, []game.Element{game.Fire, game.Water}, time.Now())
	f.Points = 20
	b.cache[1] = f
	v := b.viewStats(f)
	t.Log(v.text)
	if !captionFits(v.text) {
		t.Fatal("stats caption too long for a photo")
	}
	if !captionFits(b.viewMain(f).text) {
		t.Fatal("main caption too long")
	}
}

func TestMainMenuUsesCard(t *testing.T) {
	b := newTestBot()
	b.cardBase = loadBase(t)
	f := game.NewFighter(1, "꧁ MADARA UCHIHA ꧂", game.Human, false, []game.Element{game.Fire, game.Wind}, time.Now())
	b.cache[1] = f
	v := b.viewMain(f)
	if v.card == nil || v.card.Name != "MADARA UCHIHA" {
		t.Fatalf("card missing: %+v", v.card)
	}
	if b.viewStats(f).card != nil {
		t.Fatal("stats view must not use the card")
	}
	if !captionFits(v.text) {
		t.Fatal("caption too long")
	}
}
