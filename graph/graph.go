package graph

import "fmt"

// Sentinel node names.
const (
	START = "__start__"
	END   = "__end__"
)

// edge represents a static edge between two nodes.
type edge struct {
	from string
	to   string
}

// conditionalEdge represents a dynamic edge that routes based on state.
type conditionalEdge[S any] struct {
	from    string
	router  RouterFunc[S]
	targets map[string]string // router return value → target node name
}

// GraphOption configures a Graph.
type GraphOption[S any] func(*Graph[S])

// Graph is a generic graph parameterized by state type S.
type Graph[S any] struct {
	name             string
	nodes            map[string]NodeFunc[S]
	edges            []edge
	conditionalEdges []conditionalEdge[S]
	entryPoint       string
	endPoints        map[string]bool
}

// NewGraph creates a new graph.
func NewGraph[S any](name string, opts ...GraphOption[S]) *Graph[S] {
	g := &Graph[S]{
		name:      name,
		nodes:     make(map[string]NodeFunc[S]),
		endPoints: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// AddNode adds a processing node to the graph.
func (g *Graph[S]) AddNode(name string, fn NodeFunc[S]) {
	g.nodes[name] = fn
}

// AddEdge adds a static edge between two nodes.
func (g *Graph[S]) AddEdge(from, to string) {
	g.edges = append(g.edges, edge{from: from, to: to})
}

// AddConditionalEdge adds a dynamic routing edge.
// The router function returns a key, which is looked up in targets to find the next node.
func (g *Graph[S]) AddConditionalEdge(from string, router RouterFunc[S], targets map[string]string) {
	g.conditionalEdges = append(g.conditionalEdges, conditionalEdge[S]{
		from:    from,
		router:  router,
		targets: targets,
	})
}

// SetEntryPoint sets the start node of the graph.
func (g *Graph[S]) SetEntryPoint(name string) {
	g.entryPoint = name
}

// SetEndPoint marks node(s) as terminal.
func (g *Graph[S]) SetEndPoint(names ...string) {
	for _, n := range names {
		g.endPoints[n] = true
	}
}

// Compile validates the graph and returns a CompiledGraph ready for execution.
func (g *Graph[S]) Compile() (*CompiledGraph[S], error) {
	if g.entryPoint == "" {
		return nil, fmt.Errorf("graph %q: no entry point set", g.name)
	}

	// Entry point must be a node or START
	if g.entryPoint != START {
		if _, ok := g.nodes[g.entryPoint]; !ok {
			return nil, fmt.Errorf("graph %q: entry point %q is not a node", g.name, g.entryPoint)
		}
	}

	// Validate all edges reference known nodes or END
	allNodes := make(map[string]bool)
	for name := range g.nodes {
		allNodes[name] = true
	}
	allNodes[START] = true
	allNodes[END] = true

	for _, e := range g.edges {
		if !allNodes[e.from] {
			return nil, fmt.Errorf("graph %q: edge from unknown node %q", g.name, e.from)
		}
		if !allNodes[e.to] {
			return nil, fmt.Errorf("graph %q: edge to unknown node %q", g.name, e.to)
		}
	}

	for _, ce := range g.conditionalEdges {
		if !allNodes[ce.from] {
			return nil, fmt.Errorf("graph %q: conditional edge from unknown node %q", g.name, ce.from)
		}
		for _, target := range ce.targets {
			if !allNodes[target] {
				return nil, fmt.Errorf("graph %q: conditional edge target unknown node %q", g.name, target)
			}
		}
	}

	// Build adjacency: node → list of static targets
	staticAdj := make(map[string][]string)
	for _, e := range g.edges {
		staticAdj[e.from] = append(staticAdj[e.from], e.to)
	}

	condAdj := make(map[string]conditionalEdge[S])
	for _, ce := range g.conditionalEdges {
		condAdj[ce.from] = ce
	}

	// Verify every non-END node has at least one outgoing edge
	for name := range g.nodes {
		if g.endPoints[name] {
			continue
		}
		_, hasStatic := staticAdj[name]
		_, hasCond := condAdj[name]
		if !hasStatic && !hasCond {
			return nil, fmt.Errorf("graph %q: node %q has no outgoing edges", g.name, name)
		}
	}

	return &CompiledGraph[S]{
		name:       g.name,
		nodes:      g.nodes,
		staticAdj:  staticAdj,
		condAdj:    condAdj,
		entryPoint: g.entryPoint,
		endPoints:  g.endPoints,
		maxIter:    25,
	}, nil
}
