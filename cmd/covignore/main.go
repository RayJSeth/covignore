package main

import (
	"os"

	"github.com/RayJSeth/covignore/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
