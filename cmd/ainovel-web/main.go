package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/voocel/ainovel-cli/internal/web"
)

func main() {
	cfg, err := web.ParseFlags(os.Args[1:])
	if err != nil {
		if err == os.ErrExist {
			return
		}
		fmt.Fprintf(os.Stderr, "flags: %v\n", err)
		os.Exit(1)
	}

	reader, err := web.NewReader(cfg.DataRoot)
	if err != nil {
		log.Fatalf("catalog: %v", err)
	}

	srv := web.NewServer(reader, cfg.StaticDir)
	log.Printf("ainovel-web listening on http://%s (data-root=%s)", cfg.Addr, cfg.DataRoot)
	if err := http.ListenAndServe(cfg.Addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}