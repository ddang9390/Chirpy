package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
		authorId := r.URL.Query().Get("author_id")
		sortOrder := r.URL.Query().Get("sort")

		// Step 1: Call GetChirps to fetch all chirps from the database
		chirps, err := db.GetChirps()
		if err != nil {
			http.Error(w, "Could not retrieve chirps", http.StatusInternalServerError)
			return
		}

		if authorId != "" {
			var matchedChirps []Chirp
			for _, chirp := range chirps {
				if fmt.Sprint(chirp.Author_ID) == authorId {
					matchedChirps = append(matchedChirps, chirp)
				}
			}
			chirps = matchedChirps

		}

		if sortOrder == "asc" {
			sort.Slice(chirps, func(i, j int) bool {
				return chirps[i].ID < chirps[j].ID
			})
		} else {
			sort.Slice(chirps, func(i, j int) bool {
				return chirps[i].ID > chirps[j].ID
			})
		}

		// Step 2: Respond with the list of chirps in JSON format
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(chirps)

	}
}

func postHandler(db *DB, cfg *apiConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, _ := db.loadDB()
		authorID := 0

		claims, _ := jwtValidate(r, cfg.jwtSecret)

		for _, user := range users.Users {
			if fmt.Sprint(user.ID) == claims.Subject {
				authorID = int(user.ID)
			}
		}

		// Step 1: Read and validate the request body
		var reqBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil || reqBody["body"] == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Step 2: Call CreateChirp with the body content
		chirp, err := db.CreateChirp(reqBody["body"], authorID)
		if err != nil {
			http.Error(w, "Could not create chirp", http.StatusInternalServerError)
			return
		}

		// Step 3: Respond with the created chirp
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(chirp)
	}
}
