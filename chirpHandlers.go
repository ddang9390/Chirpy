package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

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

func deleteChirp(db *DB, cfg *apiConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chirps, err := db.loadDB()
		if err != nil {
			http.Error(w, "Issue getting chirps", 404)
			return
		}

		claims, _ := jwtValidate(r, cfg.jwtSecret)

		authorID := claims.Subject

		vars := mux.Vars(r)
		chirpIDStr := vars["chirpID"]
		chirpID, err := strconv.Atoi(chirpIDStr)
		if err != nil {
			http.Error(w, "Invalid chirp ID", 404)
			return
		}
		chirp, found := chirps.Chirps[chirpID]
		if found {
			if fmt.Sprint(chirp.Author_ID) == authorID {
				delete(chirps.Chirps, chirp.ID)
				w.WriteHeader(204)
			} else {
				w.WriteHeader(403)
			}
		}
	}
}
