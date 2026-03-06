.PHONY: build run

build:
	go build -o logparser ./cmd/logparser

run: build
	./logparser $(ARGS)
