package external

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/alecthomas/gometalinter/config"
)

// maxCommandBytes is the maximum number of bytes used when executing a command
const maxCommandBytes = 32000

// Partitioning function.
type partitionStrategy func(cmdArgs []string, paths []string) (partitions [][]string, err error)

func partitionStrategyFromConfig(strategy config.PartitionStrategy) partitionStrategy {
	switch strategy {
	case config.PartitionByDirectories:
		return partitionPathsAsDirectories
	case config.PartitionByFiles:
		return partitionPathsAsFiles
	case config.PartitionByPackages:
		return partitionPathsAsPackages
	case config.PartitionByFilesByPackage:
		return partitionPathsAsFilesGroupedByPackage
	case config.PartitionBySingleDirectory:
		return partitionPathsByDirectory
	}
	panic("invalid PartitionStrategy")
}

func pathsToFileGlobs(paths []string) ([]string, error) {
	filePaths := []string{}
	for _, dir := range paths {
		paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			return nil, err
		}
		filePaths = append(filePaths, paths...)
	}
	return filePaths, nil
}

func partitionPathsAsDirectories(cmdArgs []string, paths []string) ([][]string, error) {
	return partitionToMaxSize(cmdArgs, paths, maxCommandBytes), nil
}

func partitionToMaxSize(cmdArgs []string, paths []string, maxSize int) [][]string {
	partitions := newSizePartitioner(cmdArgs, maxSize)
	for _, path := range paths {
		partitions.add(path)
	}
	return partitions.end()
}

type sizePartitioner struct {
	base    []string
	parts   [][]string
	current []string
	size    int
	max     int
}

func newSizePartitioner(base []string, max int) *sizePartitioner {
	p := &sizePartitioner{base: base, max: max}
	p.new()
	return p
}

func (p *sizePartitioner) add(arg string) {
	if p.size+len(arg)+1 > p.max {
		p.new()
	}
	p.current = append(p.current, arg)
	p.size += len(arg) + 1
}

func (p *sizePartitioner) new() {
	p.end()
	p.size = 0
	p.current = []string{}
	for _, arg := range p.base {
		p.add(arg)
	}
}

func (p *sizePartitioner) end() [][]string {
	if len(p.current) > 0 {
		p.parts = append(p.parts, p.current)
	}
	return p.parts
}

func partitionPathsAsFiles(cmdArgs []string, paths []string) ([][]string, error) {
	filePaths, err := pathsToFileGlobs(paths)
	if err != nil || len(filePaths) == 0 {
		return nil, err
	}
	return partitionPathsAsDirectories(cmdArgs, filePaths)
}

func partitionPathsAsFilesGroupedByPackage(cmdArgs []string, paths []string) ([][]string, error) {
	parts := [][]string{}
	for _, path := range paths {
		filePaths, err := pathsToFileGlobs([]string{path})
		if err != nil {
			return nil, err
		}
		if len(filePaths) == 0 {
			continue
		}
		parts = append(parts, append(cmdArgs, filePaths...))
	}
	return parts, nil
}

func partitionPathsAsPackages(cmdArgs []string, paths []string) ([][]string, error) {
	packagePaths, err := pathsToPackagePaths(paths)
	if err != nil || len(packagePaths) == 0 {
		return nil, err
	}
	return partitionPathsAsDirectories(cmdArgs, packagePaths)
}

func pathsToPackagePaths(paths []string) ([]string, error) {
	packages := []string{}

	for _, path := range paths {
		pkg, err := packageNameFromPath(path)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func packageNameFromPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}
	for _, gopath := range getGoPathList() {
		rel, err := filepath.Rel(filepath.Join(gopath, "src"), path)
		if err != nil {
			continue
		}
		return rel, nil
	}
	return "", fmt.Errorf("%s not in GOPATH", path)
}

func partitionPathsByDirectory(cmdArgs []string, paths []string) ([][]string, error) {
	parts := [][]string{}
	for _, path := range paths {
		parts = append(parts, append(cmdArgs, path))
	}
	return parts, nil
}

// Go 1.8 compatible GOPATH.
func getGoPath() string {
	path := os.Getenv("GOPATH")
	if path == "" {
		user, err := user.Current()
		if err != nil {
			panic(err)
		}
		path = filepath.Join(user.HomeDir, "go")
	}
	return path
}

func getGoPathList() []string {
	return strings.Split(getGoPath(), string(os.PathListSeparator))
}
