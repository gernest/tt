package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/wasmerio/wasmer-go/wasmer"
)

type Import struct {
	Name      string
	Namespace string
	Kind      string
}

type Export struct {
	Name string
	Kind string
}

func main() {
	e := wasmer.NewEngine()
	s := wasmer.NewStore(e)
	flag.Parse()
	dir := flag.Arg(0)
	imports := make(map[Import]struct{})
	exports := make(map[Export]struct{})
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
			for _, v := range m.Exports() {
				exports[Export{
					Name: v.Name(),
					Kind: v.Type().Kind().String(),
				}] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	var m []Import
	for k := range imports {
		m = append(m, k)
	}
	sort.Slice(m, func(i, j int) bool {
		return m[i].Namespace < m[j].Namespace &&
			m[i].Name < m[j].Name
	})
	for _, k := range m {
		fmt.Printf("import %s/%s => %s\n", k.Namespace, k.Name, k.Kind)
	}
	fmt.Println()
	fmt.Println()
	var ex []Export
	for k := range exports {
		ex = append(ex, k)
	}
	sort.Slice(ex, func(i, j int) bool {
		return ex[i].Name < ex[j].Name
	})
	for _, k := range ex {
		fmt.Printf("export %s => %s\n", k.Name, k.Kind)
	}
}
