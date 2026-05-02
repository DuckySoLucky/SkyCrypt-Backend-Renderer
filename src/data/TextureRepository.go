package data

import (
	"bytes"
	"duckysolucky/gorenderer/src/assets"
	"duckysolucky/gorenderer/src/global"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"
)

type TextureRepository struct {
	_assetNamespaces assets.AssetNamespaceRegistry

	_cache                   map[string]image.RGBA
	_missingTexture          image.RGBA
	_activeAnimationOverride *AnimationOverride
	_sources                 []TextureSource
	_trimPaletteLookup       map[int]int
	_trimPaletteLength       int
	_embedded                map[string]string
	_animationCache          map[string]TextureAnimation

	GrassColorMap      *image.RGBA
	FoliageColorMap    *image.RGBA
	DryFoliageColorMap *image.RGBA
}

func NewTextureRepository(dataRoot string, embeddedTextureFile *string, overlayRoots []string, assetNamespaces assets.AssetNamespaceRegistry) *TextureRepository {
	_textureRepository := &TextureRepository{
		_assetNamespaces: assetNamespaces,
		_cache:           make(map[string]image.RGBA),
	}

	_textureRepository._sources = _textureRepository.BuildSourceList(dataRoot, overlayRoots, &assetNamespaces)
	_textureRepository._missingTexture = _textureRepository.CreateMissingTexture()

	if trimPaletteColors, found := TryLoadTrimPaletteColors(_textureRepository); found {
		fmt.Printf("Loaded trim palette with %d colors\n", len(trimPaletteColors))
		_textureRepository._trimPaletteLength = len(trimPaletteColors)
		_textureRepository._trimPaletteLookup = make(map[int]int)
		for i, color := range trimPaletteColors {
			_textureRepository._trimPaletteLookup[int(color.R)<<24|int(color.G)<<16|int(color.B)<<8|int(color.A)] = i
		}
	} else {
		_textureRepository._trimPaletteLength = 0
		_textureRepository._trimPaletteLookup = make(map[int]int)
	}

	colormapRoot := _textureRepository.FindColormapRoot()
	if colormapRoot != nil {
		_textureRepository.GrassColorMap, _ = _textureRepository.TryLoadColormapTexture("colormap/grass")
		_textureRepository.FoliageColorMap, _ = _textureRepository.TryLoadColormapTexture("colormap/foliage")
		_textureRepository.DryFoliageColorMap, _ = _textureRepository.TryLoadColormapTexture("colormap/dry_foliage")
	}

	if embeddedTextureFile != nil && *embeddedTextureFile != "" {
		if _, err := os.Stat(*embeddedTextureFile); err == nil {
			var data, err = os.ReadFile(*embeddedTextureFile)
			if err != nil {
				fmt.Printf("Failed to read embedded texture file: %v\n", err)
			}

			var entries []TextureContentEntry
			if err := global.JSON.Unmarshal(data, &entries); err != nil {
				fmt.Printf("Failed to parse embedded texture JSON: %v\n", err)
			}

			for _, entry := range entries {
				if strings.TrimSpace(*entry.Texture) != "" {
					key := _textureRepository.NormalizeTextureId(entry.Name)
					_textureRepository._embedded[key] = *entry.Texture
				}
			}
		}
	}

	return _textureRepository
}

func (_textureRepository *TextureRepository) GetTexture(textureId string) *image.RGBA {
	fmt.Printf("\nGetTexture: %v\n", textureId)
	if textureId == "" {
		return &_textureRepository._missingTexture
	}

	normalizedTextureId := _textureRepository.NormalizeTextureId(textureId)
	overrideContext := _textureRepository._activeAnimationOverride
	if overrideContext != nil {
		if overrideFrame, found := overrideContext.GetAnimationOverrideFrame(normalizedTextureId); found {
			return &overrideFrame.Image
		}
	}

	if texture, found := _textureRepository._cache[normalizedTextureId]; found {
		return &texture
	}

	loadedTexture := _textureRepository.LoadTextureInternal(normalizedTextureId)

	_textureRepository._cache[normalizedTextureId] = loadedTexture

	fmt.Printf("Returning image with height and width of %d and %d\n", loadedTexture.Bounds().Dy(), loadedTexture.Bounds().Dx())
	return &loadedTexture
}

