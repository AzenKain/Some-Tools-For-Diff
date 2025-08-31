package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"path/filepath"
	"runtime"
	"sync"

	"github.com/amenzhinsky/go-memexec"
	"github.com/schollz/progressbar/v3"
)

func collectFiles(root string) (map[string]string, error) {
	files := sync.Map{}
	var wg sync.WaitGroup
	dirs := make(chan string, 100)

	workers := runtime.NumCPU() * 2
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		go func() {
			for dir := range dirs {
				entries, err := os.ReadDir(dir)
				if err != nil {
					wg.Done()
					continue
				}
				for _, e := range entries {
					path := filepath.Join(dir, e.Name())
					if e.IsDir() {
						wg.Add(1)
						dirs <- path
					} else {
						rel, _ := filepath.Rel(root, path)
						files.Store(filepath.ToSlash(rel), path)
					}
				}
				wg.Done()
			}
		}()
	}
	wg.Add(1)
	dirs <- root

	go func() {
		wg.Wait()
		close(dirs)
	}()

	wg.Wait()
	out := make(map[string]string)
	files.Range(func(k, v any) bool {
		out[k.(string)] = v.(string)
		return true
	})

	return out, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func ZipWith7za(src, dest string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source folder does not exist: %s", src)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	files, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("source folder is empty: %s", src)
	}

	destAbs, err := filepath.Abs(filepath.Join(".", dest))
	if err != nil {
		return err
	}

	args := []string{"a", "-tzip", "-mx=1", "-mmt=on", destAbs}
	for _, f := range files {
		args = append(args, f.Name())
	}

	exe, err := memexec.New(sevenZip)
	if err != nil {
		return err
	}
	defer exe.Close()
	cmd := exe.Command(args...)
	cmd.Dir = src
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func RemoveFolderWithProgress(folder string, title string) error {
	var total int
	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total++
		}
		return nil
	})

	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(title),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetPredictTime(true),
	)

	_ = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		var rmErr error
		for i := 0; i < 10; i++ {
			rmErr = os.Remove(path)
			if rmErr == nil {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if rmErr != nil {
			fmt.Printf("⚠️ Could not remove %s: %v\n", path, rmErr)
		}

		bar.Add(1)
		return nil
	})

	if err := os.RemoveAll(folder); err != nil {
		return fmt.Errorf("failed to remove folder %s: %w", folder, err)
	}
	bar.Finish()
	return nil
}

func Hdiffz(oldPath, newPath, outDiff string) error {
	args := []string{"-s-64", "-SD", "-c-zstd-21-24", "-d", oldPath, newPath, outDiff}
	exe, err := memexec.New(hdiffz)
	if err != nil {
		return err
	}
	defer exe.Close()
	cmd := exe.Command(args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}
