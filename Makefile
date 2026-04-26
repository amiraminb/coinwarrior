APP := coinw
PKG := ./...
BIN_DIR := ~/.local/bin

.PHONY: build dev clean


build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP) .

dep:
	go mod tidy
	go mod vendor

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)/$(APP)
