package data

import (
	"duckysolucky/gorenderer/src/assets"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestAnimatedTextureUsesFirstFrameDimensions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assets", "minecraft")
	texturePath := filepath.Join(root, "textures", "item", "clock.png")
	writeTextureRepoAnimatedPNG(t, texturePath)
	writeTextureRepoText(t, texturePath+".mcmeta", `{"animation":{"frametime":1,"frames":[0,1],"width":16,"height":16}}`)

	provider, err := assets.NewDirectoryResourceProvider(filepath.Join(root, "textures"))
	if err != nil {
		t.Fatal(err)
	}
	registry := assets.NewAssetNamespaceRegistry()
	registry.AddNamespaceWithProvider("minecraft", filepath.Join(root, "textures"), "test", true, provider)
	repository := NewTextureRepository(filepath.Join(root, "textures"), nil, nil, *registry)
	texture := repository.GetTexture("minecraft:item/clock")
	if texture.Bounds().Dx() != 16 || texture.Bounds().Dy() != 16 {
		t.Fatalf("animated first frame size = %dx%d, want 16x16", texture.Bounds().Dx(), texture.Bounds().Dy())
	}
	animation, ok := repository.GetAnimation("minecraft:item/clock")
	if !ok || len(animation.Frames) != 2 {
		t.Fatalf("animation frames = %#v, ok=%v; want 2 frames", animation, ok)
	}
}

func writeTextureRepoText(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTextureRepoAnimatedPNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 16, 32))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
			img.SetRGBA(x, y+16, color.RGBA{G: 255, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}
