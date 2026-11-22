package bot

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"
)

func loadBase(t *testing.T) image.Image {
	f, err := os.Open("../../assets/card_stats.jpg")
	if err != nil {
		t.Skip("no base picture")
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestRenderCard(t *testing.T) {
	base := loadBase(t)
	data, err := renderCard(base, cardData{Name: "Ardian_The_Great_Warrior", Level: 27, Gold: 1234567, HP: 640, MaxHP: 1000, Mana: 120, MaxMana: 500})
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytesReader(data))
	if err != nil || img.Bounds() != base.Bounds() {
		t.Fatalf("bad output: %v %v", err, img.Bounds())
	}
	if out := os.Getenv("CARD_OUT"); out != "" {
		os.WriteFile(out, data, 0o644)
	}
	// edge cases must not panic
	for _, c := range []cardData{{}, {Name: "x", HP: -5, MaxHP: 0, MaxMana: 0}, {Name: "علی", Level: 100, Gold: 9e12, HP: 9, MaxHP: 3, Mana: 1, MaxMana: 1}} {
		if _, err := renderCard(base, c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCardKeyAndCompact(t *testing.T) {
	if compact(950) != "950" || compact(1_500_000) != "1.5M" || compact(250_000) != "250K" {
		t.Fatal("compact wrong")
	}
	if asciiName("علی") || !asciiName("Ali_1") || asciiName("") {
		t.Fatal("asciiName wrong")
	}
}
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
