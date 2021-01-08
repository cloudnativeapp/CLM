package dag

import (
	"reflect"
	"testing"
)

func initTest() DAG {
	dag := new(DAG)
	dag.Init()
	dag.AddNode("a", []string{"b"})
	dag.AddNode("b", []string{"d", "c"})
	dag.AddNode("c", []string{})
	dag.AddNode("d", []string{"c"})
	return *dag
}

func initTestErr() DAG {
	dag := new(DAG)
	dag.Init()
	dag.AddNode("a", []string{"b"})
	dag.AddNode("b", []string{"d", "c"})
	dag.AddNode("c", []string{"a"})
	dag.AddNode("d", []string{"c"})
	return *dag
}

func TestDAG_Shape_Err(t *testing.T) {
	dag := initTestErr()
	if ok := dag.Shape(); ok {
		t.Errorf("test dag shape failed")
	}
}

func TestDAG_Shape(t *testing.T) {
	dag := initTest()
	if ok := dag.Shape(); !ok {
		t.Errorf("test dag shape failed")
	}
}

func TestDAG_GetInstallQueue(t *testing.T) {
	dag := initTest()
	dag.Shape()
	if q, ok := dag.GetInstallQueue(); !ok {
		t.Errorf("get install queue failed")
	} else {
		t.Log(q)
		if !reflect.DeepEqual(q, []string{"c", "d", "b", "a"}) {
			t.Errorf("get wrong install queue")
		}
	}
}

func TestDAG_GetUninstallQueue(t *testing.T) {
	dag := initTest()
	dag.Shape()
	if q, ok := dag.GetUninstallQueue(); !ok {
		t.Errorf("get uninstall queue failed")
	} else {
		t.Log(q)
		if !reflect.DeepEqual(q, []string{"a", "b", "d", "c"}) {
			t.Errorf("get wrong uninstall queue")
		}
	}

}

func TestDAG_GetUninstallQueueOfNode(t *testing.T) {
	dag := initTest()
	dag.Shape()
	if q, ok := dag.GetUninstallQueueOfNode("d"); !ok {
		t.Errorf("get uninstall queue failed")
	} else {
		t.Log(q)
		if !reflect.DeepEqual(q, []string{"a", "b", "d"}) {
			t.Errorf("get wrong uninstall queue of node %s", "d")
		}
	}
}
