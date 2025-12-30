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
	g             *Graph
	data          map[AttrID]any
	dirty         map[AttrID]bool
	parallelLimit int
}

func NewProcessor(ctx context.Context, data map[AttrID]any) *Processor {
	return &Processor{g: G.Load(), data: maps.Clone(data), dirty: make(map[AttrID]bool), parallelLimit: Cfg.ParallelLimit}
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

func (p *Processor) Process(ctx context.Context) error {
	type computeResult struct {
		attrID  AttrID
		result  any
		changed bool
	}

	for _, layer := range p.g.layers {
		fns, results := make([]func(ctx context.Context) error, 0), make(chan *computeResult, len(layer))
		for _, attrID := range layer {
			if p.g.nodes[attrID].ComputeFunc == nil {
				continue
			}

			fns = append(fns, func(ctx context.Context) error {
				dependenciesDirty := false
				for _, dependency := range p.g.nodes[attrID].Dependencies {
					if p.dirty[dependency] {
						dependenciesDirty = true
						break
					}
				}
				if !dependenciesDirty {
					return nil
				}

				newVal, err := p.g.nodes[attrID].ComputeFunc(ctx, p.data, attrID)
				if err != nil {
					return err
				}

				results <- &computeResult{attrID: attrID, result: newVal, changed: !reflect.DeepEqual(p.data[attrID], newVal)}
				return nil
			})
		}

		eg, newCtx := errgroup.WithContext(ctx)
		eg.SetLimit(p.parallelLimit)
		for _, fn := range fns {
			eg.Go(func() error { return fn(newCtx) })
		}
		if err := eg.Wait(); err != nil {
			return errors.WithMessagef(err, "process err when Propagate")
		}

		close(results)
		for ret := range results {
			if !ret.changed {
				continue
			}
			p.data[ret.attrID], p.dirty[ret.attrID] = ret.result, true
		}
	}

	return nil
}
