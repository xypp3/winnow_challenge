package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed content/* content/icons/* content/menus/*
var contentFS embed.FS

type ManifestItem struct {
	Unavailable bool           `json:"unavailable"`
	Items       []ContentEntry `json:"items"`
}

type ContentEntry struct {
	Unavailable bool   `json:"unavailable"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	ExpiresAt   string `json:"expiresAt"`
	ETag        string `json:"ETag"`
}

func main() {
	port := getEnv("PORT", "8080")
	baseURL := getEnv("BASE_URL", fmt.Sprintf("http://localhost:%s", port))
	etag := getEnv("MANIFEST_ETAG", `W/"demo-1"`)

	manifest := map[string]ManifestItem{
		"icons": {
			Items: []ContentEntry{
				{
					Name:      "icon-1.png",
					URI:       fmt.Sprintf("%s/content/icons/icon-1.png", baseURL),
					ExpiresAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
					ETag:      "icon-1",
				},
			},
		},
		"menus": {
			Items: []ContentEntry{
				{
					Name:      "menu-1.json",
					URI:       fmt.Sprintf("%s/content/menus/menu-1.json", baseURL),
					ExpiresAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
					ETag:      "menu-1",
				},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/manifest", func(w http.ResponseWriter, r *http.Request) {
		ifNone := r.Header.Get("If-None-Match")
		if ifNone != "" && ifNone == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", etag)
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			log.Printf("encode manifest: %v", err)
		}
	})

	contentDir, err := fs.Sub(contentFS, "content")
	if err != nil {
		log.Fatalf("failed to load embedded content: %v", err)
	}
	mux.Handle("/content/", http.StripPrefix("/content/", http.FileServer(http.FS(contentDir))))

	addr := ":" + port
	log.Printf("stub manifest server listening on %s (base=%s)", addr, baseURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
