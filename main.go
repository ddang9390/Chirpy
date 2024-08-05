package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type apiConfig struct {
	fileserverHits int
	jwtSecret      string
}

func main() {
	// by default, godotenv will look for a file named .env in the current directory
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")

	// clears database file whenever we run program to make testing faster
	debugCode()

	apiCfg := &apiConfig{jwtSecret: jwtSecret}
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
			postHandler(db, apiCfg).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.HandleFunc("/api/chirps/{chirpID}", getChirp(db)).Methods("GET")
	r.HandleFunc("/api/chirps/{chirpID}", deleteChirp(db, apiCfg)).Methods("DELETE")

	r.HandleFunc("/api/users", postUsers(db)).Methods("POST")
	r.HandleFunc("/api/users", updateUser(db, apiCfg)).Methods("PUT")

	r.HandleFunc("/api/login", loginUser(db, apiCfg)).Methods("POST")
	r.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			refreshUser(w, r, db, apiCfg)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.HandleFunc("/api/revoke", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			revokeUser(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	r.HandleFunc("/api/polka/webhooks", polkaHandler(db)).Methods("POST")

	http.Handle("/", r)

	http.ListenAndServe(":8080", r)

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
