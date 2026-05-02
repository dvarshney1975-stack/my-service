package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	connectDB()
	migrate()

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/info", handleInfo)
	mux.HandleFunc("/notes", handleNotes)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func connectDB() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	var err error
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			log.Println("connected to postgres")
			return
		}
		log.Printf("waiting for postgres: %v", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("could not connect to postgres: %v", err)
}

func migrate() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			body TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`)
	if err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello from my-service v2 (now with postgres!)\nhostname: %s\ntime: %s\n",
		hostname(), time.Now().Format(time.RFC3339))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, "db down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"service":  "my-service",
		"version":  os.Getenv("APP_VERSION"),
		"hostname": hostname(),
		"now":      time.Now().UTC(),
	})
}

type Note struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func handleNotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		rows, err := db.QueryContext(r.Context(),
			"SELECT id, body, created_at FROM notes ORDER BY id DESC LIMIT 100")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		notes := []Note{}
		for rows.Next() {
			var n Note
			if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			notes = append(notes, n)
		}
		json.NewEncoder(w).Encode(notes)

	case http.MethodPost:
		var n Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		err := db.QueryRowContext(r.Context(),
			"INSERT INTO notes (body) VALUES ($1) RETURNING id, created_at",
			n.Body).Scan(&n.ID, &n.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(n)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}
