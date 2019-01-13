// gosince-cli is a command line client of api.gosince.com.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/yfzq/gosince"
)

const version = "0.1.0"

const gosinceURLFmt = "https://api.gosince.com/v1?name=%s"

func main() {
	showVersion := flag.Bool("version", false, "Print version of gosince")
	showHelp := flag.Bool("help", false, "Print this help message")
	cat := flag.String("cat", "", `Category of API to look for. Must be one of {"const", "var", "func", "method", "type"}`)
	flag.Parse()
	if *showVersion {
		fmt.Printf("gosince %s\n", version)
		return
	}
	if *showHelp || !flag.Parsed() {
		flag.Usage()
		return
	}
	if *cat != "" && *cat != "const" && *cat != "var" && *cat != "func" && *cat != "method" && *cat != "type" {
		fmt.Println("Invalid value of -cat")
		flag.Usage()
		return
	}

	names := flag.Args()
	if len(names) == 0 {
		fmt.Println("Must provide one name to query golang API")
		return
	}
	if len(names) > 1 {
		fmt.Printf("More than one name are provided. Only to query first name, %v\n", names[0])
	}

	getURL := fmt.Sprintf(gosinceURLFmt, names[0])
	if *cat != "" {
		getURL += fmt.Sprintf("&cat=%s", *cat)
	}
	resp, err := http.Get(getURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server error. %v\n", resp)
		return
	}

	records := []gosince.APIRecord{}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = json.Unmarshal(data, &records); err != nil {
		fmt.Println(err)
		return
	}
	if len(records) == 0 {
		fmt.Println("Cannot find matching API")
		return
	}
	outputRecords(os.Stdout, records)
}

// Print out gosince.APIRecords in table style to stdout
func outputRecords(w io.Writer, records []gosince.APIRecord) {
	data := [][]string{}
	for _, r := range records {
		data = append(data, []string{r.Version, r.PackageName, r.Name, r.Category, r.GolangURL})
	}
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Version", "Package", "Name", "Category", "URL"})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
}
