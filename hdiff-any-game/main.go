package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/hdiffz.exe
//go:embed bin/7za.exe
var embeddedFiles embed.FS

func ensureBinaries() (map[string]string, error) {
	binDir := "bin"
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		if err := os.MkdirAll(binDir, 0755); err != nil {
			return nil, err
		}
	}

	files := []string{"hdiffz.exe", "7za.exe"}
	paths := make(map[string]string)

	for _, f := range files {
		destPath := filepath.Join(binDir, f)
		paths[f] = destPath

		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			data, err := embeddedFiles.ReadFile("bin/" + f)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(destPath, data, 0755); err != nil {
				return nil, err
			}
		}
	}

	return paths, nil
}

func main() {
	paths, err := ensureBinaries()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, path := range paths {
		fmt.Println("Binary ready at:", path)
	}
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter OLD game path: ")
	oldPath, _ := reader.ReadString('\n')
	oldPath = strings.TrimSpace(oldPath)
	if oldPath == "" {
		fmt.Fprintln(os.Stderr, "no old path provided")
		os.Exit(1)
	}

	fmt.Print("Enter NEW game path: ")
	newPath, _ := reader.ReadString('\n')
	newPath = strings.TrimSpace(newPath)
	if newPath == "" {
		fmt.Fprintln(os.Stderr, "no new path provided")
		os.Exit(1)
	}

	fmt.Print("Enter zip hdiff output name: ")
	hdiffName, _ := reader.ReadString('\n')
	hdiffName = strings.TrimSpace(hdiffName)
	if hdiffName == "" {
		fmt.Fprintln(os.Stderr, "no hdiff output provided")
		os.Exit(1)
	}

	if !strings.HasSuffix(strings.ToLower(hdiffName), ".zip") {
		hdiffName += ".zip"
	}

	result, err := DiffFolders(oldPath, newPath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	hdiffFolderPath := filepath.Join(".", "hdiff")
	os.MkdirAll(hdiffFolderPath, 0755)
	if err := CopyNewFiles(newPath, result); err != nil {
		fmt.Println("Error writing diff:", err)
		return
	}
	if err := MakeHdiffFile(oldPath, newPath, result.Changed); err != nil {
		fmt.Println("Error writing diff:", err)
		return
	}

	if err := ZipWith7za(hdiffFolderPath, hdiffName); err != nil {
		fmt.Println("Error writing diff:", err)
		return
	}

	if err := RemoveFolderWithProgress(hdiffFolderPath); err != nil {
		fmt.Fprintln(os.Stderr, "error removing temp dir:", err)
		os.Exit(1)
	}

	fmt.Println("Done")
}
