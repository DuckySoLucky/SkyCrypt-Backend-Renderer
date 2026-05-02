package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AssetNamespaceRegistry struct {
	_roots            []AssetNamespaceRoot
	_rootsByNamespace map[string][]AssetNamespaceRoot
	_deduplicationSet map[string]struct{}
}

func (r *AssetNamespaceRegistry) AddNamespace(namespace string, path string, sourceId string, isVanilla bool) {
	if strings.TrimSpace(namespace) == "" {
		namespace = "minecraft"
	}

	if strings.TrimSpace(path) == "" {
		return
	}

	fullPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return
	}

	provider := &DirectoryResourceProvider{rootPath: fullPath}
	r.AddNamespaceWithProvider(namespace, fullPath, sourceId, isVanilla, provider)
}

func (r *AssetNamespaceRegistry) AddNamespaceWithProvider(namespace string, displayPath string, sourceId string, isVanilla bool, provider ResourceProvider) {
	if strings.TrimSpace(namespace) == "" {
		namespace = "minecraft"
	}

	if strings.TrimSpace(displayPath) == "" {
		return
	}

	identity := fmt.Sprintf("%s|%s", strings.ToLower(namespace), strings.ToLower(displayPath))
	if _, exists := r._deduplicationSet[identity]; exists {
		return
	}

	r._deduplicationSet[identity] = struct{}{}

	root := AssetNamespaceRoot{
		Namespace: namespace,
		Path:      displayPath,
		SourceId:  sourceId,
		IsVanilla: isVanilla,
		Provider:  provider,
	}

	r._roots = append(r._roots, root)
	r._rootsByNamespace[namespace] = append(r._rootsByNamespace[namespace], root)
}

func (_assetNamespaceRegistry *AssetNamespaceRegistry) GetRoots(namespace string, sourceId ...string) []AssetNamespaceRoot {
	if strings.TrimSpace(namespace) == "" {
		namespace = "minecraft"
	}

	roots := _assetNamespaceRegistry._rootsByNamespace[namespace]
	if len(sourceId) == 0 {
		return roots
	}

	filter := sourceId[0]
	var results []AssetNamespaceRoot
	for _, root := range roots {
		if strings.EqualFold(root.SourceId, filter) {
			results = append(results, root)
		}
	}

	return results
}

func (r *AssetNamespaceRegistry) ResolveRoots(namespace string, fallbackToMinecraft bool) []AssetNamespaceRoot {
	roots := r.GetRoots(namespace)
	if len(roots) == 0 && fallbackToMinecraft && namespace != "minecraft" {
		roots = r.GetRoots("minecraft")
	}

	return roots
}

func (r *AssetNamespaceRegistry) EnumerateCandidatePaths(namespace string, relativePath string, preferOverrides bool) []string {
	roots := r.ResolveRoots(namespace, true)
	if len(roots) == 0 {
		return []string{}
	}

	var paths []string
	if preferOverrides {
		for i := len(roots) - 1; i >= 0; i-- {
			paths = append(paths, fmt.Sprintf("%s/%s", roots[i].Path, relativePath))
		}

		return paths
	}

	for _, root := range roots {
		paths = append(paths, fmt.Sprintf("%s/%s", root.Path, relativePath))
	}

	return paths
}

func (_assetNamespaceRegistry *AssetNamespaceRegistry) GetSources() []string {
	sources := make([]string, 0)
	seen := make(map[string]struct{})
	for _, root := range _assetNamespaceRegistry._roots {
		if _, exists := seen[root.SourceId]; !exists {
			seen[root.SourceId] = struct{}{}
			sources = append(sources, root.SourceId)
		}
	}

	return sources
}

func (_assetNamespaceRegistry *AssetNamespaceRegistry) Roots() []AssetNamespaceRoot {
	return _assetNamespaceRegistry._roots
}

type AssetNamespaceRoot struct {
	Namespace string
	Path      string
	SourceId  string
	IsVanilla bool
	Provider  ResourceProvider
}

func NewAssetNamespaceRegistry() *AssetNamespaceRegistry {
	return &AssetNamespaceRegistry{
		_roots:            []AssetNamespaceRoot{},
		_rootsByNamespace: make(map[string][]AssetNamespaceRoot),
		_deduplicationSet: make(map[string]struct{}),
	}
}
