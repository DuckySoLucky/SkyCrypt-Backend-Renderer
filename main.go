package renderer

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	mbr "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/MinecraftBlockRenderer"
	texturepacks "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/TexturePacks"
)

const (
	DefaultSize = 128
)

type Options struct {
	AssetsRoot        string
	ResourcePacksRoot string
	PackIDs           []string
	Size              int
	Preload           bool
}

type RenderedPNG struct {
	PNG        []byte
	ResourceID string
}

type PreRenderOptions struct {
	OutputDir string
	Workers   int
	Overwrite bool
}

type PreRenderedItem struct {
	InputID    string
	ResourceID string
	Path       string
	Error      string
}

type PreRenderResult struct {
	Entries   []PreRenderedItem
	Succeeded int
	Failed    int
}

type Renderer struct {
	renderer *mbr.MinecraftBlockRenderer
	packIDs  []string
	size     int
}

func NewRenderer(options Options) (*Renderer, error) {
	return NewRendererWithOptions(options)
}

func NewRendererWithOptions(options Options) (*Renderer, error) {
	if options.AssetsRoot == "" {
		return nil, fmt.Errorf("assets root is required")
	}
	if options.ResourcePacksRoot == "" {
		return nil, fmt.Errorf("resource packs root is required")
	}
	if len(options.PackIDs) == 0 {
		return nil, fmt.Errorf("at least one pack id is required")
	}

	size := options.Size
	if size <= 0 {
		size = DefaultSize
	}

	packIDs := append([]string(nil), options.PackIDs...)

	registry := texturepacks.NewTexturePackRegistry()
	registry.RegisterAllPacks(options.ResourcePacksRoot, false)

	blockRenderer := mbr.CreateFromMinecraftAssets(options.AssetsRoot, registry, packIDs)
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

func (r *Renderer) PreRenderSkyBlockItemIDs(ctx context.Context, ids []string, options PreRenderOptions) (*PreRenderResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.renderer == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		return nil, fmt.Errorf("output dir is required")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	workers := options.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers < 1 {
		workers = 1
	}

	result := &PreRenderResult{
		Entries: make([]PreRenderedItem, len(ids)),
	}
	indexesByID := make(map[string][]int)
	uniqueIDs := make([]string, 0, len(ids))
	for index, id := range ids {
		trimmed := strings.TrimSpace(id)
		result.Entries[index] = PreRenderedItem{InputID: id}
		if trimmed == "" {
			result.Entries[index].Error = "skyBlockItemID cannot be empty"
			continue
		}
		if _, exists := indexesByID[trimmed]; !exists {
			uniqueIDs = append(uniqueIDs, trimmed)
		}
		indexesByID[trimmed] = append(indexesByID[trimmed], index)
	}

	type renderResult struct {
		inputID    string
		resourceID string
		path       string
		err        error
	}

	jobs := make(chan string)
	results := make(chan renderResult)
	workerCount := minInt(workers, maxInt(1, len(uniqueIDs)))
	var wg sync.WaitGroup
	writtenByResourceID := make(map[string]string)
	var writtenMu sync.Mutex

	renderOne := func(id string) renderResult {
		if err := ctx.Err(); err != nil {
			return renderResult{inputID: id, err: err}
		}

		rendered, err := r.RenderSkyBlockItemID(id)
		if err != nil {
			return renderResult{inputID: id, err: err}
		}
		if rendered == nil || rendered.Image == nil || strings.TrimSpace(rendered.ResourceId.ResourceId) == "" {
			return renderResult{inputID: id, err: fmt.Errorf("failed to render item")}
		}

		resourceID := rendered.ResourceId.ResourceId
		targetPath := filepath.Join(outputDir, resourceID+".png")

		writtenMu.Lock()
		if existingPath, found := writtenByResourceID[resourceID]; found {
			writtenMu.Unlock()
			return renderResult{inputID: id, resourceID: resourceID, path: existingPath}
		}
		if !options.Overwrite {
			if _, statErr := os.Stat(targetPath); statErr == nil {
				writtenByResourceID[resourceID] = targetPath
				writtenMu.Unlock()
				return renderResult{inputID: id, resourceID: resourceID, path: targetPath}
			} else if !os.IsNotExist(statErr) {
				writtenMu.Unlock()
				return renderResult{inputID: id, resourceID: resourceID, err: statErr}
			}
		}

		if err := writeRenderedPNGAtomic(targetPath, rendered.Image); err != nil {
			writtenMu.Unlock()
			return renderResult{inputID: id, resourceID: resourceID, err: err}
		}
		writtenByResourceID[resourceID] = targetPath
		writtenMu.Unlock()

		return renderResult{inputID: id, resourceID: resourceID, path: targetPath}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				results <- renderOne(id)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, id := range uniqueIDs {
			if ctx.Err() != nil {
				return
			}
			select {
			case jobs <- id:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	completedByID := make(map[string]renderResult)
	for rendered := range results {
		completedByID[rendered.inputID] = rendered
	}

	cancelErr := ctx.Err()
	for _, id := range uniqueIDs {
		rendered, completed := completedByID[id]
		if !completed {
			rendered = renderResult{inputID: id, err: cancelErr}
			if rendered.err == nil {
				rendered.err = context.Canceled
			}
		}

		for _, index := range indexesByID[id] {
			entry := result.Entries[index]
			entry.ResourceID = rendered.resourceID
			entry.Path = rendered.path
			if rendered.err != nil {
				entry.Error = rendered.err.Error()
			}
			result.Entries[index] = entry
		}
	}

	for _, entry := range result.Entries {
		if entry.Error != "" {
			result.Failed++
		} else if strings.TrimSpace(entry.Path) != "" {
			result.Succeeded++
		}
	}

	if cancelErr != nil {
		return result, cancelErr
	}
	return result, nil
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

func writeRenderedPNGAtomic(targetPath string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".prerender-*.png")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := png.Encode(tmp, img); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return err
	}
	removeTmp = false
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
