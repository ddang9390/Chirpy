package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	apiCfg := &apiConfig{}
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("."))
	wrappedFileServer := apiCfg.middlewareMetricsInc(fileServer)

	db, err := NewDB("database.json")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	mux.Handle("/app/*", http.StripPrefix("/app", wrappedFileServer))

	mux.HandleFunc("GET /api/healthz", apiCfg.readyHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.hitsHandler)
	mux.HandleFunc("/api/reset", apiCfg.resetHandler)

	mux.HandleFunc("/api/chirps", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getHandler(db).ServeHTTP(w, r)
		case http.MethodPost:
			postHandler(db).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

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
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	//html template
	const htmlTemplate = `
	<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited {{.Count}} times!</p>
	</body>
	</html>
	`
	//parse the template
	var tmpl = template.Must(template.New("metrics").Parse(htmlTemplate))

	//inject data into that template
	data := struct {
		Count int
	}{Count: cfg.fileserverHits}

	tmpl.Execute(w, data)

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

func (cfg *apiConfig) jsonHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Id           int    `json:"id"`
		Body         string `json:"body"`
		Error        string `json:"error"`
		Valid        bool   `json:"valid"`
		Cleaned_body string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	errorString := ""
	status := 200
	valid := true

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		errorString = "Something went wrong"
		status = 500
		valid = false

	}

	if len(params.Body) > 140 {
		errorString = "Chirp is too long"
		status = 400
		valid = false
	}
	cleanedBody := checkProfane(params.Body)
	respBody := parameters{
		Body:         params.Body,
		Error:        errorString,
		Valid:        valid,
		Cleaned_body: cleanedBody,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(status)
	w.Write(dat)

}

func checkProfane(body string) string {
	badWords := [3]string{"kerfuffle", "sharbert", "fornax"}
	bodyText := strings.Split(body, " ")

	result := []string{}
	for _, word := range bodyText {
		for _, badWord := range badWords {
			if strings.ToLower(word) == badWord {
				word = "****"
				break
			}
		}
		result = append(result, word)
	}

	return strings.Join(result, " ")
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
