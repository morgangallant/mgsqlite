package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func port() string {
	port, ok := os.LookupEnv("PORT")
	if ok {
		return port
	}
	return "8080"
}

const dbname = "mgsqlite.db"

func dbpath() string {
	path, ok := os.LookupEnv("DBPATH")
	if ok {
		return filepath.Join(path, dbname)
	}
	return dbname
}

func migrate(conn *sqlite.Conn) (err error) {
	defer sqlitex.Save(conn)(&err)
	err = sqlitex.Exec(conn, `
		CREATE TABLE IF NOT EXISTS requests (
			ts DATETIME NOT NULL,
			host TEXT NOT NULL,
			method TEXT NOT NULL,
			user_agent TEXT NOT NULL
		);`, nil)
	return
}

var dbpool *sqlitex.Pool

func run() error {
	var err error
	dbpool, err = sqlitex.Open(dbpath(), 0, 10)
	if err != nil {
		return err
	}
	mctx := context.TODO()
	conn := dbpool.Get(mctx)
	if conn == nil {
		return errors.New("nil connection")
	}
	if err := migrate(conn); err != nil {
		return err
	}
	dbpool.Put(conn)
	log.Printf("Migrated successfully.")
	http.HandleFunc("/", handler)
	return http.ListenAndServe(":"+port(), nil)
}

func logRequest(conn *sqlite.Conn, r *http.Request) (err error) {
	defer sqlitex.Save(conn)(&err)
	err = sqlitex.Exec(conn, `INSERT INTO requests (
			ts, host, method, user_agent
		) VALUES (
			?, ?, ?, ?
		);`, nil, time.Now().Format(time.RFC3339), r.Host+r.RequestURI, r.Method, r.UserAgent())
	return
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	conn := dbpool.Get(r.Context())
	if conn == nil {
		return
	}
	defer dbpool.Put(conn)
	if err := logRequest(conn, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stmt := conn.Prep(`SELECT COUNT(*) AS count FROM requests;`)
	var count int64
	for {
		if next, err := stmt.Step(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !next {
			break
		}
		count = stmt.GetInt64("count")
	}
	d := time.Since(start)
	fmt.Fprintf(w, "%d Total Requests, Handler Took %d us.", count, d.Microseconds())
}
