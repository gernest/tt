
.PHONY: build
build:
	go build  -o bin/http 
tools/servers/echo/main.go:
	go build -o bin/echo ./tools/servers/echo 
tools/servers/http/main.go:
	go build  -o bin/http ./tools/servers/http
modules:
	tinygo build -o ./wasm/modules/s404/main.go.wasm -scheduler=none -target=wasi ./wasm/modules/s404/main.go
