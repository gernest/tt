package wasm

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gernest/tt/api"
	wasmerGo "github.com/wasmerio/wasmer-go/wasmer"
)

var ErrModuleNotFound = errors.New("module: 404")

type Wasm struct {
	engine  *wasmerGo.Engine
	store   *wasmerGo.Store
	mu      sync.RWMutex
	modules map[string]*wasmerGo.Module
}

func New() *Wasm {
	e := wasmerGo.NewEngine()
	s := wasmerGo.NewStore(e)
	return &Wasm{
		engine:  e,
		store:   s,
		modules: make(map[string]*wasmerGo.Module),
	}
}

func (w *Wasm) Compile(name string, wasmBytes []byte) error {
	m, err := wasmerGo.NewModule(w.store, wasmBytes)
	if err != nil {
		return err
	}
	w.modules[name] = m
	return nil
}

func (w *Wasm) NewInstance(mw *api.Middleware_Wasm) (*Instance, error) {
	opts := mw.GetConfig().Instance
	m, err := w.get(mw.GetName())
	if err != nil {
		return nil, err
	}
	name := mw.GetName()
	if opts.ProgramName != "" {
		name = opts.ProgramName
	}
	state := wasmerGo.NewWasiStateBuilder(name)
	for _, a := range opts.Arguments {
		state.Argument(a)
	}
	for _, e := range opts.Environments {
		state.Environment(e.Key, e.Value)
	}
	for _, d := range opts.PreopenDirectories {
		state.PreopenDirectory(d)
	}
	for _, d := range opts.MapDirectories {
		state.MapDirectory(d.Alias, d.Directory)
	}
	if opts.InheritStdin {
		state.InheritStdin()
	}
	if opts.CaptureStdout {
		state.CaptureStdout()
	}
	if opts.InheritStdout {
		state.InheritStdout()
	}
	if opts.CaptureStderr {
		state.CaptureStderr()
	}
	if opts.InheritStderr {
		state.InheritStderr()
	}
	env, err := state.Finalize()
	if err != nil {
		return nil, err
	}
	o, err := env.GenerateImportObject(w.store, m)
	if err != nil {
		return nil, err
	}
	inst, err := wasmerGo.NewInstance(m, o)
	if err != nil {
		return nil, err
	}
	return NewWasmerInstance(w, o, inst, mw), nil
}

func (w *Wasm) get(name string) (*wasmerGo.Module, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	m, ok := w.modules[name]
	if !ok {
		return nil, ErrModuleNotFound
	}
	return m, nil
}

func (w *Wasm) LoadFromDir(dir string) error {
	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".wasm", ".wat":
			return w.CompileFile(path)
		}
		return nil
	})
}

func (w *Wasm) CompileFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return w.Compile(name(path), b)
}

func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