func (_textureRepository *TextureRepository) TryGetTexture(textureId string) (*image.RGBA, bool) {
	if textureId == "" {
		return nil, false
	}

	texture := _textureRepository.GetTexture(textureId)
	if texture == nil || sameRGBA(*texture, _textureRepository._missingTexture) {
		return nil, false
	}

	return texture, true
}

func (_textureRepository *TextureRepository) LoadTextureInternal(normalizedTextureId string) image.RGBA {

	fmt.Printf("normalizedTextureId: %s\n", normalizedTextureId)

	namespaceName, pathWithinNamespace := _textureRepository.ParseNamespace(normalizedTextureId)
	logicalPaths := EnumerateLogicalPaths(pathWithinNamespace)

	fmt.Printf("Parsed namespace: %s, pathWithinNamespace: %s\n", namespaceName, pathWithinNamespace)

	// Iterate sources in reverse order (High Priority -> Low Priority)
	for i := len(_textureRepository._sources) - 1; i >= 0; i-- {
		source := _textureRepository._sources[i]
		for _, logicalPath := range logicalPaths {
			fmt.Printf("Trying to resolve texture for namespace: %s, logicalPath: %s\n", namespaceName, logicalPath)
			if resource, found := source.TryResolve(namespaceName, logicalPath); found {
				fmt.Printf("Found resource for namespace: %s, logicalPath: %s, attempting to load image\n", namespaceName, logicalPath)

				stream, err := resource.OpenImage()
				if err != nil {
					continue
				}
				loadedTexture, err := LoadImageFromStream(stream)
				if closeErr := stream.Close(); closeErr != nil {
					fmt.Printf("warning: failed to close stream for texture resource: %v\n", closeErr)
				}
				if err != nil {
					continue
				}

				fmt.Printf("Loaded texture for %s from source, processing animation if needed\n", normalizedTextureId)
				return _textureRepository.ProcessAnimatedTexture(normalizedTextureId, loadedTexture, resource.OpenMcmeta)
			}
		}
	}

	if dataUri, found := _textureRepository._embedded[normalizedTextureId]; found {
		if image, err := DecodeDataUri(dataUri); err == nil {
			fmt.Printf("Loaded texture for %s from embedded data\n", normalizedTextureId)
			return image
		}
	}

	shortKeyIndex := strings.LastIndex(normalizedTextureId, "/")
	if shortKeyIndex >= 0 {
		shortKey := normalizedTextureId[shortKeyIndex+1:]
		if dataUri, found := _textureRepository._embedded[shortKey]; found {
			if image, err := DecodeDataUri(dataUri); err == nil {
				fmt.Printf("Loaded texture for %s using short key from embedded data\n", normalizedTextureId)
				return image
			}
		}
	}

	if generated, found := _textureRepository.TryGenerateArmorTrimTexture(normalizedTextureId); found {
		fmt.Printf("Generated armor trim texture for %s\n", normalizedTextureId)
		return generated
	}

	fmt.Printf("Failed to load texture for %s, returning missing texture\n", normalizedTextureId)
	return _textureRepository._missingTexture
}

func (_textureRepository *TextureRepository) ParseNamespace(normalized string) (string, string) {
	sanitized := strings.TrimLeft(normalized, "/")
	sanitized = strings.ReplaceAll(sanitized, "\\", "/")

	colonIndex := strings.IndexByte(sanitized, ':')
	if colonIndex >= 0 {
		return sanitized[:colonIndex], sanitized[colonIndex+1:]
	}

	return "minecraft", sanitized
}

func (_textureRepository *TextureRepository) NormalizeTextureId(textureId string) string {
	normalized := strings.TrimSpace(textureId)

	if strings.HasPrefix(strings.ToLower(normalized), "minecraft:") {
		normalized = normalized[10:]
	}

	return strings.TrimLeft(normalized, "/\\")
}

