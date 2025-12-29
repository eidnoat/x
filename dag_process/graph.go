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
	g := &Graph{make(map[AttrID]*Node), make([][]AttrID, 1)}
	return g
}

func (g *Graph) Register(attrID AttrID, dependencies []AttrID, cf ComputeFunc) error {
	if attrID == "" || cf == nil {
		return errors.New(fmt.Sprintf("attrID[%v] or cf[%v] is nil", attrID, cf))
	}
	if len(dependencies) == 0 && !slices.Contains(g.layers[0], attrID) {
		g.layers[0] = append(g.layers[0], attrID)
	}

	slices.Sort(dependencies)
	g.nodes[attrID] = &Node{attrID, slices.Compact(dependencies), cf}
	return nil
}

func (g *Graph) Compile() error {
	type entry struct {
		attrID AttrID
		layer  int
	}

	var (
		path    = make(map[AttrID]bool)
		visited = make(map[AttrID]bool)
		dfs     func(entry *entry) bool
	)
	dfs = func(e *entry) bool {
		if path[e.attrID] {
			return true
		}
		if visited[e.attrID] {
			return false
		}

		path[e.attrID], visited[e.attrID] = true, true

		if e.layer > len(g.layers)-1 {
			g.layers = append(g.layers, []AttrID{e.attrID})
		} else {
			g.layers[e.layer] = append(g.layers[e.layer], e.attrID)
		}
		for _, dependency := range g.nodes[e.attrID].Dependencies {
			if dfs(&entry{dependency, e.layer + 1}) {
				return true
			}
		}

		path[e.attrID] = false
		return false
	}
	for _, attrID := range g.layers[0] {
		if dfs(&entry{attrID, 0}) {
			return errors.New("cycle detected in dependency graph")
		}
	}

	return nil
}
