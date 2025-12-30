package dag_process

import (
	"context"
	"maps"
	"reflect"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type ComputeFunc func(ctx context.Context, data map[AttrID]any, attrID AttrID) (any, error)

type Processor struct {
	g *Graph

	rwm   sync.RWMutex
	data  map[AttrID]any
	dirty map[AttrID]bool

	parallelLimit int
}

func NewProcessor(ctx context.Context, data map[AttrID]any) *Processor {
	return &Processor{g: G.Load(), data: maps.Clone(data), dirty: make(map[AttrID]bool), parallelLimit: max(Cfg.ParallelLimit, 1)}
}

func (p *Processor) Input(ctx context.Context, input map[AttrID]any) {
	p.rwm.Lock()
	defer p.rwm.Unlock()
	for id, newVal := range input {
		oldVal := p.data[id]
		if reflect.DeepEqual(newVal, oldVal) {
			continue
		}

		p.data[id], p.dirty[id] = newVal, true
	}
}

func (p *Processor) Process(ctx context.Context) error {
	type computeResult struct {
		val     any
		changed bool
	}

	var (
		remainDeps   = make(map[AttrID]*int32)
		eg, egCtx    = errgroup.WithContext(ctx)
		unchangedRet = &computeResult{}
		sema         = make(chan struct{}, p.parallelLimit)
		compute      func(ctx context.Context, id AttrID) (*computeResult, error)
		runNode      func(ctx context.Context, id AttrID) error
	)

	for attrID, node := range p.g.nodes {
		cnt := int32(len(node.Parents))
		remainDeps[attrID] = &cnt
	}

	compute = func(ctx context.Context, id AttrID) (*computeResult, error) {
		if p.g.nodes[id].ComputeFunc == nil {
			return unchangedRet, nil
		}

		depDirty := false
		for _, depID := range p.g.nodes[id].Parents {
			if p.dirty[depID] {
				depDirty = true
				break
			}
		}
		if !depDirty {
			return unchangedRet, nil
		}

		p.rwm.RLock()
		data := make(map[AttrID]any)
		for _, depID := range p.g.nodes[id].Parents {
			data[depID] = p.data[depID]
		}
		p.rwm.RUnlock()

		select {
		case sema <- struct{}{}:
			defer func() { <-sema }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		newVal, err := p.g.nodes[id].ComputeFunc(ctx, data, id)
		if err != nil {
			return nil, err
		}

		changed := false
		if !reflect.DeepEqual(newVal, p.data[id]) {
			changed = true
		}
		return &computeResult{val: newVal, changed: changed}, nil
	}

	runNode = func(ctx context.Context, id AttrID) error {
		ret, err := compute(ctx, id)
		if err != nil {
			return err
		}
		if ret.changed {
			p.rwm.Lock()
			p.data[id], p.dirty[id] = ret.val, true
			p.rwm.Unlock()
		}

		for _, child := range p.g.nodes[id].Children {
			if v := atomic.AddInt32(remainDeps[child], -1); v > 0 {
				continue
			}
			eg.Go(func() error { return runNode(ctx, child) })
		}

		return nil
	}

	for id, remainDep := range remainDeps {
		if atomic.LoadInt32(remainDep) > 0 {
			continue
		}
		eg.Go(func() error { return runNode(egCtx, id) })
	}

	return eg.Wait()
}