func EnumerateLogicalPaths(pathWithinNamespace string) []string {
	if strings.TrimSpace(pathWithinNamespace) == "" {
		return nil
	}

	var paths []string
	paths = append(paths, pathWithinNamespace)

	segments := strings.Split(pathWithinNamespace, "/")
	workingSegments := segments

	if len(workingSegments) > 1 && strings.EqualFold(workingSegments[0], "textures") {
		workingSegments = workingSegments[1:]
		if len(workingSegments) > 0 {
			paths = append(paths, strings.Join(workingSegments, "/"))
		}
	}

	if len(workingSegments) > 1 {
		first := workingSegments[0]
		remainder := strings.Join(workingSegments[1:], "/")
		for _, variant := range EnumerateFolderCandidates(first) {
			if !strings.EqualFold(variant, first) {
				paths = append(paths, variant+"/"+remainder)
			}
		}
	}

	if len(workingSegments) > 0 {
		paths = append(paths, workingSegments[len(workingSegments)-1])
	}

	return paths
}

func EnumerateFolderCandidates(folder string) []string {
	var candidates []string
	candidates = append(candidates, folder)

	if strings.EqualFold(folder, "block") {
		candidates = append(candidates, "blocks")
	} else if strings.EqualFold(folder, "blocks") {
		candidates = append(candidates, "block")
	} else if strings.EqualFold(folder, "item") {
		candidates = append(candidates, "items")
	} else if strings.EqualFold(folder, "items") {
		candidates = append(candidates, "item")
	}

	return candidates
}

func (_textureRepository *TextureRepository) CreateMissingTexture() image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	magenta := color.RGBA{0xFF, 0x00, 0xFF, 0xFF}
	black := color.RGBA{0x00, 0x00, 0x00, 0xFF}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			isMagenta := (x/8+y/8)%2 == 0
			if isMagenta {
				img.Set(x, y, magenta)
			} else {
				img.Set(x, y, black)
			}
		}
	}

	return *img
}

type AnimationOverride struct {
	Frames         map[string]TextureAnimationFrame
	CacheKeySuffix string
}

func (override *AnimationOverride) GetAnimationOverrideFrame(normalizedTextureId string) (TextureAnimationFrame, bool) {
	if override == nil || override.Frames == nil {
		return TextureAnimationFrame{}, false
	}

	frame, found := override.Frames[normalizedTextureId]
	return frame, found
}

type TextureAnimationFrame struct {
	Image      image.RGBA
	DurationMs int
	FrameIndex int
}

type TextureSource interface {
	TryResolve(namespaceName string, relativePath string) (ResolvedTextureResource, bool)
}

type ResolvedTextureResource struct {
	OpenImage  func() (io.ReadCloser, error)
	OpenMcmeta func() (io.ReadCloser, error)
}

func (_textureRepository *TextureRepository) BuildSourceList(primaryRoot string, overlayRoots []string, assetNamespaces *assets.AssetNamespaceRegistry) []TextureSource {
	var sources []TextureSource

	if assetNamespaces != nil {
		for _, namespace := range assetNamespaces.GetSources() {
			sources = append(sources, NewRegistryTextureSource(namespace, assetNamespaces))
			fmt.Printf("Adding registry texture source for sourceId: %s\n", namespace)
			// Dump roots for this sourceId for diagnostics
			roots := assetNamespaces.GetRoots("minecraft", namespace)
			for _, r := range roots {
				providerType := "<nil>"
				if r.Provider != nil {
					providerType = "provider"
				}
				fmt.Printf("  root: Namespace=%s Path=%s SourceId=%s IsVanilla=%v Provider=%s\n", r.Namespace, r.Path, r.SourceId, r.IsVanilla, providerType)
			}
		}
	} else {
		// Fallback: Treat each root as a separate source to preserve order
		unorderedRoots := make(map[string]struct{})

		tryAddDirectory := func(root string) {
			if strings.TrimSpace(root) == "" {
				return
			}

			if _, err := os.Stat(root); os.IsNotExist(err) {
				fmt.Printf("Warning: Texture source directory does not exist: %s\n", root)
				return
			}

			if _, exists := unorderedRoots[root]; !exists {
				unorderedRoots[root] = struct{}{}
			}

			textureSubdirectory := root + "/textures"
			if _, err := os.Stat(textureSubdirectory); err == nil {
				if _, exists := unorderedRoots[textureSubdirectory]; !exists {
					unorderedRoots[textureSubdirectory] = struct{}{}
				}
			}
		}

		tryAddDirectory(primaryRoot)
		for _, overlayRoot := range overlayRoots {
			tryAddDirectory(overlayRoot)
		}

		for root := range unorderedRoots {
			sources = append(sources, NewDirectoryTextureSource(root))
			fmt.Printf("Adding directory texture source for root: %s\n", root)
		}
	}

	return sources
}

