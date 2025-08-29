package main

import (
	"archive/zip"
	"fmt"
	"io"
	"ldiff-converter/pb"
	"os"
	"path/filepath"
	"strings"
)

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func Zip(src, dest string) error {
	zipFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if relPath == "." {
				return nil
			}
			_, err := zw.Create(relPath + "/")
			return err
		}

		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = relPath
		fh.Method = zip.Deflate

		writer, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	return err
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
