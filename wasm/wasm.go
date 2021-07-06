package wasm

import (
	"errors"

	wasmerGo "github.com/wasmerio/wasmer-go/wasmer"
)

var ErrModuleNotFound = errors.New("module: 404")

type Wasm struct {
	engine  *wasmerGo.Engine
	store   *wasmerGo.Store
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

type InstanceOptions struct {
	ProgramName        string
	Arguments          []string
	Environments       []Env
	PreopenDirectories []string
	MapDirectories     []DirectoryMap
	InheritStdin       bool
	CaptureStdout      bool
	InheritStdout      bool
	CaptureStderr      bool
	InheritStderr      bool
}

type Env struct {
	Key, Value string
}

type DirectoryMap struct {
	Alias     string
	Directory string
}

func (w *Wasm) Instance(name string, opts InstanceOptions) (*wasmerGo.Instance, error) {
	m, err := w.get(name)
	if err != nil {
		return nil, err
	}
	state := wasmerGo.NewWasiStateBuilder(opts.ProgramName)
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
	return wasmerGo.NewInstance(m, o)
}

func (w *Wasm) get(name string) (*wasmerGo.Module, error) {
	return nil, ErrModuleNotFound
}
