
build:
	go build  -o bin/tt 
	go build -o bin/echo ./tools/servers/echo 
	go build  -o bin/http ./tools/servers/http
modules:
	tinygo build -o ./wasm/modules/s404/main.go.wasm -scheduler=none -target=wasi ./wasm/modules/s404/main.go
