package main

import (
	"fmt"
	"os"

	"github.com/svaruag/mocli/internal/app"
	"github.com/svaruag/mocli/internal/migrate"
)

func main() {
	if err := migrate.Run(os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: startup migration skipped: %v\n", err)
	}
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr, os.LookupEnv))
}
