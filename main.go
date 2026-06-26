package renderer

import (
	"bytes"
	"fmt"
	"image/png"

	mbr "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/MinecraftBlockRenderer"
	texturepacks "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/TexturePacks"
)

const (
	DefaultSize = 128
)

var DefaultPackIDs = []string{"fsr", "hplus"}

type Options struct {
	Size    int
	PackIDs []string
	Preload bool
}

type RenderedPNG struct {
	PNG        []byte
	ResourceID string
}

type Renderer struct {
	renderer *mbr.MinecraftBlockRenderer
	packIDs  []string
	size     int
}

func NewRenderer(assetsRoot, texturePacksRoot string) (*Renderer, error) {
	return NewRendererWithOptions(assetsRoot, texturePacksRoot, Options{
		Size:    DefaultSize,
		PackIDs: DefaultPackIDs,
		Preload: true,
	})
}

func NewRendererWithOptions(assetsRoot, texturePacksRoot string, options Options) (*Renderer, error) {
	size := options.Size
	if size <= 0 {
		size = DefaultSize
	}

	packIDs := append([]string(nil), options.PackIDs...)
	if len(packIDs) == 0 {
		packIDs = append([]string(nil), DefaultPackIDs...)
	}

	registry := texturepacks.NewTexturePackRegistry()
	registry.RegisterAllPacks(texturePacksRoot, false)

	blockRenderer := mbr.CreateFromMinecraftAssets(assetsRoot, registry, packIDs)
	if options.Preload {
		blockRenderer.PreloadRegisteredPacks(true)
	}

	return &Renderer{
		renderer: blockRenderer,
		packIDs:  packIDs,
		size:     size,
	}, nil
}

func (r *Renderer) Size() int {
	if r == nil || r.size <= 0 {
		return DefaultSize
	}
	return r.size
}

func (r *Renderer) PackIDs() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.packIDs...)
}

func (r *Renderer) MinecraftRenderer() *mbr.MinecraftBlockRenderer {
	if r == nil {
		return nil
	}
	return r.renderer
}

func (r *Renderer) RenderSkyBlockItemID(id string) (*mbr.RenderedResource, error) {
	if r == nil || r.renderer == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	return r.renderer.RenderSkyBlockItemID(id, &mbr.BlockRenderOptions{
		Size:    r.Size(),
		PackIds: r.packIDs,
	})
}

func (r *Renderer) RenderItemNBT(item any) (*mbr.RenderedResource, error) {
	if r == nil || r.renderer == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	return r.renderer.RenderItemNBT(item, &mbr.BlockRenderOptions{
		Size:    r.Size(),
		PackIds: r.packIDs,
	})
}

func (r *Renderer) TextureFromSkyBlockItemID(id string) ([]byte, string, error) {
	rendered, err := r.RenderSkyBlockItemID(id)
	if err != nil {
		return nil, "", err
	}
	return EncodeRendered(rendered)
}

func (r *Renderer) TextureFromNBT(item any) ([]byte, string, error) {
	rendered, err := r.RenderItemNBT(item)
	if err != nil {
		return nil, "", err
	}
	return EncodeRendered(rendered)
}

func (r *Renderer) PNGFromSkyBlockItemID(id string) (*RenderedPNG, error) {
	pngBytes, resourceID, err := r.TextureFromSkyBlockItemID(id)
	if err != nil {
		return nil, err
	}
	return &RenderedPNG{PNG: pngBytes, ResourceID: resourceID}, nil
}

func (r *Renderer) PNGFromNBT(item any) (*RenderedPNG, error) {
	pngBytes, resourceID, err := r.TextureFromNBT(item)
	if err != nil {
		return nil, err
	}
	return &RenderedPNG{PNG: pngBytes, ResourceID: resourceID}, nil
}

func EncodeRendered(rendered *mbr.RenderedResource) ([]byte, string, error) {
	if rendered == nil || rendered.Image == nil {
		return nil, "", fmt.Errorf("rendered resource is nil")
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, rendered.Image); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), rendered.ResourceId.ResourceId, nil
}
