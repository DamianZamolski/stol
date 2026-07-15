SRC := $(wildcard *.go)

~/.local/bin/stol: $(SRC)
	go build -o ~/.local/bin/stol .
