
build:
	go build  -o bin/tt 
	go build -o bin/echo ./tools/servers/echo 
	go build  -o bin/http ./tools/servers/http
	go build  -o bin/rando ./tools/servers/rando
	go build  -o bin/cert ./tools/cert
	go build  -o bin/speed_client ./tools/clients/speed


certs:
	./bin/cert -host=echo.test -dir=./bin/certs/echo.test
	./bin/cert -host=rando.test -dir=./bin/certs/rando.test

modules:
	tinygo build -o ./wasm/modules/s404/main.go.wasm -scheduler=none -target=wasi ./wasm/modules/s404/main.go
