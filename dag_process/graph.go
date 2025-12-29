package dag_process

import (
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/pkg/errors"
)

var (
	G atomic.Pointer[Graph]
)

type AttrID string

type Node struct {
	ID           AttrID
	Dependencies []AttrID
	ComputeFunc  ComputeFunc
}

type Graph struct {
	nodes  map[AttrID]*Node
	layers [][]AttrID
}

func NewGraph() *Graph {
	g := &Graph{make(map[AttrID]*Node), make([][]AttrID, 0)}
	return g
}

func (g *Graph) Register(attrID AttrID, dependencies []AttrID, fn ComputeFunc) error {
	if attrID == "" || (len(dependencies) > 0 && fn == nil) {
		return errors.New(fmt.Sprintf("attrID[%v] or fn[%v] is nil", attrID, fn))
	}

	deps := slices.Clone(dependencies)
	slices.Sort(deps)
	g.nodes[attrID] = &Node{attrID, slices.Compact(deps), fn}
	return nil
}

func (g *Graph) Compile() error {
	indegree := make(map[AttrID]int)
	for _, node := range g.nodes {
		indegree[node.ID] = 0
		for _, dep := range node.Dependencies {
			if _, exist := g.nodes[dep]; !exist {
				return errors.New(fmt.Sprintf("node [%s] depends on missing node [%s]", node.ID, dep))
			}
			indegree[node.ID]++
		}
	}

	queue, cnt := make([]AttrID, 0), 0
	for id, v := range indegree {
		if v != 0 {
			continue
		}
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		g.layers, cnt = append(g.layers, slices.Clone(queue)), cnt+len(queue)

		nextQueue := make([]AttrID, 0)
		for _, id := range queue {
			for _, node := range g.nodes {
				if slices.Contains(node.Dependencies, id) {
					indegree[node.ID]--
					if indegree[node.ID] == 0 {
						nextQueue = append(nextQueue, node.ID)
					}
				}
			}
		}
		slices.Sort(nextQueue)
		queue = nextQueue
	}
	if cnt < len(g.nodes) {
		return errors.New("cycle detected in graph")
	}

	return nil
}
