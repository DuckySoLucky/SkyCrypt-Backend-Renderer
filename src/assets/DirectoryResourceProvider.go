package assets

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DirectoryResourceProvider struct {
	rootPath string
}

func NewDirectoryResourceProvider(rootPath string) (*DirectoryResourceProvider, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("rootPath cannot be empty or whitespace")
	}

	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	return &DirectoryResourceProvider{
		rootPath: absPath,
	}, nil
}

func (d *DirectoryResourceProvider) RootPath() string {
	return d.rootPath
}

func (d *DirectoryResourceProvider) FileExists(relativePath string) bool {
	fullPath, err := d.toFullPath(relativePath)
	if err != nil {
		return false
	}
	info, err := os.Stat(fullPath)
	return err == nil && !info.IsDir()
}

func (d *DirectoryResourceProvider) DirectoryExists(relativePath string) bool {
	if strings.TrimSpace(relativePath) == "" {
		info, err := os.Stat(d.rootPath)
		return err == nil && info.IsDir()
	}

	fullPath, err := d.toFullPath(relativePath)
	if err != nil {
		return false
	}
	info, err := os.Stat(fullPath)
	return err == nil && info.IsDir()
}

func (d *DirectoryResourceProvider) OpenRead(relativePath string) (io.ReadCloser, error) {
	fullPath, err := d.toFullPath(relativePath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found in directory provider: '%s'", relativePath)
		}
		return nil, err
	}
	return file, nil
}

func (d *DirectoryResourceProvider) EnumerateFiles(directory, pattern string, recursive bool) ([]string, error) {
	return d.enumerate(directory, pattern, recursive, false)
}

func (d *DirectoryResourceProvider) EnumerateDirectories(directory, pattern string, recursive bool) ([]string, error) {
	return d.enumerate(directory, pattern, recursive, true)
}

func (d *DirectoryResourceProvider) Close() error {
	return nil
}

func (d *DirectoryResourceProvider) toFullPath(relativePath string) (string, error) {
	normalized := filepath.FromSlash(relativePath)
	combined := filepath.Join(d.rootPath, normalized)
	fullPath, err := filepath.Abs(combined)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(d.rootPath, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path '%s' resolves outside the provider root", relativePath)
	}

	return fullPath, nil
}

func (d *DirectoryResourceProvider) enumerate(directory, pattern string, recursive, wantDirs bool) ([]string, error) {
	searchDir := d.rootPath
	if strings.TrimSpace(directory) != "" {
		var err error
		searchDir, err = d.toFullPath(directory)
		if err != nil {
			return nil, err
		}
	}

	var results []string
	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !recursive && path != searchDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		if info.IsDir() == wantDirs && path != searchDir {
			matched, _ := filepath.Match(pattern, info.Name())
			if matched {
				rel, _ := filepath.Rel(d.rootPath, path)
				results = append(results, filepath.ToSlash(rel))
			}
		}
		return nil
	})

	return results, err
}

func (d *DirectoryResourceProvider) GetRelativePath(fullRelativePath string, directoryPrefix string) (string, error) {
	panic("GetRelativePath is not implemented for DirectoryResourceProvider")
}

func (d *DirectoryResourceProvider) ReadAllText(path string) (string, error) {
	panic("ReadAllText is not implemented for DirectoryResourceProvider")
}
