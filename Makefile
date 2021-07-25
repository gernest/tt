
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

WASM:=tools/examples
MODULE:=tools/modules

modules:
	tinygo build -o $(MODULE)/dispatch_call_on_tick.wasm -scheduler=none -target=wasi $(WASM)/dispatch_call_on_tick/main.go
	tinygo build -o $(MODULE)/foreign_call_on_tick.wasm -scheduler=none -target=wasi $(WASM)/foreign_call_on_tick/main.go
	tinygo build -o $(MODULE)/helloworld.wasm -scheduler=none -target=wasi $(WASM)/helloworld/main.go
	tinygo build -o $(MODULE)/http_auth_random.wasm -scheduler=none -target=wasi $(WASM)/http_auth_random/main.go
	tinygo build -o $(MODULE)/http_body.wasm -scheduler=none -target=wasi $(WASM)/http_body/main.go
	tinygo build -o $(MODULE)/http_headers.wasm -scheduler=none -target=wasi $(WASM)/http_headers/main.go
	tinygo build -o $(MODULE)/http_routing.wasm -scheduler=none -target=wasi $(WASM)/http_routing/main.go
	tinygo build -o $(MODULE)/metrics.wasm -scheduler=none -target=wasi $(WASM)/metrics/main.go
	tinygo build -o $(MODULE)/network.wasm -scheduler=none -target=wasi $(WASM)/network/main.go
	tinygo build -o $(MODULE)/shared_data.wasm -scheduler=none -target=wasi $(WASM)/shared_data/main.go
	tinygo build -o $(MODULE)/shared_queue_sender.wasm -scheduler=none -target=wasi $(WASM)/shared_queue/sender/main.go
	tinygo build -o $(MODULE)/shared_queue_receiver.wasm -scheduler=none -target=wasi $(WASM)/shared_queue/receiver/main.go
	tinygo build -o $(MODULE)/vm_plugin_configuration.wasm -scheduler=none -target=wasi $(WASM)/vm_plugin_configuration/main.go

clean:
	rm -r ./.tt/

dev: clean
	go build
	./tt -c tools/config.json