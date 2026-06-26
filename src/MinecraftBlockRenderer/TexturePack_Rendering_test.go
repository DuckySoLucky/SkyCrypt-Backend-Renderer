package minecraftblockrenderer

import (
	texturepacks "duckysolucky/gorenderer/src/TexturePacks"
	"image/color"
	"os"
	"path/filepath"
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
