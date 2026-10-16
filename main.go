package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	fs := flag.NewFlagSet("mediaservice", flag.ExitOnError)
	noElevate := fs.Bool("no-elevate", false, "跳过自动提权（仅供开发调试）")
	_ = fs.Parse(os.Args[1:])

	if err := run(*noElevate); err != nil {
		fmt.Fprintln(os.Stderr, "mediaservice:", err)
		os.Exit(1)
	}
}
