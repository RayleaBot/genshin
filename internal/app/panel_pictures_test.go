package app

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestPanelPicturesNumberAndRemoveLikeMiao(t *testing.T) {
	store := &PanelPictureStore{Directory: t.TempDir()}
	picture := func(width int) []byte {
		var out bytes.Buffer
		if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, width, 1))); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	for _, width := range []int{1, 2, 1} {
		if err := store.Add("10000046", picture(width)); err != nil {
			t.Fatal(err)
		}
	}
	// The same picture is kept once.
	names := store.List("10000046")
	if len(names) != 2 || store.Remove("10000046", 3) || !store.Remove("10000046", 1) || len(store.List("10000046")) != 1 || store.List("10000046")[0] != names[1] {
		t.Fatalf("names = %v then %v", names, store.List("10000046"))
	}
}
