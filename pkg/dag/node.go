package dag

type Node struct {
	OutEdge    map[string]bool
	InEdge     map[string]bool
	OutCounter int
	InCounter  int
}

func NewNode(deps []string) *Node {
	taskNode := new(Node)
	taskNode.OutCounter = 0
	taskNode.InCounter = 0
	taskNode.OutEdge = make(map[string]bool)
	taskNode.InEdge = make(map[string]bool)
	for _, dep := range deps {
		taskNode.OutEdge[dep] = true
		taskNode.OutCounter += 1
	}
	return taskNode
}
