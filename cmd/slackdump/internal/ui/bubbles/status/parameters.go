package status

import (
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"
)

type Parameter struct {
	Name  string
	Value any
}

type Parameters struct {
	mu     sync.RWMutex
	params []Parameter
	idx    map[string]int
}

func NewParameters(p ...Parameter) *Parameters {
	var idx = make(map[string]int, len(p))
	for i, p := range p {
		idx[p.Name] = i
	}
	return &Parameters{params: p, idx: idx}
}

func (p *Parameters) Get(name string) (any, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if i, ok := p.idx[name]; ok {
		return p.params[i].Value, true
	}
	return nil, false
}

func (p *Parameters) Set(name string, value any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if i, ok := p.idx[name]; ok {
		p.params[i].Value = value
	} else {
		p.idx[name] = len(p.params)
		p.params = append(p.params, Parameter{Name: name, Value: value})
	}
}

func (p *Parameters) Delete(name string) {
	if i, ok := p.idx[name]; ok {
		p.params = append(p.params[:i], p.params[i+1:]...)
		delete(p.idx, name)
	}
}

func (p *Parameters) String() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	for i := range p.params {
		fmt.Fprintf(tw, "%s:\t%v\n", p.params[i].Name, p.params[i].Value)
	}
	_ = tw.Flush()
	return buf.String()
}