type RegistryTextureSource struct {
	_sourceId string
	_registry *assets.AssetNamespaceRegistry
}

func NewRegistryTextureSource(sourceId string, registry *assets.AssetNamespaceRegistry) *RegistryTextureSource {
	return &RegistryTextureSource{
		_sourceId: sourceId,
		_registry: registry,
	}
}

func (source *RegistryTextureSource) TryResolve(namespaceName string, relativePath string) (ResolvedTextureResource, bool) {
	roots := source._registry.GetRoots(namespaceName, source._sourceId)
	if len(roots) == 0 && !strings.EqualFold(namespaceName, "minecraft") {
		roots = source._registry.GetRoots("minecraft", source._sourceId)
	}

	withExtension := relativePath + ".png"
	// mcmetaExtension := withExtension + ".mcmeta"
	for _, root := range roots {
		provider := root.Provider
		if provider != nil {
			if provider.FileExists(withExtension) {
				imgPath := withExtension
				metaPath := mcmetaExtension
				resource := ResolvedTextureResource{
					OpenImage: func() (io.ReadCloser, error) {
						return provider.OpenRead(imgPath)
					},
					OpenMcmeta: func() (io.ReadCloser, error) {
						if provider.FileExists(metaPath) {
							return provider.OpenRead(metaPath)
						}
						return nil, nil
					},
				}
				return resource, true
			}

			continue
		}

		candidate := root.Path + "/" + strings.ReplaceAll(withExtension, "/", string(os.PathSeparator))
		fmt.Print("\n\nCandidate: " + candidate + "\n")
		if _, err := os.Stat(candidate); err == nil {
			path := candidate
			resource := ResolvedTextureResource{
				OpenImage: func() (io.ReadCloser, error) {
					return os.Open(path)
				},
				OpenMcmeta: func() (io.ReadCloser, error) {
					if _, err := os.Stat(path + ".mcmeta"); err == nil {
						return os.Open(path + ".mcmeta")
					}
					return nil, nil
				},
			}
			return resource, true
		}
	}

	return ResolvedTextureResource{}, false
}

type DirectoryTextureSource struct {
	_root string
}

func NewDirectoryTextureSource(root string) *DirectoryTextureSource {
	return &DirectoryTextureSource{
		_root: root,
	}
}

func (source *DirectoryTextureSource) TryResolve(namespaceName string, relativePath string) (ResolvedTextureResource, bool) {
	withExtension := strings.ReplaceAll(relativePath, "/", string(os.PathSeparator)) + ".png"
	candidate := source._root + "/" + withExtension
	if _, err := os.Stat(candidate); err == nil {
		path := candidate
		resource := ResolvedTextureResource{
			OpenImage: func() (io.ReadCloser, error) {
				return os.Open(path)
			},
			OpenMcmeta: func() (io.ReadCloser, error) {
				if _, err := os.Stat(path + ".mcmeta"); err == nil {
					return os.Open(path + ".mcmeta")
				}
				return nil, nil
			},
		}
		return resource, true
	}

	return ResolvedTextureResource{}, false
}

