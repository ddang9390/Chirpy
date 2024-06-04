package main

import (
	"encoding/json"
	"os"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	db := DB{
		path: path,
		mux:  &sync.RWMutex{},
	}

	os.WriteFile(path, []byte(""), os.ModePerm)
	return &db, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	c := Chirp{}
	c.Body = body

	return c, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	var chirps = []Chirp{}
	res, _ := os.ReadFile(db.path)

	err := json.Unmarshal(res, &chirps)

	return chirps, err
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	if _, err := os.Stat("database.json"); err != nil {
		NewDB("database.json")
	}
	return nil
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
	type parameters struct {
		Chirps map[int]Chirp `json:"chirps"`
	}

	respBody := parameters{
		Chirps: dbStructure.Chirps,
	}

	_, err := json.Marshal(respBody)

	return err
}
