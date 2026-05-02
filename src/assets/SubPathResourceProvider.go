package assets

import (
	"io"
	"strings"
)

type SubPathResourceProvider struct {
	inner     ResourceProvider
	prefix    string
	OwnsInner bool
}

func NewSubPathResourceProvider(inner ResourceProvider, prefix string) *SubPathResourceProvider {
	if inner == nil {
		panic("inner cannot be nil")
	}

	normalizedPrefix := normalizeSubPath(prefix)
	if normalizedPrefix != "" && !strings.HasSuffix(normalizedPrefix, "/") {
		normalizedPrefix += "/"
	}

	return &SubPathResourceProvider{
		inner:  inner,
		prefix: normalizedPrefix,
	}
}

func (s *SubPathResourceProvider) RootPath() string {
	if s.prefix == "" {
		return s.inner.RootPath()
	}

	return strings.TrimRight(s.inner.RootPath(), "/\\") + "/" + strings.TrimRight(s.prefix, "/")
}

func (s *SubPathResourceProvider) FileExists(path string) bool {
	return s.inner.FileExists(s.prefix + normalizeSubPath(path))
}

func (s *SubPathResourceProvider) DirectoryExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return s.inner.DirectoryExists(strings.TrimRight(s.prefix, "/"))
	}

	return s.inner.DirectoryExists(s.prefix + normalizeSubPath(path))
}

func (s *SubPathResourceProvider) OpenRead(path string) (io.ReadCloser, error) {
	return s.inner.OpenRead(s.prefix + normalizeSubPath(path))
}

func (s *SubPathResourceProvider) EnumerateFiles(dir string, pattern string, recursive bool) ([]string, error) {
	innerDir := strings.TrimRight(s.prefix, "/")
	if strings.TrimSpace(dir) != "" {
		innerDir = s.prefix + normalizeSubPath(dir)
	}

	paths, err := s.inner.EnumerateFiles(innerDir, pattern, recursive)
	if err != nil {
		return nil, err
	}

	return s.stripPrefixedPaths(paths), nil
}

func (s *SubPathResourceProvider) EnumerateDirectories(dir string, pattern string, recursive bool) ([]string, error) {
	innerDir := strings.TrimRight(s.prefix, "/")
	if strings.TrimSpace(dir) != "" {
		innerDir = s.prefix + normalizeSubPath(dir)
	}

	paths, err := s.inner.EnumerateDirectories(innerDir, pattern, recursive)
	if err != nil {
		return nil, err
	}

	return s.stripPrefixedPaths(paths), nil
}

func (s *SubPathResourceProvider) Close() error {
	if s.OwnsInner {
		return s.inner.Close()
	}

	return nil
}

func (s *SubPathResourceProvider) stripPrefixedPaths(paths []string) []string {
	if s.prefix == "" {
		return paths
	}

	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if stripped, ok := s.stripPrefix(path); ok {
			result = append(result, stripped)
		}
	}

	return result
}

func (s *SubPathResourceProvider) stripPrefix(path string) (string, bool) {
	if s.prefix == "" {
		return path, true
	}

	if strings.HasPrefix(strings.ToLower(path), strings.ToLower(s.prefix)) {
		return path[len(s.prefix):], true
	}

	return "", false
}

func normalizeSubPath(path string) string {
	return strings.TrimLeft(strings.ReplaceAll(path, "\\", "/"), "/")
}

func (s *SubPathResourceProvider) ReadAllText(path string) (string, error) {
	panic("ReadAllText is not implemented for SubPathResourceProvider")
}

func (s *SubPathResourceProvider) GetRelativePath(fullRelativePath string, directoryPrefix string) (string, error) {
	panic("GetRelativePath is not implemented for SubPathResourceProvider")
}
