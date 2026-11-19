// Command ledger is the payments-ledger service entry point.
package main

import (
	"fmt"
	"io"
	"os"
)

func run(w io.Writer) error {
	_, err := fmt.Fprintln(w, "payments-ledger")
	return err
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
