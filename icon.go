package main

import (
	"fmt"
	"os"
)

// stageIcon copies a real icon into the local OS temp directory. Windows toast
// rendering can reject an otherwise readable WSL UNC path, so the broker gets
// a local path instead. Non-file values remain available as stock icon names.
func stageIcon(icon string) (string, func(), error) {
	if icon == "" {
		return "", func() {}, nil
	}

	info, err := os.Stat(icon)
	if err != nil {
		return icon, func() {}, nil
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("icon is not a regular file: %s", icon)
	}

	data, err := os.ReadFile(icon)
	if err != nil {
		return "", nil, fmt.Errorf("could not read icon: %w", err)
	}

	temp, err := os.CreateTemp("", "notify-send-*.png")
	if err != nil {
		return "", nil, fmt.Errorf("could not stage icon: %w", err)
	}
	tempPath := temp.Name()
	remove := func() {
		_ = os.Remove(tempPath)
	}

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		remove()
		return "", nil, fmt.Errorf("could not stage icon: %w", err)
	}
	if err := temp.Close(); err != nil {
		remove()
		return "", nil, fmt.Errorf("could not stage icon: %w", err)
	}

	return tempPath, remove, nil
}