func TryLoadTrimPaletteColors(_textureRepository *TextureRepository) ([]color.RGBA, bool) {
	for _, resource := range _textureRepository.EnumerateResolvedResources("trims/color_palettes/trim_palette") {
		stream, err := resource.OpenImage()
		if err != nil {
			continue
		}
		defer func() {
			if closeErr := stream.Close(); closeErr != nil {
				fmt.Printf("warning: failed to close stream for trim palette resource: %v\n", closeErr)
			}
		}()

		img, err := LoadImageFromStream(stream)
		if err != nil {
			continue
		}

		if img.Bounds().Dy() <= 0 {
			continue
		}

		// Extract first row of pixels
		var colors []color.RGBA
		bounds := img.Bounds()
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, bounds.Min.Y)
			// Convert to RGBA color
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			colors = append(colors, rgba)
		}
		return colors, true
	}

	return nil, false
}

func LoadImageFromStream(stream io.ReadCloser) (image.RGBA, error) {
	img, _, err := image.Decode(stream)
	if err != nil {
		return image.RGBA{}, err
	}

	bounds := img.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, img.At(x, y))
		}
	}

	return *rgbaImg, nil
}

func (_textureRepository *TextureRepository) EnumerateResolvedResources(normalized string) []ResolvedTextureResource {
	namespaceName, pathWithinNamespace := _textureRepository.ParseNamespace(normalized)
	logicalPaths := EnumerateLogicalPaths(pathWithinNamespace)

	var resources []ResolvedTextureResource
	for _, logicalPath := range logicalPaths {
		for i := len(_textureRepository._sources) - 1; i >= 0; i-- {
			source := _textureRepository._sources[i]
			if resource, found := source.TryResolve(namespaceName, logicalPath); found {
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (_textureRepository *TextureRepository) FindColormapRoot() *string {
	output := "provider-resolved"
	for i := len(_textureRepository._sources) - 1; i >= 0; i-- {
		source := _textureRepository._sources[i]
		if _, found := source.TryResolve("minecraft", "colormap/grass"); found {
			// For directory-based sources, we need the filesystem path for backward compat.
			// For provider-based sources, we'll load colormaps via the provider.
			// Return a marker so that provider-based colormap loading is handled separately.
			return &output
		}
	}

	return nil
}

func (_textureRepository *TextureRepository) TryLoadColormapTexture(textureRelativePath string) (*image.RGBA, bool) {
	for i := len(_textureRepository._sources) - 1; i >= 0; i-- {
		if resource, found := _textureRepositor_textureRepository._sources[i].TryResolve("minecraft", textureRelativePath); found {
			stream, err := resource.OpenImage()
			if err != nil {
				continue
			}
			defer func() {
				if closeErr := stream.Close(); closeErr != nil {
					fmt.Printf("warning: failed to close stream for colormap resource: %v\n", closeErr)
				}
			}()

			img, err := LoadImageFromStream(stream)
			if err != nil {
				continue
			}

			return &img, true
		}
	}

	return nil, false
}

type TextureContentEntry struct {
	Name    string
	Texture *string
}

func (_textureRepository *TextureRepository) ProcessAnimatedTexture(normalizedKey string, spriteSheet image.RGBA, openMcmeta func() (io.ReadCloser, error)) image.RGBA {
	animation := TryBuildTextureAnimation(spriteSheet, openMcmeta)
	if animation == nil || len(animation.Frames) == 0 {
		return spriteSheet
	}

	_textureRepository._animationCache[normalizedKey] = *animation
	firstFrame := animation.Frames[0].Image
	return firstFrame
}

type TextureAnimation struct {
	Frames          []TextureAnimationFrame
	Interpolate     bool
	FrameWidth      int
	FrameHeight     int
	TotalDurationMs int
}

func NewTextureAnimation(frames []TextureAnimationFrame, interpolate bool, frameWidth int, frameHeight int) *TextureAnimation {
	totalDurationMs := 0
	for _, frame := range frames {
		durationMs := frame.DurationMs
		if durationMs < 50 {
			durationMs = 50
		}
		totalDurationMs += durationMs
	}

	return &TextureAnimation{
		Frames:          frames,
		Interpolate:     interpolate,
		FrameWidth:      frameWidth,
		FrameHeight:     frameHeight,
		TotalDurationMs: totalDurationMs,
	}
}

func (animation *TextureAnimation) GetFrameAtTime(elapsedMilliseconds int64) (TextureAnimationFrame, bool) {
	if len(animation.Frames) == 0 {
		panic("Animation does not contain frames.")
	}

	if animation.TotalDurationMs <= 0 {
		return animation.Frames[0], false
	}

	normalized := int(elapsedMilliseconds % int64(animation.TotalDurationMs))
	accumulated := 0
	for index, frame := range animation.Frames {
		duration := frame.DurationMs
		if duration < 50 {
			duration = 50
		}
		nextAccumulated := accumulated + duration
		if normalized < nextAccumulated {
			if !animation.Interpolate || len(animation.Frames) == 1 {
				return frame, false
			}

			spanWithinFrame := normalized - accumulated
			if spanWithinFrame <= 0 {
				return frame, false
			}

			progress := 0.0
			if duration > 0 {
				progress = float64(spanWithinFrame) / float64(duration)
			}
			if progress <= 0 {
				return frame, false
			}

			if progress >= 0.999 {
				nextFrameNearly := animation.Frames[(index+1)%len(animation.Frames)]
				return nextFrameNearly, false
			}

			return frame, true
		}

		accumulated = nextAccumulated
	}

	return animation.Frames[len(animation.Frames)-1], false
}

func TryBuildTextureAnimation(spriteSheet image.RGBA, openMcmeta func() (io.ReadCloser, error)) *TextureAnimation {
	if openMcmeta == nil {
		return nil
	}

	stream, err := openMcmeta()
	if err != nil {
		return nil
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close stream for animation metadata: %v\n", closeErr)
		}
	}()

	var metaData struct {
		Animation struct {
			FrameTime   float64 `json:"frametime"`
			Interpolate bool    `json:"interpolate"`
			Frames      []struct {
				Index     int     `json:"index"`
				FrameTime float64 `json:"frametime"`
			} `json:"frames"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"animation"`
	}

	if err := global.JSON.NewDecoder(stream).Decode(&metaData); err != nil {
		return nil
	}

	if metaData.Animation.FrameTime <= 0 {
		metaData.Animation.FrameTime = 1
	}

	frameWidth := metaData.Animation.Width
	if frameWidth <= 0 {
		frameWidth = spriteSheet.Bounds().Dx()
	} else if frameWidth > spriteSheet.Bounds().Dx() {
		frameWidth = spriteSheet.Bounds().Dx()
	}

	frameHeight := metaData.Animation.Height
	if frameHeight <= 0 {
		frameHeight = frameWidth
	} else if frameHeight > spriteSheet.Bounds().Dy() {
		frameHeight = spriteSheet.Bounds().Dy()
	}

	framesPerRow := max(1, spriteSheet.Bounds().Dx()/frameWidth)
	framesPerColumn := max(1, spriteSheet.Bounds().Dy()/frameHeight)
	maximumFrameIndex := max(framesPerRow*framesPerColumn-1, 0)

	var frames []TextureAnimationFrame
	for _, descriptor := range metaData.Animation.Frames {
		if descriptor.Index < 0 {
			continue
		}

		normalizedIndex := descriptor.Index
		if maximumFrameIndex > 0 {
			normalizedIndex = descriptor.Index % (maximumFrameIndex + 1)
		}
		column := normalizedIndex % framesPerRow
		row := normalizedIndex / framesPerRow
		x := column * frameWidth
		y := row * frameHeight

		if x+frameWidth > spriteSheet.Bounds().Dx() || y+frameHeight > spriteSheet.Bounds().Dy() {
			continue
		}

		frameImage := image.NewRGBA(image.Rect(0, 0, frameWidth, frameHeight))
		for fy := 0; fy < frameHeight; fy++ {
			for fx := 0; fx < frameWidth; fx++ {
				frameImage.Set(fx, fy, spriteSheet.At(x+fx, y+fy))
			}
		}

		durationMs := int(descriptor.FrameTime * 50)
		if durationMs < 50 {
			durationMs = 50
		}
		frames = append(frames, TextureAnimationFrame{
			Image:      *frameImage,
			DurationMs: durationMs,
			FrameIndex: descriptor.Index,
		})
	}

	if len(frames) == 0 {
		return nil
	}

	return NewTextureAnimation(frames, metaData.Animation.Interpolate, frameWidth, frameHeight)
}

func DecodeDataUri(dataUri string) (image.RGBA, error) {
	const prefix = "data:image/png;base64,"
	if strings.HasPrefix(strings.ToLower(dataUri), prefix) {
		base64Data := dataUri[len(prefix):]
		bytes, err := global.Base64.DecodeString(base64Data)
		if err != nil {
			return image.RGBA{}, err
		}
		img, _, err := image.Decode(strings.NewReader(string(bytes)))
		if err != nil {
			return image.RGBA{}, err
		}

		bounds := img.Bounds()
		rgbaImg := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbaImg.Set(x, y, img.At(x, y))
			}
		}

		return *rgbaImg, nil
	}

	return image.RGBA{}, fmt.Errorf("data URI does not have expected prefix")
}

func (_textureRepository *TextureRepository) TryGenerateArmorTrimTexture(normalized string) (image.RGBA, bool) {
	if !strings.HasPrefix(strings.ToLower(normalized), "trims/items/") {
		return image.RGBA{}, false
	}

	// TODO: MIGHT BE BROKEN, SHOULD BE FILE NAME
	fileName := normalized[strings.LastIndex(normalized, "/")+1:]
	if strings.TrimSpace(fileName) == "" {
		return image.RGBA{}, false
	}

	trimMarkerIndex := strings.Index(strings.ToLower(fileName), "_trim_")
	if trimMarkerIndex < 0 {
		return image.RGBA{}, false
	}

	baseOverlayName := fileName[:trimMarkerIndex+5]
	materialToken := fileName[trimMarkerIndex+6:]
	if strings.TrimSpace(baseOverlayName) == "" || strings.TrimSpace(materialToken) == "" {
		return image.RGBA{}, false
	}

	baseOverlayId := "trims/items/" + baseOverlayName
	overlayBase := _textureRepository.GetTexture(baseOverlayId)
	if overlayBase == nil || sameRGBA(*overlayBase, _textureRepository._missingTexture) {
		return image.RGBA{}, false
	}

	if _textureRepository._trimPaletteLength == 0 || len(_textureRepository._trimPaletteLookup) == 0 {
		return image.RGBA{}, false
	}

	materialPalette := _textureRepository.ResolveArmorTrimPalette(materialToken)
	if materialPalette == nil || sameRGBA(*materialPalette, _textureRepository._missingTexture) || materialPalette.Bounds().Dy() == 0 {
		return image.RGBA{}, false
	}

	materialPaletteRow := materialPalette.Pix
	if len(materialPaletteRow) == 0 {
		return image.RGBA{}, false
	}

	tinted := image.NewRGBA(overlayBase.Bounds())
	for y := overlayBase.Bounds().Min.Y; y < overlayBase.Bounds().Max.Y; y++ {
		for x := overlayBase.Bounds().Min.X; x < overlayBase.Bounds().Max.X; x++ {
			sourcePixel := overlayBase.At(x, y).(color.RGBA)
			if sourcePixel.A == 0 {
				tinted.Set(x, y, sourcePixel)
				continue
			}

			packedValue := int(sourcePixel.R)<<24 | int(sourcePixel.G)<<16 | int(sourcePixel.B)<<8 | int(sourcePixel.A)
			paletteIndex, found := _textureRepository._trimPaletteLookup[packedValue]
			if !found {
				tinted.Set(x, y, sourcePixel)
				continue
			}

			clampedIndex := paletteIndex
			if clampedIndex < 0 {
				clampedIndex = 0
			} else if clampedIndex >= len(materialPaletteRow)/4 {
				clampedIndex = len(materialPaletteRow)/4 - 1
			}
			replacement := color.RGBA{
				R: materialPaletteRow[clampedIndex*4],
				G: materialPaletteRow[clampedIndex*4+1],
				B: materialPaletteRow[clampedIndex*4+2],
				A: sourcePixel.A,
			}
			tinted.Set(x, y, replacement)
		}
	}

	return *tinted, true
}

func (_textureRepository *TextureRepository) ResolveArmorTrimPalette(materialToken string) *image.RGBA {
	for _, candidate := range EnumerateArmorTrimPaletteCandidates(materialToken) {
		palette := _textureRepository.GetTexture("trims/color_palettes/" + candidate)
		if palette != nil && sameRGBA(*palette, _textureRepository._missingTexture) {
			return palette
		}
	}

	return nil
}

func EnumerateArmorTrimPaletteCandidates(materialToken string) []string {
	var candidates []string
	if strings.TrimSpace(materialToken) == "" {
		return candidates
	}

	normalizedMaterial := strings.TrimSpace(materialToken)
	if strings.HasSuffix(strings.ToLower(normalizedMaterial), "_darker") {
		candidates = append(candidates, normalizedMaterial)
		if len(normalizedMaterial) > 7 {
			candidates = append(candidates, normalizedMaterial[:len(normalizedMaterial)-7])
		}
	} else {
		candidates = append(candidates, normalizedMaterial)
	}

	return candidates
}

func sameRGBA(a, b image.RGBA) bool {
	return a.Stride == b.Stride &&
		a.Rect == b.Rect &&
		bytes.Equal(a.Pix, b.Pix)
}

func (_textureRepository *TextureRepository) GetTintedTexture(textureId string, tint color.RGBA, strengthMultiplier float64, blend float64) *image.RGBA {
	if tint.A == 0 {
		return _textureRepository.GetTexture(textureId)
	}

	normalized := _textureRepository.NormalizeTextureId(textureId)
	cacheKey := fmt.Sprintf("%s_%02X%02X%02X%02X_%.3f_%.3f", normalized, tint.R, tint.G, tint.B, tint.A, strengthMultiplier, blend)

	if animationOverride := _textureRepository._activeAnimationOverride; animationOverride != nil && animationOverride.CacheKeySuffix != "" {
		cacheKey += "|anim:" + animationOverride.CacheKeySuffix
	}

	if cached, found := _textureRepository._cache[cacheKey]; found {
		return &cached
	}

	original := _textureRepository.GetTexture(textureId)
	if original == nil || sameRGBA(*original, _textureRepository._missingTexture) {
		_textureRepository._cache[cacheKey] = _textureRepository._missingTexture
		return &_textureRepository._missingTexture
	}

	tinted := image.NewRGBA(original.Bounds())
	strengthFactor := color.RGBA{
		R: uint8(min(float64(tint.R)*strengthMultiplier, 255)),
		G: uint8(min(float64(tint.G)*strengthMultiplier, 255)),
		B: uint8(min(float64(tint.B)*strengthMultiplier, 255)),
		A: tint.A,
	}

	for y := original.Bounds().Min.Y; y < original.Bounds().Max.Y; y++ {
		for x := original.Bounds().Min.X; x < original.Bounds().Max.X; x++ {
			pixel := original.At(x, y).(color.RGBA)
			if pixel.A == 0 {
				tinted.Set(x, y, pixel)
				continue
			}

			tintedPixel := color.RGBA{
				R: uint8((float64(pixel.R) * float64(strengthFactor.R)) / 255),
				G: uint8((float64(pixel.G) * float64(strengthFactor.G)) / 255),
				B: uint8((float64(pixel.B) * float64(strengthFactor.B)) / 255),
				A: uint8((float64(pixel.A) * float64(strengthFactor.A)) / 255),
			}

			if blend < 0.999 {
				tintedPixel = color.RGBA{
					R: uint8(float64(pixel.R)*(1-blend) + float64(tintedPixel.R)*blend),
					G: uint8(float64(pixel.G)*(1-blend) + float64(tintedPixel.G)*blend),
					B: uint8(float64(pixel.B)*(1-blend) + float64(tintedPixel.B)*blend),
					A: uint8(float64(pixel.A)*(1-blend) + float64(tintedPixel.A)*blend),
				}
			}

			tinted.Set(x, y, tintedPixel)
		}
	}

	_textureRepository._cache[cacheKey] = *tinted
	return tinted
}
