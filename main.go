package renderer

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	mbr "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/MinecraftBlockRenderer"
	texturepacks "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/TexturePacks"
	"github.com/DuckySoLucky/SkyCrypt-Backend-Renderer/src/imagecache"
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
	CacheDir          string
}

type RenderedItem struct {
	Path          string
	TexturePackID string
}

type PreRenderOptions struct {
	Workers   int
	Overwrite bool
}

type PreRenderedItem struct {
	InputID       string
	Path          string
	TexturePackID string
	Error         string
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
	cacheDir string

	renderCache   map[string]RenderedItem
	renderCacheMu sync.RWMutex
	inflight      map[string]*renderInflight
	inflightMu    sync.Mutex
}

type renderInflight struct {
	wg   sync.WaitGroup
	item *RenderedItem
	err  error
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
	cacheDir := strings.TrimSpace(options.CacheDir)
	if cacheDir == "" {
		return nil, fmt.Errorf("cache dir is required")
	}

	size := options.Size
	if size <= 0 {
		size = DefaultSize
	}

	packIDs := append([]string(nil), options.PackIDs...)

	registry := texturepacks.NewTexturePackRegistry()
	registry.RegisterAllPacks(options.ResourcePacksRoot, false)

	if err := imagecache.EnsureCacheVersion(cacheDir, imagecache.CacheFormatVersion, "rendered", "player_skins", "derived"); err != nil {
		return nil, err
	}

	blockRenderer := mbr.CreateFromMinecraftAssets(options.AssetsRoot, registry, packIDs)
	blockRenderer.SetCacheDirectory(cacheDir)
	if options.Preload {
		blockRenderer.PreloadRegisteredPacks(true)
	}

	return &Renderer{
		renderer:      blockRenderer,
		packIDs:       packIDs,
		size:          size,
		cacheDir:      cacheDir,
		renderCache:   make(map[string]RenderedItem),
		inflight:      make(map[string]*renderInflight),
		renderCacheMu: sync.RWMutex{},
		inflightMu:    sync.Mutex{},
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

func (r *Renderer) RenderSkyBlockItemID(id string) (*RenderedItem, error) {
	return r.cachedSkyBlockItemID(id, false)
}

func (r *Renderer) RenderItemNBT(item any) (*RenderedItem, error) {
	if r == nil || r.renderer == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	options := r.renderOptions()
	return r.cachedRenderedItem(
		false,
		func() (*mbr.ResourceIdResult, error) {
			return r.renderer.ComputeResourceIdFromNBT(item, options)
		},
		func() (*mbr.AnimatedRenderedResource, error) {
			return r.renderer.RenderAnimatedItemNBT(item, options)
		},
	)
}

func (r *Renderer) FileFromSkyBlockItemID(id string) (*RenderedItem, error) {
	return r.RenderSkyBlockItemID(id)
}

func (r *Renderer) FileFromNBT(item any) (*RenderedItem, error) {
	return r.RenderItemNBT(item)
}

func (r *Renderer) PreRenderSkyBlockItemIDs(ctx context.Context, ids []string, options PreRenderOptions) (*PreRenderResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.renderer == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	if _, err := r.requireCacheDir(); err != nil {
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
		inputID string
		item    *RenderedItem
		err     error
	}

	jobs := make(chan string)
	results := make(chan renderResult)
	workerCount := minInt(workers, maxInt(1, len(uniqueIDs)))
	var wg sync.WaitGroup

	renderOne := func(id string) renderResult {
		if err := ctx.Err(); err != nil {
			return renderResult{inputID: id, err: err}
		}
		item, err := r.cachedSkyBlockItemID(id, options.Overwrite)
		return renderResult{inputID: id, item: item, err: err}
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
			if rendered.item != nil {
				entry.Path = rendered.item.Path
				entry.TexturePackID = rendered.item.TexturePackID
			}
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

func (r *Renderer) renderOptions() *mbr.BlockRenderOptions {
	return &mbr.BlockRenderOptions{
		Size:    r.Size(),
		PackIds: r.packIDs,
	}
}

func (r *Renderer) requireCacheDir() (string, error) {
	if r == nil || r.renderer == nil {
		return "", fmt.Errorf("renderer is nil")
	}
	cacheDir := strings.TrimSpace(r.cacheDir)
	if cacheDir == "" {
		return "", fmt.Errorf("cache dir is required")
	}
	return cacheDir, nil
}

func (r *Renderer) cachedSkyBlockItemID(id string, overwrite bool) (*RenderedItem, error) {
	if r == nil || r.renderer == nil {
		return nil, fmt.Errorf("renderer is nil")
	}
	options := r.renderOptions()
	return r.cachedRenderedItem(
		overwrite,
		func() (*mbr.ResourceIdResult, error) {
			return r.renderer.ComputeResourceIdFromSkyBlockItemID(id, options)
		},
		func() (*mbr.AnimatedRenderedResource, error) {
			return r.renderer.RenderAnimatedSkyBlockItemID(id, options)
		},
	)
}

func (r *Renderer) cachedRenderedItem(overwrite bool, compute func() (*mbr.ResourceIdResult, error), render func() (*mbr.AnimatedRenderedResource, error)) (*RenderedItem, error) {
	cacheDir, err := r.requireCacheDir()
	if err != nil {
		return nil, err
	}
	resourceID, err := compute()
	if err != nil {
		return nil, err
	}
	if resourceID == nil || strings.TrimSpace(resourceID.ResourceId) == "" {
		return nil, fmt.Errorf("resource id is empty")
	}

	key := resourceID.ResourceId
	target := &RenderedItem{
		Path:          renderedWebPPath(cacheDir, key),
		TexturePackID: sourcePackID(resourceID),
	}

	if !overwrite {
		if cached := r.lookupRenderedItem(key); cached != nil {
			return cached, nil
		}
		if _, statErr := os.Stat(target.Path); statErr == nil {
			r.rememberRenderedItem(key, target)
			return cloneRenderedItem(target), nil
		} else if !os.IsNotExist(statErr) {
			return nil, statErr
		}
	}

	inflight, owner := r.beginRender(key)
	if !owner {
		inflight.wg.Wait()
		if inflight.err != nil {
			return nil, inflight.err
		}
		return cloneRenderedItem(inflight.item), nil
	}
	defer r.finishRender(key, inflight)

	if !overwrite {
		if cached := r.lookupRenderedItem(key); cached != nil {
			inflight.item = cached
			return cloneRenderedItem(cached), nil
		}
		if _, statErr := os.Stat(target.Path); statErr == nil {
			r.rememberRenderedItem(key, target)
			inflight.item = target
			return cloneRenderedItem(target), nil
		} else if !os.IsNotExist(statErr) {
			inflight.err = statErr
			return nil, statErr
		}
	}

	rendered, err := render()
	if err != nil {
		inflight.err = err
		return nil, err
	}
	if err := writeRenderedWebP(target.Path, rendered); err != nil {
		inflight.err = err
		return nil, err
	}

	r.rememberRenderedItem(key, target)
	inflight.item = target
	return cloneRenderedItem(target), nil
}

func (r *Renderer) lookupRenderedItem(key string) *RenderedItem {
	r.renderCacheMu.RLock()
	cached, found := r.renderCache[key]
	r.renderCacheMu.RUnlock()
	if !found {
		return nil
	}
	return &RenderedItem{Path: cached.Path, TexturePackID: cached.TexturePackID}
}

func (r *Renderer) rememberRenderedItem(key string, item *RenderedItem) {
	if item == nil {
		return
	}
	r.renderCacheMu.Lock()
	if r.renderCache == nil {
		r.renderCache = make(map[string]RenderedItem)
	}
	r.renderCache[key] = *item
	r.renderCacheMu.Unlock()
}

func (r *Renderer) beginRender(key string) (*renderInflight, bool) {
	r.inflightMu.Lock()
	defer r.inflightMu.Unlock()
	if r.inflight == nil {
		r.inflight = make(map[string]*renderInflight)
	}
	if existing, found := r.inflight[key]; found {
		return existing, false
	}
	inflight := &renderInflight{}
	inflight.wg.Add(1)
	r.inflight[key] = inflight
	return inflight, true
}

func (r *Renderer) finishRender(key string, inflight *renderInflight) {
	inflight.wg.Done()
	r.inflightMu.Lock()
	if r.inflight[key] == inflight {
		delete(r.inflight, key)
	}
	r.inflightMu.Unlock()
}

func renderedWebPPath(cacheDir string, resourceID string) string {
	return filepath.Join(cacheDir, "rendered", resourceID+".webp")
}

func renderedPNGPathFromWebP(webpPath string) string {
	return strings.TrimSuffix(webpPath, filepath.Ext(webpPath)) + ".png"
}

func sourcePackID(resourceID *mbr.ResourceIdResult) string {
	if resourceID == nil || strings.TrimSpace(resourceID.SourcePackId) == "" {
		return mbr.VanillaPackId
	}
	return resourceID.SourcePackId
}

func writeRenderedWebP(targetPath string, rendered *mbr.AnimatedRenderedResource) error {
	if rendered == nil {
		return fmt.Errorf("rendered resource is nil")
	}

	frames := rendered.Frames
	if len(frames) == 0 {
		if rendered.Image == nil {
			return fmt.Errorf("rendered image is nil")
		}
		return writeRenderedStillImages(targetPath, rendered.Image)
	}

	if len(frames) == 1 {
		if frames[0].Image == nil {
			return fmt.Errorf("rendered frame image is nil")
		}
		return writeRenderedStillImages(targetPath, frames[0].Image)
	}

	images := make([]image.Image, 0, len(frames))
	durations := make([]uint, 0, len(frames))
	for _, frame := range frames {
		if frame.Image == nil {
			return fmt.Errorf("rendered frame image is nil")
		}
		duration := frame.DurationMs
		if duration <= 0 {
			duration = 50
		}
		images = append(images, frame.Image)
		durations = append(durations, uint(duration))
	}
	if err := imagecache.WriteAnimatedWebPAtomic(targetPath, images, durations); err != nil {
		return err
	}
	return imagecache.WritePNGAtomic(renderedPNGPathFromWebP(targetPath), frames[0].Image)
}

func writeRenderedStillImages(targetPath string, img image.Image) error {
	if err := imagecache.WriteWebPAtomic(targetPath, img); err != nil {
		return err
	}
	return imagecache.WritePNGAtomic(renderedPNGPathFromWebP(targetPath), img)
}

func cloneRenderedItem(item *RenderedItem) *RenderedItem {
	if item == nil {
		return nil
	}
	return &RenderedItem{
		Path:          item.Path,
		TexturePackID: item.TexturePackID,
	}
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
