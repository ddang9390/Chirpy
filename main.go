package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	debugCode()

	apiCfg := &apiConfig{}
	r := mux.NewRouter()

	//mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("."))
	wrappedFileServer := apiCfg.middlewareMetricsInc(fileServer)

	db, err := NewDB("database.json")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	r.Handle("/app/*", http.StripPrefix("/app", wrappedFileServer))

	r.HandleFunc("GET /api/healthz", apiCfg.readyHandler)
	r.HandleFunc("GET /admin/metrics", apiCfg.hitsHandler)
	r.HandleFunc("/api/reset", apiCfg.resetHandler)

	r.HandleFunc("/api/chirps", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getHandler(db).ServeHTTP(w, r)
		case http.MethodPost:
			postHandler(db).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.HandleFunc("/api/chirps/{chirpID}", getChirp(db)).Methods("GET")

	r.HandleFunc("/api/users", postUsers(db)).Methods("POST")

	http.Handle("/", r)

	http.ListenAndServe(":8080", r)

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

func getChirp(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chirps, err := db.loadDB()
		if err != nil {
			http.Error(w, "Issue getting chirps", 404)
			return
		}

		vars := mux.Vars(r)
		chirpIDStr := vars["chirpID"]
		chirpID, err := strconv.Atoi(chirpIDStr)
		if err != nil {
			http.Error(w, "Invalid chirp ID", 404)
			return
		}
		chirp, found := chirps.Chirps[chirpID]
		if found {
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(chirp)
		} else {
			w.WriteHeader(404)

		}
	}
}

func postUsers(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil || reqBody["email"] == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		user, err := db.CreateUser(reqBody["email"])
		if err != nil {
			http.Error(w, "Could not create user", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

func debugCode() {
	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	if *dbg {
		err := os.Remove("database.json")
		if err != nil {
			// Handle error if the file doesn't exist or couldn't be deleted
			fmt.Println("Error deleting database:", err)
		} else {
			fmt.Println("Database deleted successfully!")
		}
	}
}
