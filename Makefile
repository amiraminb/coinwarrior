APP := coinw
PKG := ./...
BIN_DIR := ~/.local/bin

.PHONY: build dev clean


build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP) .

dev:
	go mod tidy
	go mod vendor

clean:
	rm -rf $(BIN_DIR)/$(APP)
