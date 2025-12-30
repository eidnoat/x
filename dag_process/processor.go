package dag_process

import (
	"context"
	"maps"
	"reflect"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type ComputeFunc func(ctx context.Context, data map[AttrID]any, attr AttrID) (any, error)

type Processor struct {
	g *Graph

	rwm   sync.RWMutex
	data  map[AttrID]any
	input map[AttrID]any
	dirty map[AttrID]bool

	parallelLimit int
}

func NewProcessor(ctx context.Context, data map[AttrID]any, input map[AttrID]any) *Processor {
	if data == nil {
		data = make(map[AttrID]any)
	}
	if input == nil {
		input = make(map[AttrID]any)
	}
	return &Processor{g: G.Load(), data: maps.Clone(data), input: maps.Clone(input), dirty: make(map[AttrID]bool), parallelLimit: max(Cfg.ParallelLimit, 1)}
}

func (p *Processor) Process(ctx context.Context) error {
	type computeResult struct {
		val     any
		changed bool
	}

	var (
		remainDeps        = make(map[AttrID]*int32)
		eg, egCtx         = errgroup.WithContext(ctx)
		unchangedRet      = &computeResult{}
		queue             = make(chan AttrID, len(p.g.nodes))
		waitingProcessCnt = atomic.Int32{}
		compute           func(ctx context.Context, attr AttrID) (*computeResult, error)
		runNode           func(ctx context.Context, attr AttrID) error
		work              func(ctx context.Context) error
	)
	waitingProcessCnt.Store(int32(len(p.g.nodes)))
	if waitingProcessCnt.Load() == 0 {
		return nil
	}

	for attr, node := range p.g.nodes {
		cnt := int32(len(node.Parents))
		remainDeps[attr] = &cnt
		if cnt == 0 {
			queue <- attr
		}
	}

	compute = func(ctx context.Context, attr AttrID) (*computeResult, error) {
		depDirty := false
		p.rwm.RLock()
		for _, depID := range p.g.nodes[attr].Parents {
			if p.dirty[depID] {
				depDirty = true
				break
			}
		}
		p.rwm.RUnlock()
		if !depDirty && p.input[attr] == nil {
			return unchangedRet, nil
		}

		var (
			oldVal any
			newVal any
			err    error
		)
		p.rwm.RLock()
		oldVal = p.data[attr]
		p.rwm.RUnlock()

		if v := p.input[attr]; v != nil {
			newVal = v
		} else if p.g.nodes[attr].ComputeFunc != nil {
			p.rwm.RLock()
			data := make(map[AttrID]any)
			for _, depID := range p.g.nodes[attr].Parents {
				data[depID] = p.data[depID]
			}
			p.rwm.RUnlock()

			newVal, err = p.g.nodes[attr].ComputeFunc(ctx, data, attr)
		} else {
			newVal = oldVal
		}
		if err != nil {
			return nil, err
		}

		changed := false
		if !reflect.DeepEqual(newVal, oldVal) {
			changed = true
		}
		return &computeResult{val: newVal, changed: changed}, nil
	}

	runNode = func(ctx context.Context, attr AttrID) error {
		defer func() {
			if v := waitingProcessCnt.Add(-1); v == 0 {
				close(queue)
			}
		}()

		ret, err := compute(ctx, attr)
		if err != nil {
			return err
		}
		if ret.changed {
			p.rwm.Lock()
			p.data[attr], p.dirty[attr] = ret.val, true
			p.rwm.Unlock()
		}

		for _, child := range p.g.nodes[attr].Children {
			if v := atomic.AddInt32(remainDeps[child], -1); v > 0 {
				continue
			}
			queue <- child
		}

		return nil
	}

	work = func(ctx context.Context) error {
		for {
			select {
			case <-egCtx.Done():
				return nil
			case attr, ok := <-queue:
				if !ok {
					return nil
				}
				if err := runNode(ctx, attr); err != nil {
					return err
				}
			}
		}
	}

	for i := 0; i < p.parallelLimit; i++ {
		eg.Go(func() error { return work(egCtx) })
	}

	return eg.Wait()
}
