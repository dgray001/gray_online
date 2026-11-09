package goldset

import (
	"encoding/json"
	"net/http"
	"path/filepath"
)

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, err error) {
	respond(w, status, map[string]string{"error": err.Error()})
}

func Handler(store *Store, web string, images string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) { respond(w, 200, map[string]any{"items": store.Queue()}) })
	mux.HandleFunc("GET /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		item, exists := store.Get(r.PathValue("id"))
		if !exists {
			http.NotFound(w, r)
			return
		}
		respond(w, 200, item)
	})
	mux.HandleFunc("POST /api/items/{id}/label", store.labelHandler)
	mux.HandleFunc("POST /api/items/generate", store.generateHandler)
	mux.HandleFunc("GET /api/export", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename=fiddlesticks-goldset-v2.json")
		respond(w, 200, store.Export())
	})
	mux.Handle("GET /images/", http.StripPrefix("/images/", http.FileServer(http.Dir(images))))
	mux.HandleFunc("GET /goldset.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, filepath.Join(web, "goldset.js"))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, filepath.Join(web, "index.html"))
	})
	return mux
}
