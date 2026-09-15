package assets

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// ZipResourceProvider indexes ZIP entry names and reads entry contents on demand.
type ZipResourceProvider struct {
	rootPath     string
	zipFilePath  string
	archive      *zip.ReadCloser
	closeArchive bool
	entries      map[string]struct{}
	directories  map[string]struct{}
}

// NewZipResourceProvider indexes a zip file without decompressing its entries.
func NewZipResourceProvider(zipFilePath string) *ZipResourceProvider {
	if strings.TrimSpace(zipFilePath) == "" {
		panic("zipFilePath cannot be empty")
	}

	abs, err := filepath.Abs(zipFilePath)
	if err != nil {
		panic(err)
	}

	zr, err := zip.OpenReader(abs)
	if err != nil {
		panic(err)
	}
	defer zr.Close()

	entries, dirs := buildIndexFromFiles(zr.File)

	return &ZipResourceProvider{
		rootPath:    abs,
		zipFilePath: abs,
		entries:     entries,
		directories: dirs,
	}
}

// NewZipResourceProviderFromReader reads all entries from an existing zip.ReadCloser into memory.
// If ownsArchive is true the provided archive will be closed after reading.
func NewZipResourceProviderFromReader(archive *zip.ReadCloser, displayPath string, ownsArchive bool) (*ZipResourceProvider, error) {
	if archive == nil {
		return nil, errors.New("archive cannot be nil")
	}

	entries, dirs := buildIndexFromFiles(archive.File)

	return &ZipResourceProvider{
		rootPath:     displayPath,
		archive:      archive,
		closeArchive: ownsArchive,
		entries:      entries,
		directories:  dirs,
	}, nil
}

func (z *ZipResourceProvider) RootPath() string {
	return z.rootPath
}

func (z *ZipResourceProvider) FileExists(relativePath string) bool {
	normalized, err := normalizePath(relativePath)
	if err != nil {
		return false
	}
	_, ok := z.entries[normalized]
	return ok
}

func (z *ZipResourceProvider) DirectoryExists(relativePath string) bool {
	if strings.TrimSpace(relativePath) == "" {
		return true
	}
	normalized, err := normalizePath(relativePath)
	if err != nil {
		return false
	}
	normalized = strings.TrimRight(normalized, "/")
	_, ok := z.directories[normalized]
	return ok
}

func (z *ZipResourceProvider) OpenRead(relativePath string) (io.ReadCloser, error) {
	normalized, err := normalizePath(relativePath)
	if err != nil {
		return nil, err
	}
	if _, ok := z.entries[normalized]; !ok {
		return nil, fmt.Errorf("file not found in ZIP archive: '%s'", relativePath)
	}
	if z.archive != nil {
		for _, entry := range z.archive.File {
			entryPath, entryErr := normalizePath(entry.Name)
			if entryErr == nil && entryPath == normalized {
				return entry.Open()
			}
		}
		return nil, fmt.Errorf("file not found in ZIP archive: '%s'", relativePath)
	}
	archive, err := zip.OpenReader(z.zipFilePath)
	if err != nil {
		return nil, err
	}
	for _, entry := range archive.File {
		entryPath, entryErr := normalizePath(entry.Name)
		if entryErr == nil && entryPath == normalized {
			reader, openErr := entry.Open()
			if openErr != nil {
				_ = archive.Close()
				return nil, openErr
			}
			return &zipEntryReadCloser{ReadCloser: reader, archive: archive}, nil
		}
	}
	_ = archive.Close()
	return nil, fmt.Errorf("file not found in ZIP archive: '%s'", relativePath)
}

func (z *ZipResourceProvider) EnumerateFiles(directory, searchPattern string, recursive bool) ([]string, error) {
	prefix := strings.TrimRight(strings.ReplaceAll(directory, "\\", "/"), "/")
	if prefix == "." {
		prefix = ""
	}

	pattern, err := globToRegex(searchPattern)
	if err != nil {
		return nil, err
	}

	var results []string
	for path := range z.entries {
		if !isWithinDirectory(path, prefix, recursive) {
			continue
		}

		last := strings.LastIndex(path, "/")
		name := path
		if last >= 0 {
			name = path[last+1:]
		}

		if pattern.MatchString(name) {
			results = append(results, path)
		}
	}

	sort.Strings(results)
	return results, nil
}

func (z *ZipResourceProvider) EnumerateDirectories(directory, searchPattern string, recursive bool) ([]string, error) {
	prefix := strings.TrimRight(strings.ReplaceAll(directory, "\\", "/"), "/")
	if prefix == "." {
		prefix = ""
	}

	pattern, err := globToRegex(searchPattern)
	if err != nil {
		return nil, err
	}

	var results []string
	for dir := range z.directories {
		if !isWithinDirectory(dir, prefix, recursive) {
			continue
		}

		last := strings.LastIndex(dir, "/")
		name := dir
		if last >= 0 {
			name = dir[last+1:]
		}

		if pattern.MatchString(name) {
			results = append(results, dir)
		}
	}

	sort.Strings(results)
	return results, nil
}

func (z *ZipResourceProvider) Close() error {
	if z.closeArchive && z.archive != nil {
		return z.archive.Close()
	}
	return nil
}

// --- helpers ---

func buildIndexFromFiles(files []*zip.File) (map[string]struct{}, map[string]struct{}) {
	entries := make(map[string]struct{})
	dirs := make(map[string]struct{})

	for _, f := range files {
		path, err := normalizePath(f.Name)
		if err != nil {
			continue
		}
		if path == "" {
			continue
		}

		if strings.HasSuffix(path, "/") || f.FileInfo().IsDir() {
			dirPath := strings.TrimRight(path, "/")
			dirs[dirPath] = struct{}{}
			indexParentDirectories(dirPath, dirs)
			continue
		}

		entries[path] = struct{}{}
		indexParentDirectories(path, dirs)
	}

	return entries, dirs
}

type zipEntryReadCloser struct {
	io.ReadCloser
	archive *zip.ReadCloser
}

func (z *zipEntryReadCloser) Close() error {
	readerErr := z.ReadCloser.Close()
	archiveErr := z.archive.Close()
	if readerErr != nil {
		return readerErr
	}
	return archiveErr
}

func (z *ZipResourceProvider) GetRelativePath(fullRelativePath string, directoryPrefix string) (string, error) {
	normalized, err := normalizePath(fullRelativePath)
	if err != nil {
		return "", err
	}
	prefix, err := normalizePath(directoryPrefix)
	if err != nil {
		return "", err
	}
	return stripProviderDirectoryPrefix(normalized, prefix)
}

func (z *ZipResourceProvider) ReadAllText(path string) (string, error) {
	reader, err := z.OpenRead(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
