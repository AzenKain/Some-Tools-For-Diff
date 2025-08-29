package main

type HDiffData struct {
	SourceFileName string `json:"source_file_name"`
	TargetFileName string `json:"target_file_name"`
	PatchFileName  string `json:"patch_file_name"`
}