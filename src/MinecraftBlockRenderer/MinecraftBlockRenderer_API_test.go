package minecraftblockrenderer

import (
	"duckysolucky/gorenderer/src/assets"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type skycryptItem struct {
	ID     any            `json:"id"`
	Damage int            `json:"Damage"`
	Tag    map[string]any `json:"tag"`
}

func TestRendererPublicAPIsWithMinimalAssets(t *testing.T) {
	assetsRoot := createMinimalAssets(t)

	renderer, err := CreateFromDataDirectory(assetsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := renderer.GetKnownBlockNames(), []string{"stone"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("known blocks = %#v, want %#v", got, want)
	}
	if got := renderer.GetKnownItemNames(); !contains(got, "diamond_sword") || !contains(got, "red_wool") {
		t.Fatalf("known items missing expected entries: %#v", got)
	}

	block := renderer.RenderBlock("stone", BlockRenderOptions{Size: 32})
	assertImage(t, block, 32, 1)

	face, err := renderer.RenderBlockFace("stone", BlockFaceRenderOptions{Size: 32, Face: 0})
	if err != nil {
		t.Fatal(err)
	}
	assertImage(t, face, 32, 1)

	textureItem, err := renderer.RenderGuiItemFromTextureId("minecraft:item/diamond_sword", &BlockRenderOptions{Size: 32})
	if err != nil {
		t.Fatal(err)
	}
	assertImage(t, textureItem, 32, 1)

	rendered, err := renderer.RenderItemObjectWithResourceId(map[string]any{
		"id":     float64(35),
		"Damage": float64(14),
		"tag": map[string]any{
			"ExtraAttributes": map[string]any{"id": "RED_WOOL_TEST"},
		},
	}, &BlockRenderOptions{Size: 32})
	if err != nil {
		t.Fatal(err)
	}
	assertImage(t, rendered.Image, 32, 1)
	if rendered.ResourceId.ResourceId == "" {
		t.Fatal("empty resource id")
	}

	again, err := renderer.ComputeResourceIdFromItemObject(skycryptItem{
		ID:     35,
		Damage: 14,
		Tag: map[string]any{
			"ExtraAttributes": map[string]any{"id": "RED_WOOL_TEST"},
		},
	}, &BlockRenderOptions{Size: 32})
	if err != nil {
		t.Fatal(err)
	}
	if again.ResourceId != rendered.ResourceId.ResourceId {
		t.Fatalf("resource id changed: %s != %s", again.ResourceId, rendered.ResourceId.ResourceId)
	}
}

func TestCreateFromResourceProviderAndAnimatedRender(t *testing.T) {
	assetsRoot := createMinimalAssets(t)
	provider, err := assets.NewDirectoryResourceProvider(assetsRoot)
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := CreateFromResourceProvider(provider)
	if err != nil {
		t.Fatal(err)
	}

	animated, err := renderer.RenderAnimatedGuiItemWithResourceId("clock", &BlockRenderOptions{Size: 32})
	if err != nil {
		t.Fatal(err)
	}
	assertImage(t, animated.Image, 32, 1)
	if len(animated.Frames) != 2 {
		t.Fatalf("animated frames = %d, want 2", len(animated.Frames))
	}
	for _, frame := range animated.Frames {
		assertImage(t, frame.Image, 32, 1)
		if frame.ResourceId.ResourceId == "" {
			t.Fatal("frame resource id is empty")
		}
	}
}

func TestMinecraftAtlasGeneratorOrderingAndEntryErrors(t *testing.T) {
	renderer, err := CreateFromDataDirectory(createMinimalAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	generator := NewMinecraftAtlasGenerator(renderer)

	result, err := generator.GenerateItemAtlas(MinecraftAtlasOptions{
		Names: []string{"missing_item", "diamond_sword"},
		Size:  16,
		Cols:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Image.Bounds().Dx() != 16 || result.Image.Bounds().Dy() != 32 {
		t.Fatalf("atlas image size = %dx%d, want 16x32", result.Image.Bounds().Dx(), result.Image.Bounds().Dy())
	}
	if got, want := result.Manifest.Width, 16; got != want {
		t.Fatalf("manifest width = %d, want %d", got, want)
	}
	if got, want := result.Manifest.Height, 32; got != want {
		t.Fatalf("manifest height = %d, want %d", got, want)
	}
	if got, want := []string{result.Manifest.Entries[0].Name, result.Manifest.Entries[1].Name}, []string{"diamond_sword", "missing_item"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry order = %#v, want %#v", got, want)
	}
}

func createMinimalAssets(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "assets", "minecraft")
	writeJSON(t, root, "blockstates/stone.json", `{"variants":{"":{"model":"minecraft:block/stone"}}}`)
	writeJSON(t, root, "models/block/stone.json", `{
		"textures":{"all":"minecraft:block/stone"},
		"elements":[{"from":[0,0,0],"to":[16,16,16],"faces":{
			"north":{"texture":"#all"},"south":{"texture":"#all"},"east":{"texture":"#all"},
			"west":{"texture":"#all"},"up":{"texture":"#all"},"down":{"texture":"#all"}
		}}]
	}`)
	writeJSON(t, root, "items/diamond_sword.json", `{"model":{"model":"minecraft:item/diamond_sword"}}`)
	writeJSON(t, root, "items/red_wool.json", `{"model":{"model":"minecraft:item/red_wool"}}`)
	writeJSON(t, root, "items/clock.json", `{"model":{"model":"minecraft:item/clock"}}`)
	writeJSON(t, root, "models/item/diamond_sword.json", `{"parent":"builtin/generated","textures":{"layer0":"minecraft:item/diamond_sword"}}`)
	writeJSON(t, root, "models/item/red_wool.json", `{"parent":"builtin/generated","textures":{"layer0":"minecraft:item/red_wool"}}`)
	writeJSON(t, root, "models/item/clock.json", `{"parent":"builtin/generated","textures":{"layer0":"minecraft:item/clock"}}`)
	writePNG(t, filepath.Join(root, "textures", "block", "stone.png"), 16, 16, color.RGBA{R: 180, G: 30, B: 30, A: 255})
	writePNG(t, filepath.Join(root, "textures", "item", "diamond_sword.png"), 16, 16, color.RGBA{R: 40, G: 180, B: 220, A: 255})
	writePNG(t, filepath.Join(root, "textures", "item", "red_wool.png"), 16, 16, color.RGBA{R: 220, G: 40, B: 40, A: 255})
	writeAnimatedPNG(t, filepath.Join(root, "textures", "item", "clock.png"))
	writeJSON(t, root, "textures/item/clock.png.mcmeta", `{"animation":{"frametime":1,"frames":[0,1],"width":16,"height":16}}`)
	return root
}

func writeJSON(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePNG(t *testing.T, path string, width int, height int, c color.RGBA) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, c)
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

func writeAnimatedPNG(t *testing.T, path string) {
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

func assertImage(t *testing.T, img *image.RGBA, size int, minOpaque int) {
	t.Helper()
	if img == nil {
		t.Fatal("image is nil")
	}
	if img.Bounds().Dx() != size || img.Bounds().Dy() != size {
		t.Fatalf("image size = %dx%d, want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), size, size)
	}
	opaque := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y).A > 0 {
				opaque++
			}
		}
	}
	if opaque < minOpaque {
		t.Fatalf("opaque pixels = %d, want at least %d", opaque, minOpaque)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
