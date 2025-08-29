package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/schollz/progressbar/v3"
)

type DiffResult struct {
	OnlyInOld []string
	OnlyInNew []string
	Changed   []string
}

func safePartialMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := fi.Size()
	modTime := fi.ModTime().UnixNano()

	h := md5.New()

	binary.Write(h, binary.LittleEndian, size)
	binary.Write(h, binary.LittleEndian, modTime)
	binary.Write(h, binary.LittleEndian, fi.Mode())

	buf := make([]byte, 4096)

	if size <= 16*1024 {
		f.Seek(0, io.SeekStart)
		io.Copy(h, f)
		return hex.EncodeToString(h.Sum(nil)), nil
	}

	n, _ := f.Read(buf)
	h.Write(buf[:n])

	if size > 8192 {
		mid := size / 2
		f.Seek(mid-2048, io.SeekStart)
		n, _ = f.Read(buf)
		h.Write(buf[:n])
	}

	if size > 4096 {
		f.Seek(-4096, io.SeekEnd)
		n, _ = f.Read(buf)
		h.Write(buf[:n])
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func DiffFolders(oldPath, newPath string) (*DiffResult, error) {
	oldFiles, err := collectFiles(oldPath)
	if err != nil {
		return nil, err
	}
	newFiles, err := collectFiles(newPath)
	if err != nil {
		return nil, err
	}

	result := &DiffResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	jobs := make(chan [3]string, 100)

	total := 0
	for rel := range oldFiles {
		if _, ok := newFiles[rel]; ok {
			total++
		}
	}

	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription("🔍 Comparing files"),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
	)

	workers := runtime.NumCPU() * 2

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				rel, oldFile, newFile := job[0], job[1], job[2]
				oldHash, _ := safePartialMD5(oldFile)
				newHash, _ := safePartialMD5(newFile)
				if oldHash != newHash {
					mu.Lock()
					result.Changed = append(result.Changed, rel)
					mu.Unlock()
				}
				bar.Add(1)
			}
		}()
	}

	for rel, oldFile := range oldFiles {
		if newFile, ok := newFiles[rel]; ok {
			jobs <- [3]string{rel, oldFile, newFile}
		} else {
			result.OnlyInOld = append(result.OnlyInOld, rel)
		}
	}

	for rel := range newFiles {
		if _, ok := oldFiles[rel]; !ok {
			result.OnlyInNew = append(result.OnlyInNew, rel)
		}
	}

	close(jobs)
	wg.Wait()
	bar.Finish()

	return result, nil
}
