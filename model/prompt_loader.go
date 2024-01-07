package model

import (
	"fmt"
	"os"
	"path/filepath"
)

func LoadPromptFromFile(filename string) (string, error) {
	fullPath := filepath.Join("prompts", filename+".txt")
	bytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", fullPath, err)
	}
	return string(bytes), nil
}

