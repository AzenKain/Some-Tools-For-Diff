package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/hdiffz.exe
var hdiffz []byte

//go:embed bin/7za.exe
var sevenZip []byte

func main() {
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

	fmt.Println("Diffing folders...")
	result, err := DiffFolders(oldPath, newPath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println()
	fmt.Println("Diffing folders done.")

	hdiffFolderPath := filepath.Join(".", "hdiff")
	os.MkdirAll(hdiffFolderPath, 0755)

	fmt.Println("Copying new files...")
	if err := CopyNewFiles(newPath, result); err != nil {
		fmt.Println("Error writing diff:", err)
		return
	}
	fmt.Println()
	fmt.Println("Copying new files done.")

	fmt.Println("Making hdiff files...")
	if err := MakeHdiffFile(oldPath, newPath, result.Changed); err != nil {
		fmt.Println("Error writing diff:", err)
		return
	}
	fmt.Println()
	fmt.Println("Making hdiff files done.")

	fmt.Println("Zipping hdiff files...")
	if err := ZipWith7za(hdiffFolderPath, hdiffName); err != nil {
		fmt.Println("Error writing diff:", err)
		return
	}
	fmt.Println("Zipping hdiff files done.")

	fmt.Println("Removing hdiff temp files...")
	if err := RemoveFolderWithProgress(hdiffFolderPath, "🗑️ Removing hdiff temp files"); err != nil {
		fmt.Fprintln(os.Stderr, "error removing temp dir:", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("Removing hdiff temp files done.")

	fmt.Println("Done")
}
