package minecraftblockrenderer

import (
	texturepacks "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/TexturePacks"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeResourceIdIncludesPackOverrides(t *testing.T) {
	assetsRoot := createMinimalAssets(t)
	packRoot := createOverridePack(t, "overridepack", color.RGBA{R: 20, G: 220, B: 80, A: 255})

	registry := texturepacks.NewTexturePackRegistry()
	if _, err := registry.RegisterPack(packRoot); err != nil {
		t.Fatal(err)
	}
	renderer := CreateFromMinecraftAssets(assetsRoot, registry, nil)
	options := &BlockRenderOptions{Size: 32, PackIds: []string{"overridepack"}}

	rendered := renderer.RenderGuiItemWithResourceId("diamond_sword", options)
	if rendered == nil || !hasOpaquePixels(rendered.Image) {
		t.Fatal("pack override render did not produce visible pixels")
	}
	if rendered.ResourceId.SourcePackId != "overridepack" {
		t.Fatalf("source pack = %q, want overridepack; model=%v textures=%v", rendered.ResourceId.SourcePackId, rendered.ResourceId.Model, rendered.ResourceId.Textures)
	}

	resolvedRenderer, forwarded := renderer.ResolveRendererForOptions(*options)
	recomputed := resolvedRenderer.ComputeResourceIdInternal("diamond_sword", forwarded, nil)
	if recomputed.ResourceId != rendered.ResourceId.ResourceId {
		t.Fatalf("recomputed resource id = %q, want %q", recomputed.ResourceId, rendered.ResourceId.ResourceId)
	}
}

func TestMissingSkyBlockCustomTextureFallsBackToVanillaItem(t *testing.T) {
	assetsRoot := createMinimalAssets(t)
	packRoot := createEmptyPack(t, "emptypack")

	registry := texturepacks.NewTexturePackRegistry()
	if _, err := registry.RegisterPack(packRoot); err != nil {
		t.Fatal(err)
	}
	renderer := CreateFromMinecraftAssets(assetsRoot, registry, nil)

	rendered, err := renderer.RenderItemObjectWithResourceId(map[string]any{
		"id": "minecraft:diamond_sword",
		"tag": map[string]any{
			"ExtraAttributes": map[string]any{
				"id": "MISSING_CUSTOM_SWORD",
			},
		},
	}, &BlockRenderOptions{Size: 32, PackIds: []string{"emptypack"}})
	if err != nil {
		t.Fatal(err)
	}
	if rendered == nil || !hasOpaquePixels(rendered.Image) {
		t.Fatal("fallback render did not produce visible pixels")
	}
	model := ""
	if rendered.ResourceId.Model != nil {
		model = *rendered.ResourceId.Model
	}
	if rendered.ResourceId.SourcePackId != VanillaPackId || !strings.Contains(strings.ToLower(model), "diamond_sword") {
		t.Fatalf("fallback did not resolve to vanilla diamond sword: source=%s model=%s textures=%v", rendered.ResourceId.SourcePackId, model, rendered.ResourceId.Textures)
	}
}

func TestMissingSkyBlockCustomSkullFallsBackWithSkullResolver(t *testing.T) {
	assetsRoot := createMinimalAssets(t)
	packRoot := createEmptyPack(t, "emptypack")

	registry := texturepacks.NewTexturePackRegistry()
	if _, err := registry.RegisterPack(packRoot); err != nil {
		t.Fatal(err)
	}
	renderer := CreateFromMinecraftAssets(assetsRoot, registry, nil)

	var sawResolver bool
	texture := "minecraft:item/diamond_sword"
	rendered, err := renderer.RenderItemObjectWithResourceId(map[string]any{
		"id": "minecraft:player_head",
		"tag": map[string]any{
			"ExtraAttributes": map[string]any{
				"id": "MISSING_CUSTOM_HEAD",
			},
		},
	}, &BlockRenderOptions{
		Size:    32,
		PackIds: []string{"emptypack"},
		SkullTextureResolver: func(context SkullResolverContext) *string {
			sawResolver = true
			if context.CustomDataId == nil || *context.CustomDataId != "MISSING_CUSTOM_HEAD" {
				t.Fatalf("custom data id = %#v", context.CustomDataId)
			}
			return &texture
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rendered == nil || !hasOpaquePixels(rendered.Image) {
		t.Fatal("fallback skull render did not produce visible pixels")
	}
	if !sawResolver {
		t.Fatal("skull resolver was not called for fallback player head")
	}
}

func createOverridePack(t *testing.T, id string, c color.RGBA) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), id)
	writeJSON(t, root, "meta.json", `{"id":"`+id+`","name":"Override Pack","version":"test"}`)
	writeJSON(t, root, "pack.mcmeta", `{"pack":{"pack_format":99,"description":"test"}}`)
	writeJSON(t, root, "assets/minecraft/items/diamond_sword.json", `{"model":{"type":"model","model":"minecraft:item/diamond_sword"}}`)
	writeJSON(t, root, "assets/minecraft/models/item/diamond_sword.json", `{"parent":"builtin/generated","textures":{"layer0":"minecraft:item/diamond_sword"}}`)
	writePNG(t, filepath.Join(root, "assets", "minecraft", "textures", "item", "diamond_sword.png"), 16, 16, c)
	if err := os.WriteFile(filepath.Join(root, "pack.png"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func createEmptyPack(t *testing.T, id string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), id)
	writeJSON(t, root, "meta.json", `{"id":"`+id+`","name":"Empty Pack","version":"test"}`)
	writeJSON(t, root, "pack.mcmeta", `{"pack":{"pack_format":99,"description":"test"}}`)
	if err := os.MkdirAll(filepath.Join(root, "assets", "minecraft"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pack.png"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
