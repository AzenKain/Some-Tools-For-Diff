package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"ldiff-converter/pb"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

//go:embed bin/7za.exe
var sevenZip []byte

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter ldiff path: ")
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
	if !strings.HasSuffix(strings.ToLower(hdiff), ".zip") {
		hdiff += ".zip"
	}

	tmpFolderPath := filepath.Join(".", "temp")
	if err := os.MkdirAll(tmpFolderPath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating temp dir:", err)
		os.Exit(1)
	}

	fmt.Println("Unzipping ldiff...")
	if err := UnzipWith7za(ldiff, tmpFolderPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("Unzipping ldiff done.")

	ldiffPath := filepath.Join(tmpFolderPath, "ldiff")
	manifestPath := filepath.Join(tmpFolderPath, "manifest")
	hdiffFolderPath := filepath.Join(".", "hdiff")

	fmt.Println("Loading manifest proto...")
	manifestProto, err := LoadManifestProto(manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading manifest proto:", err)
		os.Exit(1)
	}
	fmt.Println("Loading manifest proto done.")

	ldiffEntries, err := os.ReadDir(ldiffPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading ldiff dir:", err)
		os.Exit(1)
	}

	bar := progressbar.NewOptions(len(ldiffEntries),
		progressbar.OptionSetDescription("📦 Converting ldiff files"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetPredictTime(true),
	)
	fmt.Println("Converting ldiff files...")
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
		bar.Add(1)
		for _, ma := range matchingAssets {
			err := LDiffFile(ma.Asset, ma.AssetName, ma.AssetSize, ldiffPath, hdiffFolderPath)
			if err != nil {
				continue
			}
		}
	}
	bar.Finish()
	fmt.Println()
	fmt.Println("Converting ldiff files done.")
	diffMapNames := make([]string, len(ldiffEntries))
	for i, e := range ldiffEntries {
		diffMapNames[i] = e.Name()
	}

	fmt.Println("Making diff map...")
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
	fmt.Println("Making diff map done.")

	fmt.Println("Removing temp ldiff files...")
	if err := RemoveFolderWithProgress(tmpFolderPath, "🗑️ Removing temp ldiff files"); err != nil {
		fmt.Fprintln(os.Stderr, "error removing temp dir:", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("Removing temp ldiff files done.")

	fmt.Println("Zipping hdiff files...")
	if err := ZipWith7za(hdiffFolderPath, hdiff); err != nil {
		fmt.Fprintln(os.Stderr, "error zip hdiff:", err)
		os.Exit(1)
	}

	if _, err := os.Stat(hdiff); os.IsNotExist(err) {
		fmt.Println("File not found, retrying...")
		if err := ZipWith7za(hdiffFolderPath, hdiff); err != nil {
			fmt.Println("Retry failed:", err)
			os.Exit(1)
		}
	}
	fmt.Println("Zipping hdiff files done.")

	fmt.Println("Removing hdiff temp files...")
	if err := RemoveFolderWithProgress(hdiffFolderPath, "🗑️ Removing hdiff temp files"); err != nil {
		fmt.Fprintln(os.Stderr, "error removing temp dir:", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("Removing hdiff temp files done.")

	fmt.Println("Done!")

}
