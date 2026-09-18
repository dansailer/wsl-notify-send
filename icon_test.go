package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageIconCopiesFileToTemp(t *testing.T) {
	source := filepath.Join(t.TempDir(), "mise.png")
	content := []byte("png test content")
	if err := os.WriteFile(source, content, 0600); err != nil {
		t.Fatal(err)
	}

	staged, cleanup, err := stageIcon(source)
	if err != nil {
		t.Fatal(err)
	}
	if staged == source {
		t.Fatal("icon was not staged")
	}
	if got, err := os.ReadFile(staged); err != nil {
		t.Fatal(err)
	} else if string(got) != string(content) {
		t.Fatalf("staged content = %q, want %q", got, content)
	}

	cleanup()
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("staged icon still exists: %v", err)
	}
}

func TestStageIconPreservesStockName(t *testing.T) {
	staged, cleanup, err := stageIcon("warning")
	if err != nil {
		t.Fatal(err)
	}
	if staged != "warning" {
		t.Fatalf("staged stock icon = %q", staged)
	}
	cleanup()
}
