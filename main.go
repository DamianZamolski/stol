package main

import (
	"fmt"
	"os"
)

func main() {
	if error := run(os.Args[1:]); error != nil {
		fmt.Fprintln(os.Stderr, "błąd:", error)
		os.Exit(1)
	}
}

func run(argv []string) Error {
	in, error := parseArgs(argv)
	if error != nil {
		return error
	}

	config, error := loadConfig()
	if error != nil {
		return error
	}

	game, error := resolveGame(config, in.query)
	if error != nil {
		return error
	}
	if error := saveConfig(config); error != nil {
		return error
	}

	post := render(game, in)
	fmt.Println(post)
	return nil
}
