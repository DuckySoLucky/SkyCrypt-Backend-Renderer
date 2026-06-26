package imagecache

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
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

func TestWriteWebPAtomicPreservesColorsWithTransparentBackground(t *testing.T) {
	path := t.TempDir() + "/transparent-background.webp"
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 4; y < 12; y++ {
		for x := 4; x < 12; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 40, G: 180, B: 220, A: 255})
		}
	}

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
	got := color.NRGBAModel.Convert(decoded.At(8, 8)).(color.NRGBA)
	want := color.NRGBA{R: 40, G: 180, B: 220, A: 255}
	if got != want {
		t.Fatalf("decoded center color = %#v, want %#v", got, want)
	}
	transparent := color.NRGBAModel.Convert(decoded.At(0, 0)).(color.NRGBA)
	if transparent.A != 0 {
		t.Fatalf("decoded transparent color = %#v, want alpha 0", transparent)
	}
}

func TestEnsureCacheVersionPurgesManagedCategoriesOnVersionChange(t *testing.T) {
	root := t.TempDir()
	managedPath := filepath.Join(root, "rendered", "old.webp")
	unmanagedPath := filepath.Join(root, "other", "keep.txt")
	writeTestFile(t, managedPath, "old")
	writeTestFile(t, unmanagedPath, "keep")

	if err := EnsureCacheVersion(root, "test-v1", "rendered", "derived", "player_skins"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(managedPath); !os.IsNotExist(err) {
		t.Fatalf("managed cache file was not purged, err=%v", err)
	}
	if _, err := os.Stat(unmanagedPath); err != nil {
		t.Fatalf("unmanaged file was removed: %v", err)
	}

	currentPath := filepath.Join(root, "rendered", "current.webp")
	writeTestFile(t, currentPath, "current")
	if err := EnsureCacheVersion(root, "test-v1", "rendered", "derived", "player_skins"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(currentPath); err != nil {
		t.Fatalf("matching cache version purged current file: %v", err)
	}

	if err := EnsureCacheVersion(root, "test-v2", "rendered", "derived", "player_skins"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(currentPath); !os.IsNotExist(err) {
		t.Fatalf("version change did not purge managed cache file, err=%v", err)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
