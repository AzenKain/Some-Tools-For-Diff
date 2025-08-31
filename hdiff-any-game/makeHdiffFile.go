package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/schollz/progressbar/v3"
)

type HdiffFile struct {
	RemoteName string `json:"remoteName"`
}

func MakeHdiffFile(oldPath string, newPath string, changedFiles []string) error {
	delFile, err := os.Create("hdiff/hdifffiles.txt")
	if err != nil {
		return err
	}
	defer delFile.Close()

	for _, f := range changedFiles {
		data, err := json.Marshal(HdiffFile{RemoteName: f})
		if err != nil {
			return err
		}
		fmt.Fprintln(delFile, string(data))
	}

	bar := progressbar.NewOptions(len(changedFiles),
		progressbar.OptionSetDescription("📦 Creating HDiff files"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetPredictTime(true),
	)

	workers := runtime.NumCPU() / 2
	if workers < 2 {
		workers = 2
	}
	jobs := make(chan string, len(changedFiles))
	var wg sync.WaitGroup

	for i := int64(0); i < int64(workers); i++ {
		wg.Go(func() {
			for f := range jobs {
				oldFile := filepath.Join(oldPath, f)
				newFile := filepath.Join(newPath, f)
				hdiffPath := filepath.Join("hdiff", f+".hdiff")
				if err := os.MkdirAll(filepath.Dir(hdiffPath), 0755); err != nil {
					fmt.Fprintf(os.Stderr, "failed to create dir: %v\n", err)
					continue
				}
				Hdiffz(oldFile, newFile, hdiffPath)
				bar.Add(1)
			}
		})
	}

	for _, f := range changedFiles {
		jobs <- f
	}
	close(jobs)

	wg.Wait()
	bar.Finish()
	return nil
}
