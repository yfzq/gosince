package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yfzq/gosince"
)

var db *sql.DB

// Return nil iff category is one of {"const", "func", "method", "type", "var"}.
func isValidCategory(cat string) error {
	for _, s := range [...]string{"const", "func", "method", "type", "var"} {
		if cat == s {
			return nil
		}
	}
	return fmt.Errorf(`%s is not one of "const", "func", "method", "type", "var"`, cat)
}

// Return nil iff name is a valid golang identifier
func isValidName(name string) error {
	re := regexp.MustCompile(`^\w+$`)
	if re.FindString(name) == "" {
		return fmt.Errorf("%s should only contains [A-Za-z0-9_]", name)
	}
	return nil
}

// http handler for /v1
func queryNameHandler(w http.ResponseWriter, r *http.Request) {
	// validate query parameter "name"
	name := strings.TrimSpace(r.FormValue("name"))
	if err := isValidName(name); err != nil {
		log.Println("ERROR: ", err)
		w.WriteHeader(http.StatusBadRequest)
		if _, e := w.Write([]byte(err.Error())); e != nil {
			log.Println("ERROR: ", e)
		}
		return
	}
	// validate query parameter "cat"
	category := strings.TrimSpace(r.FormValue("cat"))
	if category != "" {
		if err := isValidCategory(category); err != nil {
			log.Println("ERROR: ", err)
			w.WriteHeader(http.StatusBadRequest)
			b := []byte(err.Error())
			if _, e := w.Write(b); e != nil {
				log.Println("ERROR: ", e)
			}
			return
		}
	}
	records, err := queryRecords(db, name, category)
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "max-age=3600")
	if b, err := json.Marshal(records); err != nil {
		log.Println("ERROR: ", err)
	} else {
		if _, e := w.Write(b); e != nil {
			log.Println("ERROR: ", e)
		}
	}
}

// Return a slice of APIRecords with the matching name and category.
func queryRecords(db *sql.DB, name string, category string) ([]gosince.APIRecord, error) {
	records := []gosince.APIRecord{}
	ctx, cancel := context.WithTimeout(context.Background(), gosince.DBTimeout)
	defer cancel()

	queryStrFmt := `SELECT name, category, version, package_name, description, golang_url FROM goapis WHERE %s 
	ORDER BY LENGTH(version) DESC, version DESC, package_name`
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	if category == "" {
		queryStr := fmt.Sprintf(queryStrFmt, `name LIKE ?`)
		if stmt, err = db.PrepareContext(ctx, queryStr); err != nil {
			return nil, err
		}
		if rows, err = stmt.QueryContext(ctx, name); err != nil {
			return nil, err
		}
	} else {
		queryStr := fmt.Sprintf(queryStrFmt, `name LIKE ? AND category LIKE ?`)
		if stmt, err = db.PrepareContext(ctx, queryStr); err != nil {
			return nil, err
		}
		if rows, err = stmt.QueryContext(ctx, name, category); err != nil {
			return nil, err
		}
	}
	defer stmt.Close()
	defer rows.Close()

	for rows.Next() {
		r := gosince.APIRecord{}
		if err := rows.Scan(&r.Name, &r.Category, &r.Version, &r.PackageName, &r.Description, &r.GolangURL); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func main() {
	dbFile := flag.String("file", "", "required. full file path of the sqlite3 database file")
	port := flag.Int("port", 80, "port the api service listens on")
	flag.Parse()

	if *dbFile == "" {
		flag.Usage()
		return
	}
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=ro", *dbFile)
	var err error
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1000)
	r := mux.NewRouter()
	r.HandleFunc("/v1", queryNameHandler).
		Methods("GET")
	http.Handle("/", r)
	listenAddres := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Fatal(http.ListenAndServe(listenAddres, r))
}
