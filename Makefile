.PHONY: all build build-linux run test clean

APP_NAME = oxidized-editor
LINUX_BINARY = $(APP_NAME)-linux-amd64

all: test build-linux

# Збірка під поточну ОС
build:
	go build -ldflags="-s -w" -o $(APP_NAME) .

# Крос-компіляція для Linux 64-bit (amd64)
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(LINUX_BINARY) .

# Запуск юніт-тестів
test:
	go test -v ./...

# Запуск локально
run:
	go run . -config config.yaml

# Очищення скомпільованих файлів
clean:
	rm -f $(APP_NAME) $(APP_NAME).exe $(LINUX_BINARY)
