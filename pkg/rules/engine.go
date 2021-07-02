package rules

import (
	"context"
	"sort"
	"sync"

	"github.com/gernest/tt/api"
)

type Matcher interface {
	Match(ctx context.Context, meta *api.Context) bool
}

type Node struct {
	Value   interface{}
	Matcher Matcher
	Score   int
}

type Engine struct {
	mu    sync.Mutex
	nodes []*Node
}

func (e *Engine) build(ctx context.Context, rule *api.Rule) (Matcher, error) {
	return build(rule)
}

func (e *Engine) Build(ctx context.Context, rule *api.Rule, value interface{}, priority ...int) error {
	m, err := e.build(ctx, rule)
	if err != nil {
		return err
	}
	var p int
	if len(priority) > 0 {
		p = priority[0]
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodes = append(e.nodes, &Node{
		Value:   value,
		Matcher: m,
		Score:   Priority(rule, p),
	})
	sort.Slice(e.nodes, func(i, j int) bool {
		return e.nodes[i].Score < e.nodes[j].Score
	})
	return nil
}

func (e *Engine) Match(ctx context.Context, meta *api.Context) (value interface{}, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, n := range e.nodes {
		ok = n.Matcher.Match(ctx, meta)
		if ok {
			value = n.Value
			return
		}
	}
	return
}
