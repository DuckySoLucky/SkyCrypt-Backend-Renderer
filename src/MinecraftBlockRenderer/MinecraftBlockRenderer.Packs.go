package minecraftblockrenderer

import (
	texturepacks "duckysolucky/gorenderer/src/TexturePacks"
	"duckysolucky/gorenderer/src/assets"
	"duckysolucky/gorenderer/src/data"
	"os"
	"strings"
)

var VanillaPackId = "vanilla"

type ItemRenderCapture struct {
	OriginalTarget    string
	NormalizedItemKey string
	ItemInfo          *data.ItemInfo
	Model             *data.BlockModelInstance
	ModelCandidates   []string
	ResolvedModelName *string
	FinalOptions      BlockRenderOptions
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) NormalizeModelIdentifier(identifier string) string {
	if identifier == "" {
		return identifier
	}

	trimmed := strings.TrimSpace(identifier)
	if strings.Contains(trimmed, ":") {
		return trimmed
	}

	trimmed = strings.TrimLeft(trimmed, "/")
	if trimmed == "" {
		return "minecraft:"
	}

	return "minecraft:" + trimmed
}

type RenderPackContext struct {
	AssetsRoot      string
	OverlayRoots    []OverlayRoot
	PackIds         []string
	PackStackHash   string
	Packs           []texturepacks.RegisteredResourcePack
	SearchRoots     []OverlaySearchRoot
	AssetNamespaces assets.AssetNamespaceRegistry
}

type OverlaySearchRoot struct {
	Path     string
	SourceId string
	Kind     string
}

func NewRenderPackContext(assetsRoot string, overlayRoots []OverlayRoot, packIds []string, packStackHash string, packs []texturepacks.RegisteredResourcePack, assetNamespaces *assets.AssetNamespaceRegistry) *RenderPackContext {
	context := &RenderPackContext{
		AssetsRoot:    assetsRoot,
		OverlayRoots:  overlayRoots,
		PackIds:       packIds,
		PackStackHash: packStackHash,
		Packs:         packs,
	}

	if assetNamespaces == nil {
		context.AssetNamespaces = *context.BuildAssetNamespaces()
	} else {
		context.AssetNamespaces = *assetNamespaces
	}

	return context

}

func (_renderPackContext *RenderPackContext) BuildAssetNamespaces() *assets.AssetNamespaceRegistry {
	registry := assets.NewAssetNamespaceRegistry()
	if strings.TrimSpace(_renderPackContext.AssetsRoot) != "" {
		_renderPackContext.RegisterNamespaceRoot(registry, "minecraft", _renderPackContext.AssetsRoot, VanillaPackId, true)
	}

	for _, overlay := range _renderPackContext.OverlayRoots {
		_renderPackContext.AddOverlayNamespaces(registry, overlay)
	}

	// Register provider-backed pack namespaces (zip-backed packs whose overlay paths
	// don't exist on the filesystem and were skipped by AddOverlayNamespaces above)
	for _, pack := range _renderPackContext.Packs {
		if pack.NamespaceProviders == nil {
			continue
		}

		_renderPackContext.RegisterProviderNamespaces(registry, pack)
	}

	return registry
}

func (_renderPackContext *RenderPackContext) RegisterNamespaceRoot(registry *assets.AssetNamespaceRegistry, namespaceName string, path string, sourceId string, isVanilla bool) {
	registry.AddNamespace(namespaceName, path, sourceId, isVanilla)
	texturesPath := path + "/textures"
	if fi, err := os.Stat(texturesPath); err == nil && fi.IsDir() {
		registry.AddNamespace(namespaceName, texturesPath, sourceId, isVanilla)
	}
}

func (_renderPackContext *RenderPackContext) AddOverlayNamespaces(registry *assets.AssetNamespaceRegistry, overlay OverlayRoot) {
	if strings.TrimSpace(overlay.Path) == "" {
		return
	}

	if fi, err := os.Stat(overlay.Path); err != nil || !fi.IsDir() {
		return
	}

	normalized, err := os.Getwd()
	if err != nil {
		return
	}

	assetsDirectory := normalized + "/assets"
	if fi, err := os.Stat(assetsDirectory); err == nil && fi.IsDir() {
		files, err := os.ReadDir(assetsDirectory)
		if err == nil {
			for _, file := range files {
				if file.IsDir() {
					namespaceName := file.Name()
					registry.AddNamespace(namespaceName, assetsDirectory+"/"+namespaceName, overlay.SourceId, overlay.Kind == "vanilla")
				}
			}
		}
		return
	}

	registry.AddNamespace("minecraft", normalized, overlay.SourceId, overlay.Kind == "vanilla")
}

func (_renderPackContext *RenderPackContext) RegisterProviderNamespaces(registry *assets.AssetNamespaceRegistry, pack texturepacks.RegisteredResourcePack) {
	if pack.NamespaceProviders == nil {
		return
	}

	for namespaceName, nsProvider := range pack.NamespaceProviders {
		displayPath := pack.RootPath
		registry.AddNamespaceWithProvider(namespaceName, displayPath, pack.Id, false, nsProvider)

		if nsProvider.DirectoryExists("textures") {
			texturesProvider := assets.NewSubPathResourceProvider(nsProvider, "textures")
			registry.AddNamespaceWithProvider(namespaceName, displayPath+"/textures", pack.Id, false, texturesProvider)
		}
	}

	// Register catharsis overlay namespace providers (higher priority, registered after base)
	if pack.OverlayNamespaceProviders != nil {
		for _, overlay := range pack.OverlayNamespaceProviders {
			namespaceName := overlay.Namespace
			displayPath := overlay.DisplayPath
			nsProvider := overlay.Provider

			registry.AddNamespaceWithProvider(namespaceName, displayPath, pack.Id, false, nsProvider)

			if nsProvider.DirectoryExists("textures") {
				texturesProvider := assets.NewSubPathResourceProvider(nsProvider, "textures")
				registry.AddNamespaceWithProvider(namespaceName, displayPath+"/textures", pack.Id, false, texturesProvider)
			}
		}
	}
}

