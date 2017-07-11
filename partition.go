package main

import (
	"path/filepath"
)

// MaxCommandBytes is the maximum number of bytes used when executing a command
const MaxCommandBytes = 32000

type partitionStrategy func([]string, []string) [][]string

func getPartitionStrategy(name string) partitionStrategy {
	switch {
	case linterTakesFiles.contains(name):
		return partitionToMaxArgSizeWithFileGlobs
	case linterTakesFilesGroupedByPackage.contains(name):
		return partitionToPackageFileGlobs
	}
	return partitionToMaxArgSize
}

func pathsToFileGlobs(paths []string) []string {
	filePaths := []string{}
	for _, dir := range paths {
		// ignore error because the glob pattern is hardcoded
		paths, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		filePaths = append(filePaths, paths...)
	}
	return filePaths
}

func partitionToMaxArgSize(cmdArgs []string, paths []string) [][]string {
	return partitionToMaxSize(cmdArgs, paths, MaxCommandBytes)
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

func partitionToMaxArgSizeWithFileGlobs(cmdArgs []string, paths []string) [][]string {
	filePaths := pathsToFileGlobs(paths)
	if len(filePaths) == 0 {
		return nil
	}
	return partitionToMaxArgSize(cmdArgs, filePaths)
}

func partitionToPackageFileGlobs(cmdArgs []string, paths []string) [][]string {
	parts := [][]string{}
	for _, path := range paths {
		filePaths := pathsToFileGlobs([]string{path})
		if len(filePaths) == 0 {
			continue
		}
		parts = append(parts, append(cmdArgs, filePaths...))
	}
	return parts
}
