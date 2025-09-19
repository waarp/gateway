package main

import (
	"crypto/tls"
	"log"
	"os"
	"strings"

	"code.waarp.fr/lib/r66"
)

func main() {
	conn, err := tls.Dial("tcp", "127.0.0.1:10066", &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		panic(err)
	}

	client, err := r66.NewClient(conn, log.New(os.Stdout, "", 0))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ses, err := client.NewSession()
	if err != nil {
		panic(err)
	}
	defer ses.Close()

	if _, err = ses.Authent("waarp", []byte("sesame"), &r66.Config{}); err != nil {
		panic(err)
	}

	buf := strings.NewReader("Hello World")

	if _, err = ses.Request(&r66.Request{
		ID:       456,
		Filepath: "c:/test.txt",
		FileSize: 11,
		Rule:     "default",
		IsRecv:   false,
		Block:    65535,
	}); err != nil {
		panic(err)
	}

	if _, err = ses.Send(buf, func() ([]byte, error) {
		return []byte{}, nil
	}); err != nil {
		panic(err)
	}
}
