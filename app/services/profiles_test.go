package services

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"
)

func avatarPNG(width, height int) string {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, width, height)))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestNormalizeAvatar(t *testing.T) {
	for _, input := range []string{"https://example.com/image.jpg", "data:image/svg+xml;base64,PHN2Zz4=", "data:image/jpeg;base64,bm90IGFuIGltYWdl", "data:image/png;base64," + strings.Repeat("A", 28001), avatarPNG(257, 1), avatarPNG(1, 257)} {
		if _, err := NormalizeAvatar(input); err == nil {
			t.Fatalf("unsafe avatar accepted: %.40s", input)
		}
	}
	value, err := NormalizeAvatar(avatarPNG(128, 128))
	if err != nil || !strings.HasPrefix(value, "data:image/jpeg;base64,") {
		t.Fatal("raster normalization failed", err)
	}
	if _, err := NormalizeAvatar(value); err != nil {
		t.Fatal("client JPEG normalization failed", err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "data:image/jpeg;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || format != "jpeg" || config.Width != 128 || config.Height != 128 {
		t.Fatal("invalid normalized image", config, format, err)
	}
	if empty, err := NormalizeAvatar(""); err != nil || empty != "" {
		t.Fatal("removal failed")
	}
}
