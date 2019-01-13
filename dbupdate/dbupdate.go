// dbupdate.go reads offcial golang API documents and persists each API record to
// a SQLite3 database.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	gosqlite3 "github.com/mattn/go-sqlite3"
	"github.com/yfzq/gosince"
)

const tableCreationDDL = `
CREATE TABLE IF NOT EXISTS goapis (
    name TEXT NOT NULL,
    category VARCHAR(20) NOT NULL CHECK(category in ('const', 'func', 'method', 'type', 'var')),
    version VARCHAR(5) NOT NULL,
    package_name TEXT NOT NULL,
	description TEXT NOT NULL,
	golang_url TEXT NOT NULL,
    CONSTRAINT row_unique UNIQUE (name, category, version, package_name, description)
);
CREATE INDEX IF NOT EXISTS goapis_name ON goapis(name);
`

const (
	githubAPIQuery = "https://api.github.com/search/code?q=repo:golang/go+path:api+extension:txt"
	urlBase        = "https://raw.githubusercontent.com/golang/go/master/api/"
)

// Regex expressions that match golang API
var (
	// Match package name.
	// pkg go/build, type Context struct, InstallSuffix string
	// pkg log/syslog (freebsd-arm), const LOG_AUTH Priority
	packageNameREs = []*regexp.Regexp{
		regexp.MustCompile(`^pkg ([^\(\)]+?),`),
		regexp.MustCompile(`^pkg (\S+?) \(.+?\),`),
	}
	// Match name
	categoriesRegexMap = map[string]*regexp.Regexp{
		// pkg crypto, const SHA512_224 Hash
		"const": regexp.MustCompile(`^const (\w+?) `),

		// pkg unicode, var Chakma *RangeTable
		"var": regexp.MustCompile(`^var (\w+?) `),

		// pkg database/sql/driver, type Queryer interface { Query }
		"type": regexp.MustCompile(`^type (\w+?) `),

		// pkg archive/tar, func FileInfoHeader(os.FileInfo, string) (*Header, error)
		"func": regexp.MustCompile(`^func (\w+?)\(`),

		// pkg archive/tar, method (*Header) FileInfo() os.FileInfo
		"method": regexp.MustCompile(`^method \(.+\) (\w+?)\(`),
	}

	// Match method receiver
	// pkg archive/tar, method (*Header) FileInfo() os.FileInfo
	receiverRE = regexp.MustCompile(`^method \(\*?(\w+?)\) `)
)

// Returns a slice of URL strings for official golang API.
// For example, https://raw.githubusercontent.com/golang/go/master/api/go1.1.txt
func getAPIURLs() ([]string, error) {
	apiURLs := []string{}
	resp, err := http.Get(githubAPIQuery)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respValue struct {
		Items []struct {
			Name string
			Path string
		}
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBytes, &respValue); err != nil {
		return nil, err
	}
	for _, r := range respValue.Items {
		if r.Path == "api/"+r.Name && strings.HasPrefix(r.Name, "go") {
			apiURLs = append(apiURLs, urlBase+r.Name)
		}
	}
	return apiURLs, nil
}

// Download golang API description file, e.g. go1.txt and write to recordChan
func produceAPIRecord(apiURL string, recordChan chan<- *gosince.APIRecord) {
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	apiVersion := filepath.Base(apiURL)
	apiVersion = strings.TrimPrefix(apiVersion, "go")
	apiVersion = strings.TrimSuffix(apiVersion, ".txt")
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		// ingore empty lines and comment lines.
		if len(s) == 0 || strings.HasPrefix(s, "#") {
			continue
		}
		if r, err := getAPIRecord(s); err != nil {
			log.Println("ERROR: ", err)
		} else {
			r.Version = apiVersion
			r.GolangURL = constructGolangURL(r.Category, r.Name, r.PackageName, r.Description)
			recordChan <- r
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("ERROR: ", err)
	}
}

// Return an URL to the official golang document.
func constructGolangURL(category, name, pkgName, description string) string {
	switch category {
	case "method":
		matches := receiverRE.FindStringSubmatch(description)
		if len(matches) == 2 {
			return fmt.Sprintf("https://golang.org/pkg/%s/#%s.%s", pkgName, matches[1], name)
		}
		return fmt.Sprintf("https://golang.org/pkg/%s/#%s", pkgName, name)
	default:
		return fmt.Sprintf("https://golang.org/pkg/%s/#%s", pkgName, name)
	}
}

// Insert gosince.APIRecord to database
func insertRecordToDB(db *sql.DB, r *gosince.APIRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), gosince.DBTimeout)
	defer cancel()
	result, err := db.ExecContext(ctx,
		`INSERT INTO goapis(name, category, version, package_name, description, golang_url)
		values(?, ?, ?, ?, ?, ?)`,
		r.Name, r.Category, r.Version, r.PackageName, r.Description, r.GolangURL)

	// Ignore duplicate API records. For example.
	// pkg log/syslog (darwin-386), const LOG_ALERT = 1
	// pkg log/syslog (openbsd-amd64-cgo), const LOG_ALERT = 1
	if err != nil {
		if e, ok := err.(gosqlite3.Error); ok && e.Code == 19 {
			return nil
		}
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return err
	}
	return nil
}

// Create and return an APIRecord from string s.
func getAPIRecord(s string) (*gosince.APIRecord, error) {
	record := gosince.APIRecord{}
	var matches []string
	for _, re := range packageNameREs {
		matches = re.FindStringSubmatch(s)
		if len(matches) == 2 {
			record.PackageName = matches[1]
			break
		}
	}
	if len(record.PackageName) == 0 {
		return nil, fmt.Errorf("could not match package name %s", s)
	}
	s = strings.TrimPrefix(s, matches[0])
	s = strings.TrimSpace(s)
	record.Description = s
	for category, re := range categoriesRegexMap {
		matches = re.FindStringSubmatch(s)
		if len(matches) == 2 {
			record.Category = category
			record.Name = matches[1]
			return &record, nil
		}
	}
	return nil, fmt.Errorf("could not match proper category %s", s)
}

func main() {
	dbfile := flag.String("file", "", "path of database file")
	showHelp := flag.Bool("help", false, "Print this help message")
	flag.Parse()
	if *showHelp || *dbfile == "" || !flag.Parsed() {
		flag.Usage()
		return
	}

	db, err := sql.Open("sqlite3", "file:"+*dbfile)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), gosince.DBTimeout)
	defer cancel()
	if _, err := db.ExecContext(ctx, tableCreationDDL); err != nil {
		log.Fatal(err)
	}

	// Parse golang API docs and insert every API record to sqlite
	apiURLs, err := getAPIURLs()
	if err != nil {
		log.Fatal(err)
	}

	var wgProducer sync.WaitGroup
	recordChan := make(chan *gosince.APIRecord, 10000)
	for i := range apiURLs {
		wgProducer.Add(1)
		apiURL := apiURLs[i]
		go func() {
			defer wgProducer.Done()
			produceAPIRecord(apiURL, recordChan)
		}()
	}
	var wgConsumer sync.WaitGroup
	wgConsumer.Add(1)
	go func() {
		defer wgConsumer.Done()
		for r := range recordChan {
			if err := insertRecordToDB(db, r); err != nil {
				log.Println("ERROR: ", err)
			}
		}
	}()
	wgProducer.Wait()
	close(recordChan)
	wgConsumer.Wait()
}
