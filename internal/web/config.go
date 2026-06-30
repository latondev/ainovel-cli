package web

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Config cấu hình server web Phase 0.
type Config struct {
	Addr       string
	DataRoot   string
	StaticDir  string
	NovelsGlob string // reserved for Phase 1
}

// ParseFlags đọc flag dòng lệnh.
func ParseFlags(args []string) (Config, error) {
	fs := flag.NewFlagSet("ainovel-web", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "địa chỉ lắng nghe HTTP")
	dataRoot := fs.String("data-root", ".", "thư mục gốc chứa output/novel hoặc data/novels/")
	staticDir := fs.String("static", "web/frontend/dist", "thư mục static SPA (sau npm run build)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	root, err := filepath.Abs(*dataRoot)
	if err != nil {
		return Config{}, fmt.Errorf("data-root: %w", err)
	}
	static, err := filepath.Abs(*staticDir)
	if err != nil {
		return Config{}, fmt.Errorf("static: %w", err)
	}
	return Config{
		Addr:      *addr,
		DataRoot:  root,
		StaticDir: static,
	}, nil
}

// DefaultConfig trả về cấu hình mặc định (dùng trong test).
func DefaultConfig() Config {
	wd, _ := os.Getwd()
	return Config{
		Addr:      "127.0.0.1:8080",
		DataRoot:  wd,
		StaticDir: filepath.Join(wd, "web", "frontend", "dist"),
	}
}