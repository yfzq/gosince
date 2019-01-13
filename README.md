# gosince
Gosince finds the Golang version that an API is first introduced. There are two ways to use gosince.
1. Visit https://gosince.com 
2. Command line tool 
```bash
go get github.com/yfzq/gosince/gosince # install gosince-cli
gosince Open # returns API records whose name is "Open".
gosince -cat method Open # returns API records whose name is "Open" and the API is a method.
```

This repository contains three parts.
1. `dbupdate` downloads official Golang API documents, parse the document, and store records to a Sqlite3 database.
2. `webserver` is the Go web service for `api.gosince.com/v1`. It supports queries like `/v1?name=Open` and `/v1?name=Open&cat=method`.
3. `gosince` is the Go CLI client of `api.gosince.com`
