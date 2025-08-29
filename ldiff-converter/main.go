package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"ldiff-converter/pb"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter zip ldiff path: ")
	ldiff, _ := reader.ReadString('\n')
	ldiff = strings.TrimSpace(ldiff)
	if ldiff == "" {
		fmt.Fprintln(os.Stderr, "no ldiff file provided")
		os.Exit(1)
	}

	fmt.Print("Enter zip hdiff output: ")
	hdiff, _ := reader.ReadString('\n')
	hdiff = strings.TrimSpace(hdiff)
	if hdiff == "" {
		fmt.Fprintln(os.Stderr, "no hdiff output provided")
		os.Exit(1)
	}

	tmpFolderPath := filepath.Join(".", "temp")
	if err := os.MkdirAll(tmpFolderPath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating temp dir:", err)
		os.Exit(1)
	}

	if err := Unzip(ldiff, tmpFolderPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	ldiffPath := filepath.Join(tmpFolderPath, "ldiff")
	manifestPath := filepath.Join(tmpFolderPath, "manifest")
	hdiffFolderPath := filepath.Join(".", "hdiff")

	manifestProto, err := LoadManifestProto(manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading manifest proto:", err)
		os.Exit(1)
	}

	ldiffEntries, err := os.ReadDir(ldiffPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading ldiff dir:", err)
		os.Exit(1)
	}

	for _, ldiffEntry := range ldiffEntries {
		assetName := ldiffEntry.Name()
		var matchingAssets []struct {
			AssetName string
			AssetSize int64
			Asset     *pb.AssetManifest
		}

		for _, assetGroup := range manifestProto.Assets {
			if data := assetGroup.AssetData; data != nil {
				for _, asset := range data.Assets {
					if asset.ChunkFileName == assetName {
						matchingAssets = append(matchingAssets, struct {
							AssetName string
							AssetSize int64
							Asset     *pb.AssetManifest
						}{assetGroup.AssetName, assetGroup.AssetSize, asset})
					}
				}
			}
		}

		for _, ma := range matchingAssets {
			err := LDiffFile(ma.Asset, ma.AssetName, ma.AssetSize, ldiffPath, hdiffFolderPath)
			if err != nil {
				continue
			}
		}
	}

	diffMapNames := make([]string, len(ldiffEntries))
	for i, e := range ldiffEntries {
		diffMapNames[i] = e.Name()
	}

	diffMapList, err := MakeDiffMap(manifestProto, diffMapNames)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error making diff map:", err)
		os.Exit(1)
	}
	diffMapJson, err := json.Marshal(diffMapList)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error marshal diff map:", err)
		os.Exit(1)
	}

	diffMapJsonPath := filepath.Join(hdiffFolderPath, "hdifffiles.json")
	if err := os.WriteFile(diffMapJsonPath, diffMapJson, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing diff map:", err)
		os.Exit(1)
	}

	if err := os.RemoveAll(tmpFolderPath); err != nil {
		fmt.Fprintln(os.Stderr, "error removing temp dir:", err)
		os.Exit(1)
	}

	if err := Zip(hdiffFolderPath, hdiff); err != nil {
		fmt.Fprintln(os.Stderr, "error zip hdiff:", err)
		os.Exit(1)
	}

	if err := os.RemoveAll(hdiffFolderPath); err != nil {
		fmt.Fprintln(os.Stderr, "error removing temp dir:", err)
		os.Exit(1)
	}

	fmt.Println("done!")

}