func RenderPackContextCreate(assetsDirectory *string, baseOverlayRoots []OverlayRoot, packStack *texturepacks.TexturePackStack) *RenderPackContext {
	overlays := make([]OverlayRoot, len(baseOverlayRoots))
	copy(overlays, baseOverlayRoots)

	if packStack != nil {
		for _, overlay := range packStack.OverlayRoots {
			overlays = append(overlays, OverlayRoot{
				Path:     overlay.Path,
				SourceId: overlay.PackId,
				Kind:     "resource_pack",
			})
		}
	}

	assetsRoot := ""
	if assetsDirectory != nil {
		assetsRoot = *assetsDirectory
	}

	return NewRenderPackContext(assetsRoot, overlays, nil, "", nil, nil)

}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) ResolveRendererForOptions(options BlockRenderOptions) (*MinecraftBlockRenderer, BlockRenderOptions) {
	var forwardedOptions BlockRenderOptions
	if _minecraftBlockRenderer._packRegistry == nil || options.PackIds == nil || len(options.PackIds) == 0 {
		forwardedOptions = options
		return _minecraftBlockRenderer, forwardedOptions
	}

	if len(_minecraftBlockRenderer._packContext.PackIds) > 0 && PackSequencesEqual(options.PackIds, _minecraftBlockRenderer._packContext.PackIds) {
		forwardedOptions = options
		forwardedOptions.PackIds = nil
		return _minecraftBlockRenderer, forwardedOptions
	}

	renderer := _minecraftBlockRenderer.GetRendererForPackStack(options.PackIds)
	forwardedOptions = options
	forwardedOptions.PackIds = nil
	return renderer, forwardedOptions
}

func PackSequencesEqual(candidate []string, baseline []string) bool {
	if len(candidate) != len(baseline) {
		return false
	}

	for i := 0; i < len(candidate); i++ {
		if !strings.EqualFold(candidate[i], baseline[i]) {
			return false
		}
	}

	return true
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) GetRendererForPackStack(packIds []string) *MinecraftBlockRenderer {
	if _minecraftBlockRenderer._packRegistry == nil {
		panic("This renderer was created without a texture pack registry and cannot resolve pack combinations.")
	}

	if strings.TrimSpace(_minecraftBlockRenderer._packContext.AssetsRoot) == "" {
		panic("Texture pack rendering requires a renderer created from Minecraft assets (not aggregated data files).")
	}

	packStack := _minecraftBlockRenderer._packRegistry.BuildPackStack(packIds)
	return _minecraftBlockRenderer.GetRendererForPackStackWithContext(packStack)
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) GetRendererForPackStackWithContext(packStack *texturepacks.TexturePackStack) *MinecraftBlockRenderer {
	cacheKey := packStack.Fingerprint
	if cached, exists := _minecraftBlockRenderer._packRendererCache[cacheKey]; exists {
		return &cached
	}

	renderer := _minecraftBlockRenderer.CreatePackRenderer(packStack)
	_minecraftBlockRenderer._packRendererCache[cacheKey] = *renderer
	return renderer
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) CreatePackRenderer(packStack *texturepacks.TexturePackStack) *MinecraftBlockRenderer {
	packContext := RenderPackContextCreate(&_minecraftBlockRenderer._assetsDirectory, _minecraftBlockRenderer._baseOverlayRoots, packStack)

	overlayPaths := make([]string, len(packContext.OverlayRoots))
	for i, root := range packContext.OverlayRoots {
		overlayPaths[i] = root.Path
	}

	modelResolver := data.BlockModelResolverInstance.LoadFromMinecraftAssets(_minecraftBlockRenderer._assetsDirectory, &overlayPaths, &packContext.AssetNamespaces)
	blockRegistry := data.BlockRegistryInstance.LoadFromMinecraftAssets(_minecraftBlockRenderer._assetsDirectory, modelResolver.Definitions, overlayPaths, &packContext.AssetNamespaces)
	itemRegistry := data.ItemRegistryInstance.LoadFromMinecraftAssets(_minecraftBlockRenderer._assetsDirectory, modelResolver.Definitions, overlayPaths, &packContext.AssetNamespaces)

	return NewMinecraftBlockRenderer(modelResolver, nil, blockRegistry, itemRegistry, _minecraftBlockRenderer._packContext.AssetsRoot, _minecraftBlockRenderer._packContext.OverlayRoots, nil, *packContext)
}

type PacksOverlayRoot struct {
	Path     string
	SourceId string
	Kind     string
}

type PacksOverlayRootKind = string

const (
	OverlayRootKind_CustomData   PacksOverlayRootKind = "custom_data"
	OverlayRootKind_ResourcePack PacksOverlayRootKind = "resource_pack"
	OverlayRootKind_Vanilla      PacksOverlayRootKind = "vanilla"
)
