package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

func CopyNewFiles(newPath string, result *DiffResult) error {
	delFile, err := os.Create("hdiff/deletefiles.txt")
	if err != nil {
		return err
	}
	defer delFile.Close()
	for _, f := range result.OnlyInOld {
		fmt.Fprintln(delFile, f)
	}

	bar := progressbar.NewOptions(len(result.OnlyInNew),
		progressbar.OptionSetDescription("📂 Copying new files"),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
	)

	for _, rel := range result.OnlyInNew {
		src := filepath.Join(newPath, rel)
		dst := filepath.Join("hdiff", rel)
		os.MkdirAll(filepath.Dir(dst), 0755)

		if err := copyFile(src, dst); err != nil {
			fmt.Println("copy error:", err)
		}
		bar.Add(1)
	}
	bar.Finish()

	return nil
}
