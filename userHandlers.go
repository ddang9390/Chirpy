package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
		user.Is_chirpy_red = false

		response := map[string]interface{}{
			"id":                 user.ID,
			"email":              user.Email,
			"expires_in_seconds": user.Expires_in_seconds,
			"is_chirpy_red":      user.Is_chirpy_red,
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

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(reqBody.Password))
		if err != nil {
			fmt.Printf("Input PW:%s, Actual PW:%s\n\n", reqBody.Password, user.Password)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		expiresInSeconds := int64(86400 * 60) //seconds in 60 days
		user.Expires_in_seconds = expiresInSeconds
		user.Expires_in_seconds = reqBody.Expires_in_seconds

		token := jwtCreation(user, cfg.jwtSecret)
		refreshToken := generateRefreshToken()

		user.Token = refreshToken

		users.Users[user.ID] = user
		db.writeDB(users)

		response := map[string]interface{}{
			"id":            user.ID,
			"email":         user.Email,
			"token":         token,
			"refresh_token": refreshToken,
			"is_chirpy_red": user.Is_chirpy_red,
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
			"id":            updatedUser.ID,
			"email":         updatedUser.Email,
			"is_chirpy_red": updatedUser.Is_chirpy_red,
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)

	}
}

func refreshUser(w http.ResponseWriter, r *http.Request, db *DB, cfg *apiConfig) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("authorization header is required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	users, _ := db.loadDB()
	resp := ""

	token := ""
	for _, user := range users.Users {
		if user.Token == tokenString {
			resp = user.Token
			token = jwtCreation(user, cfg.jwtSecret)
		}
	}

	if resp != "" {
		response := map[string]interface{}{
			"token": token,
		}
		w.WriteHeader(200)
		fmt.Println(response)
		json.NewEncoder(w).Encode(response)

	} else {
		w.WriteHeader(401)
	}

	return nil

}

func revokeUser(w http.ResponseWriter, r *http.Request, db *DB) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("authorization header is required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	users, _ := db.loadDB()

	for _, user := range users.Users {
		if user.Token == tokenString {
			updatedUser := user
			updatedUser.Token = ""

			users.Users[user.ID] = updatedUser
			db.writeDB(users)
		}
	}

	w.WriteHeader(204)
	return nil

}

func polkaHandler(db *DB, cfg *apiConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(401)
			return
		}
		keyString := strings.TrimPrefix(authHeader, "ApiKey ")
		if keyString != cfg.apiKey {
			w.WriteHeader(401)
			return
		}

		var reqBody PolkaEvent
		err := json.NewDecoder(r.Body).Decode(&reqBody)

		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		event := reqBody.Event
		id := reqBody.Data.UserID
		users, _ := db.loadDB()
		user := users.Users[id]

		if event != "user.upgraded" {
			w.WriteHeader(204)
			return
		} else {
			user.Is_chirpy_red = true
			users.Users[int64(id)] = user
			db.writeDB(users)

			w.WriteHeader(204)
			return
		}
	}
}
