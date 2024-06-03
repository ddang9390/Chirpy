package main

import (
	"fmt"
	"net/http"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("."))
	wrappedFileServer := apiCfg.middlewareMetricsInc(fileServer)

	mux.Handle("/app/*", http.StripPrefix("/app", wrappedFileServer))

	mux.HandleFunc("GET /healthz", apiCfg.readyHandler)
	mux.HandleFunc("GET /metrics", apiCfg.hitsHandler)
	mux.HandleFunc("/reset", apiCfg.resetHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()

}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		fmt.Printf("Incrementing hits: %v\n", cfg.fileserverHits) // Debug statement
		next.ServeHTTP(w, r)
	})

}

func (cfg *apiConfig) hitsHandler(w http.ResponseWriter, r *http.Request) {
	s := fmt.Sprintf("Hits: %v", cfg.fileserverHits)
	w.Write([]byte(s))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Resetting hits from %v to 0\n", cfg.fileserverHits) // Debug statement
	cfg.fileserverHits = 0
	w.Write([]byte("Hits counter reset to 0"))
}

func (cfg *apiConfig) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

	ok := []byte("OK")
	w.Write(ok)
}
