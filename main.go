package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "błąd:", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	in, err := parseArgs(argv)
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	game, err := resolveGame(cfg, in.query)
	if err != nil {
		return err
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}

	post := render(game, in)
	fmt.Println(post)
	return nil
}
