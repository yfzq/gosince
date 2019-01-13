package main

import (
	"errors"
	"testing"

	"github.com/yfzq/gosince"
)

func TestGetAPIRecord(t *testing.T) {
	type pair struct {
		record *gosince.APIRecord
		err    error
	}
	input := map[string]pair{}
	input["pkg log/syslog (darwin-386), const LOG_ALERT = 1"] = pair{
		&gosince.APIRecord{Name: "LOG_ALERT", Category: "const", PackageName: "log/syslog", Description: "const LOG_ALERT = 1"}, nil}

	input["pkg log/syslog (freebsd-arm), const LOG_AUTH Priority"] = pair{
		&gosince.APIRecord{Name: "LOG_AUTH", Category: "const", PackageName: "log/syslog", Description: "const LOG_AUTH Priority"}, nil}

	input["pkg crypto, const SHA512_224 Hash"] = pair{&gosince.APIRecord{
		Name: "SHA512_224", Category: "const", PackageName: "crypto", Description: "const SHA512_224 Hash"}, nil}

	input["pkg unicode, var Chakma *RangeTable"] = pair{&gosince.APIRecord{Name: "Chakma", Category: "var", PackageName: "unicode", Description: "var Chakma *RangeTable"}, nil}

	input["pkg database/sql/driver, type Queryer interface { Query }"] = pair{&gosince.APIRecord{
		Name: "Queryer", Category: "type", PackageName: "database/sql/driver", Description: "type Queryer interface { Query }"}, nil}

	input["pkg archive/tar, func FileInfoHeader(os.FileInfo, string) (*Header, error)"] = pair{&gosince.APIRecord{
		Name: "FileInfoHeader", Category: "func", PackageName: "archive/tar", Description: "func FileInfoHeader(os.FileInfo, string) (*Header, error)"}, nil}

	input["pkg archive/tar, method (*Header) FileInfo() os.FileInfo"] = pair{&gosince.APIRecord{
		Name: "FileInfo", Category: "method", PackageName: "archive/tar", Description: "method (*Header) FileInfo() os.FileInfo"}, nil}

	input["pkg1 archive/tar, method (*Header) FileInfo() os.FileInfo"] = pair{nil, errors.New("could not match package name")}

	input["pkg archive/tar, application (*Header) FileInfo() os.FileInfo"] = pair{nil, errors.New("could not match proper category")}

	for k, v := range input {
		r, e := getAPIRecord(k)
		if e == nil && *r != *v.record {
			t.Errorf("expected (%v %v), got (%v, %v)", v.record, v.err, *r, e)
		} else if e != nil && r != nil {
			t.Errorf("exepcted error and nil apiRecord. got (%v, %v)", r, e)
		}
	}
}

func TestGolangURL(t *testing.T) {
	input := map[string]string{
		"method (IsolationLevel) String() string": "https://golang.org/pkg/database/sql/#IsolationLevel.String",
		"method (*Builder) String() string":       "https://golang.org/pkg/database/sql/#Builder.String",
	}
	for k, v := range input {
		golangURL := constructGolangURL("method", "String", "database/sql", k)
		if golangURL != v {
			t.Errorf("golang.org URL for %s should be %s", golangURL, v)
		}
	}
}
