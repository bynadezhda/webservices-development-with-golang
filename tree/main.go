package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
)

type Element struct {
	Entry       fs.DirEntry // данный объект
	FilePath    string      // путь до данного объекта
	IsLast      bool        // является ли он последним элементом в папке
	PreviousTab string      // табуляция, которую нужно вывести перед этим объектом, наследуется от предков
}

func calculatePreviousTab(element Element, isLast bool) string {
	if element.Entry == nil {
		return ""
	}

	tab := "│	"
	if element.IsLast {
		tab = "	"
	}

	return element.PreviousTab + tab
}

func calculateNearestTab(isLast bool) string {
	if isLast {
		return "└───"
	} else {
		return "├───"
	}
}

func calculateSizeInfo(element Element) (string, error) {
	var sizeInfo string = "empty"
	fileInfo, err := element.Entry.Info()

	if err != nil {
		return "", err
	}

	if fileInfo.Size() > 0 {
		sizeInfo = strconv.FormatInt(fileInfo.Size(), 10) + "b"
	}

	return sizeInfo, nil
}

func proccessFile(element Element, out io.Writer, printFiles bool) error {
	if !printFiles {
		return nil
	}

	sizeInfo, err := calculateSizeInfo(element)

	if err != nil {
		return err
	}

	neariestTab := calculateNearestTab(element.IsLast)
	fmt.Fprintf(out, "%s%s%s (%s)\n", element.PreviousTab, neariestTab, element.Entry.Name(), sizeInfo)

	return nil
}

func iterateFiles(element Element, files []fs.DirEntry, out io.Writer, printFiles bool) {
	for i, file := range files {
		var newElement Element
		newElement.Entry = file
		newElement.FilePath = element.FilePath + file.Name() + "/"

		if i == len(files)-1 {
			newElement.IsLast = true
		} else {
			newElement.IsLast = false
		}

		newElement.PreviousTab = calculatePreviousTab(element, newElement.IsLast)

		allPrintFile(newElement, out, printFiles)
	}
}

func chooseOnlyDirs(files []fs.DirEntry) []fs.DirEntry {
	var onlyDirs []fs.DirEntry

	for _, file := range files {
		if file.IsDir() {
			onlyDirs = append(onlyDirs, file)
		}
	}

	return onlyDirs
}

func proccessDir(element Element, out io.Writer, printFiles bool) error {
	neariestTab := calculateNearestTab(element.IsLast)

	if element.Entry != nil {
		fmt.Fprintf(out, "%s%s%s\n", element.PreviousTab, neariestTab, element.Entry.Name())
	}

	var readDirPath string

	if element.FilePath == "" {
		readDirPath = "."
	} else {
		readDirPath = element.FilePath
	}

	files, err := os.ReadDir(readDirPath)

	if err != nil {
		return err
	}

	if !printFiles {
		files = chooseOnlyDirs(files)
	}

	iterateFiles(element, files, out, printFiles)

	return nil
}

func allPrintFile(element Element, out io.Writer, printFiles bool) error {
	if element.Entry == nil || element.Entry.IsDir() {
		if err := proccessDir(element, out, printFiles); err != nil {
			return err
		}
	} else {
		if err := proccessFile(element, out, printFiles); err != nil {
			return err
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	var nullElement = Element{nil, path + "/", true, ""}
	if err := allPrintFile(nullElement, out, printFiles); err != nil {
		return err
	}

	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
