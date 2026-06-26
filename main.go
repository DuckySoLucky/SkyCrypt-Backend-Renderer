package main

import (
	"bytes"
	mbr "duckysolucky/gorenderer/src/MinecraftBlockRenderer"
	texturepacks "duckysolucky/gorenderer/src/TexturePacks"
	"time"

	"fmt"
	"image/png"
	"os"
	"path/filepath"
)

type Renderer struct {
	r       *mbr.MinecraftBlockRenderer
	packIDs []string
	size    int
}

func NewRenderer(assetsRoot, texturePacksRoot string) (*Renderer, error) {
	registry := texturepacks.NewTexturePackRegistry()
	registry.RegisterAllPacks(texturePacksRoot, false)

	packIDs := []string{"fsr", "hplus"}

	renderer := mbr.CreateFromMinecraftAssets(assetsRoot, registry, packIDs)
	renderer.PreloadRegisteredPacks(true)

	return &Renderer{
		r:       renderer,
		packIDs: packIDs,
		size:    128,
	}, nil
}

func (s *Renderer) TextureFromSkyBlockItemID(id string) ([]byte, string, error) {
	rendered, err := s.r.RenderSkyBlockItemID(id, &mbr.BlockRenderOptions{
		Size:    s.size,
		PackIds: s.packIDs,
	})
	if err != nil {
		return nil, "", err
	}

	return encodeRendered(rendered)
}

func (s *Renderer) TextureFromNBT(item any) ([]byte, string, error) {
	rendered, err := s.r.RenderItemNBT(item, &mbr.BlockRenderOptions{
		Size:    s.size,
		PackIds: s.packIDs,
	})
	if err != nil {
		return nil, "", err
	}

	return encodeRendered(rendered)
}

func encodeRendered(rendered *mbr.RenderedResource) ([]byte, string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, rendered.Image); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), rendered.ResourceId.ResourceId, nil
}

func main() {
	cwd, _ := os.Getwd()
	assetsPath := filepath.Join(cwd, "packs", "assets", "minecraft")
	texturePacksPath := filepath.Join(cwd, "texturepacks")

	renderer, err := NewRenderer(assetsPath, texturePacksPath)
	if err != nil {
		fmt.Printf("Error creating renderer: %v\n", err)
		return
	}

	packs := renderer.r.GetLoadedResourcePacks()
	for _, pack := range packs {
		fmt.Printf("Loaded resource pack: %s - (%+v)\n", pack.Pack.DisplayName, pack.Meta.Version)
	}

	timeNow := time.Now()

	pngBytes, cacheKey, err := renderer.TextureFromNBT(map[string]any{
		"id":    "minecraft:iron_sword",
		"Count": 1,
		"tag": map[string]any{
			"ExtraAttributes": map[string]any{
				// "id": "HYPERION",
				"id": "STRONG_DRAGON_HELMET",
			},
		},
	})
	if err != nil {
		fmt.Printf("Error rendering item: %v\n", err)
		return
	}

	if err := os.WriteFile("midas_sword.png", pngBytes, 0644); err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		return
	}

	fmt.Printf("Rendered MIDAS_SWORD as midas_sword.png with cache key %s in %v\n", cacheKey, time.Since(timeNow))

}
