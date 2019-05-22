package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jessevdk/go-flags"
)

var parser = flags.NewNamedParser("waarp-gateway", flags.Default)

func main() {

	_, err := parser.Parse()

	if err != nil {
		if !flags.WroteHelp(err) {
			fmt.Fprintln(os.Stderr, "")
			parser.WriteHelp(os.Stderr)
			fmt.Fprintln(os.Stderr, "")
		}
		os.Exit(1)
	}
}

// readJSON reads the JSON body of the given http.Response, indents it in
// human readable format, and returns it as a string. If the body cannot be read
// or if it does not contain a valid JSON, returns an error instead.
func readJSON(res *http.Response) (string, error) {
	l := res.ContentLength
	var body = make([]byte, l)
	defer res.Body.Close()
	_, err := res.Body.Read(body)
	if err != nil && err != io.EOF {
		return "", err
	}
	var out bytes.Buffer
	err = json.Indent(&out, body, "", "  ")
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
