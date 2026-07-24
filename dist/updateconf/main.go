package main

import (
	"fmt"
	"os"

	"code.waarp.fr/apps/gateway/gateway/dist/updateconf/updc"
)

func main() {
	if err := updc.Do(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(2) //nolint:mnd // too specific
	}

	fmt.Println("End of process updateconf")
}
