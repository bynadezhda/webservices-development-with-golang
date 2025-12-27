package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Element struct {
	Entry  fs.DirEntry // данный объект
	Path   string      // полный путь к текущему объекту (файлу ИЛИ директории)
	IsLast bool        // является ли он последним элементом в папке
	Prefix string      // табуляция, которую нужно вывести перед этим объектом, наследуется от предков
}

func calculatePrefix(element Element) string {
	if element.Entry == nil {
		return ""
	}

	tab := "│\t"
	if element.IsLast {
		tab = "\t"
	}

	return element.Prefix + tab
}

func calculateNearestTab(isLast bool) string {
	if isLast {
		return "└───"
	}

	return "├───"
}

func processFile(element Element, out io.Writer, printFiles bool) error {
	if !printFiles {
		return nil
	}

	fileInfo, err := element.Entry.Info()
	if err != nil {
		return err
	}

	sizeInfo := "empty"
	if fileInfo.Size() > 0 {
		sizeInfo = fmt.Sprintf("%db", fileInfo.Size())
	}

	nearestTab := calculateNearestTab(element.IsLast)
	fmt.Fprintf(out, "%s%s%s (%s)\n", element.Prefix, nearestTab, element.Entry.Name(), sizeInfo)

	return nil
}

func iterateFiles(element Element, files []fs.DirEntry, out io.Writer, printFiles bool) error {
	for i, file := range files {
		child := Element{
			Entry:  file,
			Path:   filepath.Join(element.Path, file.Name()),
			IsLast: i == len(files)-1,
			Prefix: calculatePrefix(element),
		}

		if err := processEntry(child, out, printFiles); err != nil {
			return err
		}
	}
	return nil
}

func filterDirs(files []fs.DirEntry) []fs.DirEntry {
	onlyDirs := make([]fs.DirEntry, 0, len(files))

	for _, file := range files {
		if file.IsDir() {
			onlyDirs = append(onlyDirs, file)
		}
	}

	return onlyDirs
}

func processDir(element Element, out io.Writer, printFiles bool) error {
	nearestTab := calculateNearestTab(element.IsLast)

	if element.Entry != nil {
		fmt.Fprintf(out, "%s%s%s\n", element.Prefix, nearestTab, element.Entry.Name())
	}

	readPath := element.Path
	if readPath == "" {
		readPath = "."
	}
	files, err := os.ReadDir(readPath)

	if err != nil {
		return err
	}

	if !printFiles {
		files = filterDirs(files)
	}

	return iterateFiles(element, files, out, printFiles)
}

func processEntry(element Element, out io.Writer, printFiles bool) error {
	if element.Entry == nil || element.Entry.IsDir() {
		return processDir(element, out, printFiles)
	}
	return processFile(element, out, printFiles)
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	root := Element{
		Entry:  nil,
		Path:   path,
		IsLast: true,
	}

	return processEntry(root, out, printFiles)
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: go run main.go <path> [-f]")
		os.Exit(1)
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"

	if err := dirTree(os.Stdout, path, printFiles); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
