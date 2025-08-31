package main

import (
	"fmt"
	"ldiff-converter/pb"
	"os"
	"path/filepath"
	"time"

	"github.com/amenzhinsky/go-memexec"
	"github.com/schollz/progressbar/v3"
)

func UnzipWith7za(src, dest string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", src)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	exe, err := memexec.New(sevenZip)
	if err != nil {
		return err
	}
	defer exe.Close()

	args := []string{"x", "-y", "-o" + destAbs, src}

	cmd := exe.Command(args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
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
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func MakeDiffMap(manifest *pb.ManifestProto, chunkNames []string) ([]*HDiffData, error) {
	var hdiffFiles []*HDiffData

	for _, asset := range manifest.Assets {
		assetName := asset.AssetName
		assetSize := asset.AssetSize

		if asset.AssetData != nil {
			for _, chunk := range asset.AssetData.Assets {
				matched := false
				for _, name := range chunkNames {
					if name == chunk.ChunkFileName {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}

				if chunk.OriginalFileSize != 0 || chunk.HdiffFileSize != assetSize {
					hdiffFiles = append(hdiffFiles, &HDiffData{
						SourceFileName: chunk.OriginalFilePath,
						TargetFileName: assetName,
						PatchFileName:  fmt.Sprintf("%s.hdiff", assetName),
					})
				}
			}
		}
	}

	return hdiffFiles, nil
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

		// Retry tối đa 10 lần khi xóa
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
