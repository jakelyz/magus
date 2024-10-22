/*
MIT License

Copyright (c) 2024 Jake Lilly

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	DEFAULT_SOURCE_DIRECTORY = "dotfiles"
	EXIT_ERROR = 1
	ABSENT = "ABSENT"
	PRESENT = "PRESENT"
	MISMATCH = "MISMATCH"
)

func main() {
	opts := &Options{
		Source: DEFAULT_SOURCE_DIRECTORY,
		Target: getHomeDirectory(),
	}
	if err := opts.Parse(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing options: %s\n", err)
		os.Exit(EXIT_ERROR)
	}

	cmd := os.Args[1]
	if cmd != "conjure" && cmd != "expel" && cmd != "peer" {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(EXIT_ERROR)
	}

	files, err := os.ReadDir(opts.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory %s: %s", opts.Source, err)
		os.Exit(EXIT_ERROR)
	}
	packages := getPackages(files)
	for idx, pkg := range packages {
		foundPaths := getFiles(fmt.Sprintf("%s/%s", opts.Source, pkg.name))
		for _, path := range foundPaths {
			pkgFile := PackageFile{
				path: path,
			}
			relativePath := fmt.Sprintf("%s/%s/%s", opts.Source, pkg.name, path)
			content, err := os.ReadFile(relativePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %s\n", relativePath, err)
				os.Exit(EXIT_ERROR)
			}
			pkgFile.md5 = getHash(content)
			state, err := determineState(pkgFile.path, pkgFile.md5, opts.Target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error determining current state of file %s in package \"%s\": %s", pkgFile.path, pkg.name, err)
				os.Exit(EXIT_ERROR)
			}
			pkgFile.state = state
			packages[idx].addFile(pkgFile)
		}
	}

	switch cmd {
	case "conjure":
		if err := Conjure(packages, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Conjure command failed with error: %s\n", err)
			os.Exit(EXIT_ERROR)
		}
	case "expel":
		if err := Expel(packages, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Expel command failed with error: %s\n", err)
			os.Exit(EXIT_ERROR)
		}
	case "peer":
		if err := Peer(packages, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Peer command failed with error: %s\n", err)
			os.Exit(EXIT_ERROR)
		}
	}
}

// Options struct
type Options struct {
	Source string
	Target string
}

func (opts *Options) Parse(cmd string, args []string) error {
	cli := flag.NewFlagSet(cmd, flag.ExitOnError)
	cli.StringVar(&opts.Source, "source", opts.Source, "Source path for packages")
	cli.StringVar(&opts.Target, "target", opts.Target, "Target destination for packages")
	return cli.Parse(args)
}

// Package struct
type Package struct {
	name  string
	files []PackageFile
	state string
}

func (p *Package) addFile(pf PackageFile) []PackageFile {
	p.files = append(p.files, pf)
	return p.files
}

type PackageFile struct {
	path string
	md5 string
	state string
}

func getHomeDirectory() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determing home directory: %s", err)
		os.Exit(EXIT_ERROR)
	}
	return dir
}

func getPackages(entries []fs.DirEntry) []Package {
	pkgs := make([]Package, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkg := Package{
			name: entry.Name(),
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

func getFiles(root string) []string {
	files := make([]string, 0)
	filepath.WalkDir(root, func(p string, file fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !file.IsDir() {
			relativePath, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			files = append(files, relativePath)
		}
		return nil
	})
	return files
}

func getHash(content []byte) string {
	hash := md5.New()
	hash.Write(content)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func determineState(file string, md5 string, targetPath string) (string, error) {
	targetFile := fmt.Sprintf("%s/%s", targetPath, file)
	if !checkFileExists(targetFile) {
		return ABSENT, nil
	}
	content, err := os.ReadFile(targetFile)
	if err != nil {
		return "", err
	}
	if !(md5 == getHash(content)) {
		return MISMATCH, nil
	}
	return PRESENT, nil
}

func checkFileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func copyFile(pkgFile PackageFile, pkgName string, sourcePath string, targetPath string) error {
	fullSourceFilePath := fmt.Sprintf("%s/%s/%s", sourcePath, pkgName, pkgFile.path)
	srcContent, err := os.ReadFile(fullSourceFilePath)
	if err != nil {
		return err
	}
	fullTargetfilePath := fmt.Sprintf("%s/%s", targetPath, pkgFile.path)
	fullTargetFileParents := filepath.Dir(fullTargetfilePath)
	if err := os.MkdirAll(fullTargetFileParents, 0777); err != nil {
		return err
	}
	if err := os.WriteFile(fullTargetfilePath, srcContent, 0666); err != nil {
		return err
	}
	return nil
}

func rmFile(pkgFile PackageFile, targetPath string) error {
	fullTargetfilePath := fmt.Sprintf("%s/%s", targetPath, pkgFile.path)
	if err := os.Remove(fullTargetfilePath); err != nil {
		return err
	}
	return nil
}

// Conjure subcommand
func Conjure(pkgs []Package, opts *Options) error {
	for _, pkg := range pkgs {
		fmt.Fprintf(os.Stdout, ".:. Conjuring %s\n", pkg.name)
		for _, fn := range pkg.files {
			if fn.state == ABSENT || fn.state == MISMATCH {
				if err := copyFile(fn, pkg.name, opts.Source, opts.Target); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Expel subcommand
func Expel(pkgs []Package, opts *Options) error {
	for _, pkg := range pkgs {
		fmt.Fprintf(os.Stdout, ".:. Expelling %s\n", pkg.name)
		 for _, fn := range pkg.files {
			 if fn.state == PRESENT {
				if err := rmFile(fn, opts.Target); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Peer subcommand
func Peer(pkgs []Package, opts *Options) error {
	fmt.Fprintf(os.Stdout, ".:. Peering Packages\n")
	for _, pkg := range pkgs {
		fmt.Fprintf(os.Stdout, "  %s\n", pkg.name)
		for _, fn := range pkg.files {
			fmt.Fprintf(os.Stdout, "    (%s) %s\n", fn.state, fn.path)
		}
	}
	return nil
}
