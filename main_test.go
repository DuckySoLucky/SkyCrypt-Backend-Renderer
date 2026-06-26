package renderer

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	mbr "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/MinecraftBlockRenderer"
	texturepacks "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/TexturePacks"
)

func TestNewRendererRequiresPackIDs(t *testing.T) {
	_, err := NewRenderer(Options{
		AssetsRoot:        "assets",
		ResourcePacksRoot: "resourcepacks",
	})
	if err == nil {
		t.Fatal("expected missing pack IDs to return an error")
	}
}

func TestRendererPackIDsReturnsCopy(t *testing.T) {
	source := []string{"fsr", "hplus"}
	renderer := &Renderer{packIDs: append([]string(nil), source...)}

	packIDs := renderer.PackIDs()
	packIDs[0] = "changed"

	if got := renderer.PackIDs()[0]; got != "fsr" {
		t.Fatalf("PackIDs exposed internal slice, got %q", got)
	}
}

func TestPreRenderSkyBlockItemIDsWritesPNGCache(t *testing.T) {
	renderer := newTestRenderer(t)
	outputDir := t.TempDir()

	result, err := renderer.PreRenderSkyBlockItemIDs(context.Background(), []string{"HYPERION"}, PreRenderOptions{
		OutputDir: outputDir,
		Workers:   2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 0 {
		t.Fatalf("result counts = %d succeeded, %d failed", result.Succeeded, result.Failed)
	}
	entry := result.Entries[0]
	if entry.InputID != "HYPERION" || entry.ResourceID == "" || entry.Path == "" || entry.Error != "" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if _, err := os.Stat(entry.Path); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := png.Decode(file); err != nil {
		t.Fatalf("cache file is not a png: %v", err)
	}
}

func TestPreRenderSkyBlockItemIDsSkipsExistingUnlessOverwrite(t *testing.T) {
	renderer := newTestRenderer(t)
	outputDir := t.TempDir()

	first, err := renderer.PreRenderSkyBlockItemIDs(context.Background(), []string{"TEST_ITEM"}, PreRenderOptions{OutputDir: outputDir})
	if err != nil {
		t.Fatal(err)
	}
	path := first.Entries[0].Path
	if err := os.WriteFile(path, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}

	skipped, err := renderer.PreRenderSkyBlockItemIDs(context.Background(), []string{"TEST_ITEM"}, PreRenderOptions{OutputDir: outputDir})
	if err != nil {
		t.Fatal(err)
	}
	if skipped.Entries[0].Path != path {
		t.Fatalf("path changed on skip: %q != %q", skipped.Entries[0].Path, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sentinel" {
		t.Fatal("existing cache file was overwritten with Overwrite=false")
	}

	overwritten, err := renderer.PreRenderSkyBlockItemIDs(context.Background(), []string{"TEST_ITEM"}, PreRenderOptions{
		OutputDir: outputDir,
		Overwrite: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(overwritten.Entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := png.Decode(file); err != nil {
		t.Fatalf("overwrite did not restore png content: %v", err)
	}
}

func TestPreRenderSkyBlockItemIDsPreservesDuplicates(t *testing.T) {
	renderer := newTestRenderer(t)

	result, err := renderer.PreRenderSkyBlockItemIDs(context.Background(), []string{"TEST_ITEM", "TEST_ITEM"}, PreRenderOptions{
		OutputDir: t.TempDir(),
		Workers:   2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 || result.Succeeded != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Entries[0].ResourceID == "" || result.Entries[0].ResourceID != result.Entries[1].ResourceID {
		t.Fatalf("duplicate entries did not share resource id: %#v", result.Entries)
	}
	if result.Entries[0].Path != result.Entries[1].Path {
		t.Fatalf("duplicate entries did not share cache path: %#v", result.Entries)
	}
}

func TestPreRenderSkyBlockItemIDsReportsEntryErrors(t *testing.T) {
	renderer := newTestRenderer(t)

	result, err := renderer.PreRenderSkyBlockItemIDs(context.Background(), []string{"TEST_ITEM", " "}, PreRenderOptions{
		OutputDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("result counts = %d succeeded, %d failed", result.Succeeded, result.Failed)
	}
	if result.Entries[1].Error == "" {
		t.Fatalf("blank id did not report an error: %#v", result.Entries[1])
	}
}

func TestPreRenderSkyBlockItemIDsHonorsCanceledContext(t *testing.T) {
	renderer := newTestRenderer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := renderer.PreRenderSkyBlockItemIDs(ctx, []string{"TEST_ITEM"}, PreRenderOptions{
		OutputDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if result == nil || result.Failed != 1 || result.Entries[0].Error == "" {
		t.Fatalf("canceled prerender did not return partial failure result: %#v", result)
	}
}

func newTestRenderer(t testing.TB) *Renderer {
	t.Helper()

	assetsRoot := createRootMinimalAssets(t)
	packRoot := createRootSkyblockPack(t, "testpack")
	registry := texturepacks.NewTexturePackRegistry()
	if _, err := registry.RegisterPack(packRoot); err != nil {
		t.Fatal(err)
	}
	blockRenderer := mbr.CreateFromMinecraftAssets(assetsRoot, registry, []string{"testpack"})
	return &Renderer{
		renderer: blockRenderer,
		packIDs:  []string{"testpack"},
		size:     32,
	}
}

func createRootMinimalAssets(t testing.TB) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "assets", "minecraft")
	writeRootJSON(t, root, "blockstates/stone.json", `{"variants":{"":{"model":"minecraft:block/stone"}}}`)
	writeRootJSON(t, root, "models/block/stone.json", `{
		"textures":{"all":"minecraft:block/stone"},
		"elements":[{"from":[0,0,0],"to":[16,16,16],"faces":{
			"north":{"texture":"#all"},"south":{"texture":"#all"},"east":{"texture":"#all"},
			"west":{"texture":"#all"},"up":{"texture":"#all"},"down":{"texture":"#all"}
		}}]
	}`)
	writeRootJSON(t, root, "items/player_head.json", `{"model":{"model":"minecraft:item/player_head"}}`)
	writeRootJSON(t, root, "models/item/player_head.json", `{"parent":"builtin/generated","textures":{"layer0":"minecraft:item/player_head"}}`)
	writeRootPNG(t, filepath.Join(root, "textures", "block", "stone.png"), 16, 16, color.RGBA{R: 180, G: 30, B: 30, A: 255})
	writeRootPNG(t, filepath.Join(root, "textures", "item", "player_head.png"), 16, 16, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	return root
}

func createRootSkyblockPack(t testing.TB, id string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), id)
	writeRootJSON(t, root, "meta.json", `{"id":"`+id+`","name":"Test Pack","version":"test"}`)
	writeRootJSON(t, root, "pack.mcmeta", `{"pack":{"pack_format":99,"description":"test"}}`)
	writeRootJSON(t, root, "assets/minecraft/items/player_head.json", `{"model":{"model":"minecraft:item/player_head"}}`)
	writeRootJSON(t, root, "assets/skyblock/items/test_item.json", `{"model":{"type":"model","model":"firmskyblock:item/test_item"}}`)
	writeRootJSON(t, root, "assets/firmskyblock/models/item/test_item.json", `{"parent":"builtin/generated","textures":{"layer0":"firmskyblock:item/test_item"}}`)
	writeRootPNG(t, filepath.Join(root, "assets", "firmskyblock", "textures", "item", "test_item.png"), 16, 16, color.RGBA{R: 40, G: 180, B: 220, A: 255})
	if err := os.WriteFile(filepath.Join(root, "pack.png"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeRootJSON(t testing.TB, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeRootPNG(t testing.TB, path string, width int, height int, c color.RGBA) {
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
