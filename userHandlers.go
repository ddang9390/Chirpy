package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func postUsers(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil || reqBody["email"] == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		encPW, err := bcrypt.GenerateFromPassword([]byte(reqBody["password"]), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Could not use password", http.StatusInternalServerError)
			return
		}
		reqBody["password"] = string(encPW)

		user, err := db.CreateUser(reqBody)
		if err != nil {
			http.Error(w, "Could not create user", http.StatusInternalServerError)
			return
		}

		user.Password = string(encPW)
		user.Expires_in_seconds = 2

		response := map[string]interface{}{
			"id":                 user.ID,
			"email":              user.Email,
			"expires_in_seconds": user.Expires_in_seconds,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

func loginUser(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&reqBody)

		if err != nil || reqBody["email"] == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		users, err := db.loadDB()
		if err != nil {
			http.Error(w, "Issue getting users", 404)
			return
		}

		user, ok := users.Users[reqBody["email"]]

		response := map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		}

		if ok {
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(reqBody["password"])) == nil {
				w.WriteHeader(200)
				json.NewEncoder(w).Encode(response)
				return
			} else {
				fmt.Printf("Input PW:%s, Actual PW:%s\n\n", reqBody["password"], user.Password)
				w.WriteHeader(401)
				return
			}
		}
	}
}
