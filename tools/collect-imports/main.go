package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/wasmerio/wasmer-go/wasmer"
)

type Import struct {
	Name      string
	Namespace string
	Kind      string
}

func main() {
	e := wasmer.NewEngine()
	s := wasmer.NewStore(e)
	flag.Parse()
	dir := flag.Arg(0)
	imports := make(map[Import]struct{})
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".wasm", ".wat":
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			m, err := wasmer.NewModule(s, b)
			if err != nil {
				return err
			}
			for _, v := range m.Imports() {
				imports[Import{
					Name:      v.Name(),
					Namespace: v.Module(),
					Kind:      v.Type().Kind().String(),
				}] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	for k := range imports {
		fmt.Printf("%s/%s => %s\n", k.Namespace, k.Name, k.Kind)
	}
}
