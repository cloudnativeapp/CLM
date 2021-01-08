package dag

import (
	"container/list"
)

type DAG struct {
	graph  map[string]*Node
	shaped bool
	e      []string
	d      []string
}

func (d *DAG) Init() {
	if d.graph == nil {
		d.graph = make(map[string]*Node)
	}
}

//AddNode  Add one node with dependencies.
func (d *DAG) AddNode(name string, deps []string) bool {
	if _, v := d.graph[name]; v {
		return false
	}
	node := NewNode(deps)
	d.graph[name] = node
	// Need re-shape
	d.shaped = false
	return true
}

func (d *DAG) addNodeByNode(name string, node Node) bool {
	if _, v := d.graph[name]; v {
		return false
	}
	n := Node{}
	n.InCounter = node.InCounter
	n.OutCounter = node.OutCounter
	n.InEdge = node.InEdge
	n.OutEdge = node.OutEdge
	d.graph[name] = &n
	// Need re-shape
	d.shaped = false
	return true
}

//Shape : Check the DAG
func (d *DAG) Shape() bool {
	graph := d.graph
	for name, node := range graph {
		for dep := range node.OutEdge {
			if v, ok := graph[dep]; !ok {
				return false
			} else {
				v.InEdge[name] = true
				v.InCounter += 1
			}
		}
	}
	ok := d.enqueue() && d.dequeue()
	if !ok {
		return false
	}
	d.shaped = true
	return true
}

func (d *DAG) queue(enqueue bool) (q []string, success bool) {
	graph := d.graph
	stack := list.New()
	tmpCounter := make(map[string]int)
	for name, node := range graph {
		if enqueue {
			tmpCounter[name] = node.OutCounter
		} else {
			tmpCounter[name] = node.InCounter
		}
		if tmpCounter[name] == 0 {
			stack.PushBack(name)
		}
	}

	count := 0
	for stack.Len() > 0 {
		count++
		item := stack.Front()
		stack.Remove(item)
		name := item.Value.(string)
		q = append(q, name)
		node := graph[name]

		var target map[string]bool
		if enqueue {
			target = node.InEdge
		} else {
			target = node.OutEdge
		}
		for iter := range target {
			tmpCounter[iter] -= 1
			if tmpCounter[iter] == 0 {
				stack.PushBack(iter)
			}
		}
	}

	if count != len(graph) {
		return q, false
	}
	return q, true
}

// 安装出度为0
func (d *DAG) enqueue() bool {
	q, ok := d.queue(true)
	if ok {
		d.e = q
	}
	return ok
}

// 卸载入度为0
func (d *DAG) dequeue() bool {
	q, ok := d.queue(false)
	if ok {
		d.d = q
	}
	return ok
}

func (d *DAG) isShaped() bool {
	return d.shaped
}

//GetInstallQueue  Get the install queue of DAG.
func (d *DAG) GetInstallQueue() (q []string, success bool) {
	if !d.isShaped() && !d.Shape() {
		success = false
		return
	}

	return d.e, true
}

//GetUninstallQueue  Get the uninstall queue of DAG.
func (d *DAG) GetUninstallQueue() (q []string, success bool) {
	if !d.isShaped() && !d.Shape() {
		success = false
		return
	}

	return d.d, true
}

//GetUninstallQueueOfNode  Get uninstall queue when uninstall one node.
func (d *DAG) GetUninstallQueueOfNode(name string) (q []string, success bool) {
	graph := d.graph
	stack := list.New()
	if _, ok := graph[name]; !ok {
		return nil, false
	} else {
		stack.PushBack(name)
	}

	tmp := new(DAG)
	tmp.Init()

	for stack.Len() > 0 {
		item := stack.Front()
		stack.Remove(item)
		name := item.Value.(string)
		node := graph[name]
		tmp.addNodeByNode(name, *node)

		for n := range node.InEdge {
			if _, ok := graph[n]; !ok {
				success = false
				return
			} else {
				stack.PushBack(n)
			}
		}
	}

	return tmp.queue(false)
}
