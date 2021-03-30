package main

//go:generate protoc -I api/ --go_out=plugins=grpc:./api api/tcp.proto
func main() {

}
