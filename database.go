package main

import (
	"encoding/json"
	"os"
	"sort"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	db := &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	if err := db.ensureDB(); err != nil {
		return nil, err
	}

	return db, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	// Load the database from the file
	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	// Find a unique ID for the new chirp
	newID := len(dbStructure.Chirps) + 1

	// Create the new chirp
	newChirp := Chirp{
		ID:   newID,
		Body: body,
	}

	// Add the chirp to the in-memory database structure
	dbStructure.Chirps[newID] = newChirp

	// Write the updated database to the file
	err = db.writeDB(dbStructure)
	if err != nil {
		return Chirp{}, err
	}

	return newChirp, nil
}

func (db *DB) CreateUser(body string) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	// Load the database from the file
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	newID := len(dbStructure.Users) + 1

	newUser := User{
		ID:    newID,
		Email: body,
	}

	dbStructure.Users[newID] = newUser

	err = db.writeDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return newUser, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	// Read the database file
	res, err := os.ReadFile(db.path)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into the DBStructure
	var dbStructure DBStructure
	err = json.Unmarshal(res, &dbStructure)
	if err != nil {
		return nil, err
	}

	// Gather chirps into a slice and sort them by ID
	var chirps []Chirp
	for _, chirp := range dbStructure.Chirps {
		chirps = append(chirps, chirp)
	}

	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].ID < chirps[j].ID
	})

	return chirps, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	_, err := os.Stat(db.path)
	if os.IsNotExist(err) {
		// If not, create a new database file with an empty chirps map
		emptyDB := DBStructure{
			Chirps: make(map[int]Chirp),
			Users:  make(map[int]User),
		}
		return db.writeDB(emptyDB)
	}
	return err
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	var chirps = DBStructure{}
	res, _ := os.ReadFile(db.path)

	err := json.Unmarshal(res, &chirps)

	return chirps, err
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	res, err := json.Marshal(dbStructure)
	if err != nil {
		return nil
	}

	err = os.WriteFile(db.path, res, os.ModePerm)

	return err
}
