package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	UsedRepoJSON     string             `json:"usedRepoJson"`
	NixpkgsRepoJSON  string             `json:"nixpkgsRepoJson"`
	MetadataMode     string             `json:"metadataMode"`
	MetadataWritable bool               `json:"metadataWritable"`
	Configured       ConfiguredVersions `json:"configured"`
}

type ConfiguredVersions struct {
	Platforms     []string `json:"platforms"`
	BuildTools    []string `json:"buildTools"`
	PlatformTools string   `json:"platformTools"`
	Emulator      string   `json:"emulator"`
	NDK           string   `json:"ndk"`
	CmdlineTools  string   `json:"cmdlineTools"`
	CMake         []string `json:"cmake"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.UsedRepoJSON == "" {
		return Config{}, fmt.Errorf("config missing usedRepoJson")
	}
	if cfg.NixpkgsRepoJSON == "" {
		return Config{}, fmt.Errorf("config missing nixpkgsRepoJson")
	}
	if cfg.MetadataMode == "" {
		if cfg.MetadataWritable {
			cfg.MetadataMode = "project-local writable"
		} else {
			cfg.MetadataMode = "bundled/pinned read-only"
		}
	}

	return cfg, nil
}
