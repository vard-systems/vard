package main

import (
	"flag"
	"fmt"
	"os"

	"loginer/internal/apidoc"
)

func main() {
	check := flag.Bool("check", false, "report the files that are behind the code instead of writing them")
	flag.Parse()

	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	files, err := apidoc.Generate(root)
	if err != nil {
		return err
	}

	if check {
		stale, err := apidoc.Stale(root, files)
		if err != nil {
			return err
		}
		if len(stale) > 0 {
			for _, path := range stale {
				fmt.Fprintln(os.Stderr, "behind the code:", path)
			}
			return fmt.Errorf("the API reference is behind the code; run make docs")
		}
		fmt.Println("the API reference is current")
		return nil
	}

	if err := apidoc.Write(root, files); err != nil {
		return err
	}
	fmt.Printf("wrote %d files\n", len(files))
	return nil
}
