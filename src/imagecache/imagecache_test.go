package imagecache

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/HugoSmits86/nativewebp"
)

func TestWriteWebPAtomicPreservesStraightAlphaRGBAColors(t *testing.T) {
	path := t.TempDir() + "/straight-alpha.webp"
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 40, G: 180, B: 220, A: 128})

	if err := WriteWebPAtomic(path, img); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	decoded, err := nativewebp.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	got := color.NRGBAModel.Convert(decoded.At(0, 0)).(color.NRGBA)
	want := color.NRGBA{R: 40, G: 180, B: 220, A: 128}
	if got != want {
		t.Fatalf("decoded color = %#v, want %#v", got, want)
	}
}
