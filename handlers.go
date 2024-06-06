package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		fmt.Printf("Incrementing hits: %v\n", cfg.fileserverHits) // Debug statement
		next.ServeHTTP(w, r)
	})

}

func getHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Call GetChirps to fetch all chirps from the database
		chirps, err := db.GetChirps()
		if err != nil {
			http.Error(w, "Could not retrieve chirps", http.StatusInternalServerError)
			return
		}

		// Step 2: Respond with the list of chirps in JSON format
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(chirps)
	}
}

func postHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Read and validate the request body
		var reqBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil || reqBody["body"] == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Step 2: Call CreateChirp with the body content
		chirp, err := db.CreateChirp(reqBody["body"])
		if err != nil {
			http.Error(w, "Could not create chirp", http.StatusInternalServerError)
			return
		}

		// Step 3: Respond with the created chirp
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(chirp)
	}
}
