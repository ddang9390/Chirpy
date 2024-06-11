package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
		user.Expires_in_seconds = 20

		response := map[string]interface{}{
			"id":                 user.ID,
			"email":              user.Email,
			"expires_in_seconds": user.Expires_in_seconds,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

func loginUser(db *DB, cfg *apiConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody User
		err := json.NewDecoder(r.Body).Decode(&reqBody)

		if err != nil || reqBody.Email == "" {
			fmt.Println(err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		users, err := db.loadDB()
		if err != nil {
			http.Error(w, "Issue getting users", 404)
			return
		}

		var user User
		userExists := false
		for _, u := range users.Users {
			if u.Email == reqBody.Email {
				user = u
				userExists = true
				break
			}
		}

		if !userExists {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		fmt.Println(user)
		fmt.Println(reqBody)

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(reqBody.Password))
		if err != nil {
			fmt.Printf("Input PW:%s, Actual PW:%s\n\n", reqBody.Password, user.Password)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		user.Expires_in_seconds = reqBody.Expires_in_seconds

		token := jwtCreation(user, cfg.jwtSecret)

		response := map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"token": token,
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)

	}
}

func updateUser(db *DB, cfg *apiConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody User
		err := json.NewDecoder(r.Body).Decode(&reqBody)

		if err != nil || reqBody.Email == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		claims, err := jwtValidate(r, cfg.jwtSecret)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(401)
			return
		}
		updatedUser := User{}
		if claims != nil {
			id, err := strconv.Atoi(claims.Subject)
			if err != nil {
				fmt.Println("Invalid id")
				w.WriteHeader(401)
				return
			}

			users, err := db.loadDB()
			if err != nil {
				fmt.Println("Issue loading users")
				w.WriteHeader(401)
				return
			}

			updatedUser = users.Users[int64(id)]
			updatedUser.Email = reqBody.Email

			encPW, err := bcrypt.GenerateFromPassword([]byte(reqBody.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Could not use password", http.StatusInternalServerError)
				w.WriteHeader(401)
				return
			}
			reqBody.Password = string(encPW)
			updatedUser.Password = reqBody.Password

			users.Users[int64(id)] = updatedUser
			db.writeDB(users)
		}

		response := map[string]interface{}{
			"id":    updatedUser.ID,
			"email": updatedUser.Email,
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)

	}
}
