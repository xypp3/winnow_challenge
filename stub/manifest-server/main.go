package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
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

func generateManifest(rng *rand.Rand, baseURL string) map[string]ManifestItem {
	now := time.Now().UTC()
	expiry := now.Add(24 * time.Hour).Format(time.RFC3339)

	// Randomly choose categories to include; categories themselves are randomized.
	categories := randomCategories(rng)
	manifest := make(map[string]ManifestItem, len(categories))

	for _, category := range categories {
		switch category {
		case "icons":
			choices := []ContentEntry{
				{
					Name:      "icon-1.png",
					URI:       fmt.Sprintf("%s/content/icons/icon-1.png", baseURL),
					ExpiresAt: expiry,
					ETag:      fmt.Sprintf("icon-%d", rng.Intn(1000)),
				},
			}
			manifest[category] = ManifestItem{Items: pickSome(rng, choices)}
		case "menus":
			choices := []ContentEntry{
				{
					Name:      "menu-1.json",
					URI:       fmt.Sprintf("%s/content/menus/menu-1.json", baseURL),
					ExpiresAt: expiry,
					ETag:      fmt.Sprintf("menu-%d", rng.Intn(1000)),
				},
			}
			manifest[category] = ManifestItem{Items: pickSome(rng, choices)}
		default:
			manifest[category] = ManifestItem{
				Items: []ContentEntry{
					{
						Name: fmt.Sprintf("%s-item.json", category),
						// NOTE: stable content URI as that is not being generated/tested
						URI:       fmt.Sprintf("%s/content/menus/menu-1.json", baseURL),
						ExpiresAt: expiry,
						ETag:      fmt.Sprintf("%s-%d", category, rng.Intn(1000)),
					},
				},
			}
		}
	}

	return manifest
}

func pickSome(rng *rand.Rand, all []ContentEntry) []ContentEntry {
	if len(all) == 0 {
		return nil
	}
	n := rng.Intn(len(all)) + 1 // at least one
	out := make([]ContentEntry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, all[rng.Intn(len(all))])
	}
	return out
}

func randomCategories(rng *rand.Rand) []string {
	base := []string{"icons", "menus"}
	randomCount := rng.Intn(3) + 1 // add 1-3 random categories
	for i := 0; i < randomCount; i++ {
		base = append(base, randomName(rng, 8))
	}

	perm := rng.Perm(len(base))
	n := rng.Intn(len(base)) + 1 // at least one category
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, base[perm[i]])
	}
	return out
}

func randomName(rng *rand.Rand, n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func main() {
	port := getEnv("PORT", "8080")
	baseURL := getEnv("BASE_URL", fmt.Sprintf("http://localhost:%s", port))

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initialETag := getEnv("MANIFEST_ETAG", fmt.Sprintf(`W/"demo-%d"`, time.Now().UnixNano()))

	var mu sync.Mutex
	var manifest = generateManifest(rng, baseURL)
	var etag = initialETag

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/manifest", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if rng.Intn(3) == 0 {
			manifest = generateManifest(rng, baseURL)
			etag = fmt.Sprintf(`W/"demo-%d"`, time.Now().UnixNano())
		}
		currentManifest := manifest
		currentETag := etag
		mu.Unlock()

		ifNone := r.Header.Get("If-None-Match")
		if ifNone != "" && ifNone == currentETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", currentETag)
		if err := json.NewEncoder(w).Encode(currentManifest); err != nil {
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
