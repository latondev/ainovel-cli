package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverOutputNovel(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Skipf("repo root: %v", err)
	}
	progressPath := filepath.Join(root, "output", "novel", "meta", "progress.json")
	if _, err := os.Stat(progressPath); err != nil {
		t.Skip("output/novel không có trong workspace")
	}

	reader, err := NewReader(root)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	list, err := reader.ListNovels()
	if err != nil {
		t.Fatalf("ListNovels: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one novel")
	}
	found := false
	for _, s := range list {
		if s.Title == "Người Vẽ Hồn" {
			found = true
			if s.CompletedCount != 25 {
				t.Errorf("completed_count = %d, want 25", s.CompletedCount)
			}
			if s.Phase != "writing" {
				t.Errorf("phase = %q, want writing", s.Phase)
			}
		}
	}
	if !found {
		t.Errorf("novel list = %+v, want title Người Vẽ Hồn", list)
	}
}

func TestSlugify(t *testing.T) {
	got := slugify("Người Vẽ Hồn")
	if got != "nguoi-ve-hon" {
		t.Errorf("slugify = %q, want nguoi-ve-hon", got)
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd, nil
		}
		dir = parent
	}
}