package dag_process

import (
	"context"
	"maps"
	"reflect"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type ComputeFunc func(ctx context.Context, data map[AttrID]any, attrID AttrID) (any, error)

type Processor struct {
	g     *Graph
	data  map[AttrID]any
	dirty map[AttrID]bool
}

func NewProcessor(ctx context.Context, data map[AttrID]any) *Processor {
	return &Processor{g: G.Load(), data: maps.Clone(data), dirty: make(map[AttrID]bool)}
}

func (p *Processor) Input(ctx context.Context, input map[AttrID]any) {
	for id, newVal := range input {
		oldVal, exist := p.data[id]
		if !exist || reflect.DeepEqual(newVal, oldVal) {
			continue
		}

		p.data[id], p.dirty[id] = newVal, true
	}
}

func (p *Processor) Propagate(ctx context.Context) error {
	type computeResult struct {
		attrID  AttrID
		result  any
		err     error
		changed bool
		skipped bool
	}

	limit := 5
	for i, layer := range p.g.layers {
		if i == 0 {
			continue
		}

		fns := make([]func(ctx context.Context) *computeResult, 0)
		for _, attrID := range layer {
			fns = append(fns, func(ctx context.Context) *computeResult {
				dependenciesDirty := false
				for _, dependency := range p.g.nodes[attrID].Dependencies {
					if p.dirty[dependency] {
						dependenciesDirty = true
						break
					}
				}
				if !dependenciesDirty {
					return &computeResult{attrID: attrID, skipped: true}
				}

				newVal, err := p.g.nodes[attrID].ComputeFunc(ctx, p.data, attrID)
				if err != nil {
					return &computeResult{attrID: attrID, err: err}
				}
				return &computeResult{attrID: attrID, result: newVal, changed: !reflect.DeepEqual(p.data[attrID], newVal)}
			})
		}

		eg, newCtx := errgroup.WithContext(ctx)
		eg.SetLimit(limit)
		for _, fn := range fns {
			eg.Go(func() error {
				ret := fn(newCtx)
				if ret.err != nil {
					return ret.err
				}
				if ret.skipped || !ret.changed {
					return nil
				}

				p.data[ret.attrID], p.dirty[ret.attrID] = ret.result, true

				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return errors.WithMessagef(err, "process err when Propagate")
		}
	}

	return nil
}
