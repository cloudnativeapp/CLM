package internal

import (
	"cloudnativeapp/clm/pkg/utils"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

type CRDReleasePhase string

const (
	// CRDReleaseRunning means all dependencies have been met and all modules are ready
	CRDReleaseRunning    CRDReleasePhase = "Running"
	CRDReleaseAbnormal   CRDReleasePhase = "Abnormal"
	CRDReleaseInstalling CRDReleasePhase = "Installing"
)

type CRDReleaseCondition struct {
	Type   CRDReleaseConditionType       `json:"type,omitempty"`
	Status apiextensions.ConditionStatus `json:"status,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime v1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
}

type CRDReleaseConditionType string

const (
	// Start handle the crd release by clm.
	CRDReleaseInitialized CRDReleaseConditionType = "Initialized"
	// All dependencies are satisfied, and begin to install modules.
	CRDReleasesDependenciesSatisfied CRDReleaseConditionType = "DependenciesSatisfied"
	// All modules are ready to work.
	CRDReleaseModulesReady CRDReleaseConditionType = "ModulesReady"
	// It means crd release is ready to work now.
	CRDReleaseReady CRDReleaseConditionType = "Ready"
)

var log = ctrl.Log.WithName("crd release status")

var statusRecorder = struct {
	m map[string]interface{}
	sync.RWMutex
}{
	m: make(map[string]interface{}),
}

//RecordStatus  record crd release status to clm memory.
func RecordStatus(name, version string, s interface{}) bool {
	defer statusRecorder.Unlock()
	statusRecorder.Lock()
	key := name + ":" + version
	if _, ok := statusRecorder.m[key]; ok {
		log.V(utils.Info).Info("crd release status exist already", "name", name)
		return false
	}
	statusRecorder.m[key] = s
	return true
}

//DeleteStatus  delete crd release status from clm.
func DeleteStatus(name, version string) bool {
	defer statusRecorder.Unlock()
	statusRecorder.Lock()
	key := name + ":" + version
	if _, ok := statusRecorder.m[key]; !ok {
		log.V(utils.Info).Info("crd release status does not exist", "name", name)
		return false
	}
	delete(statusRecorder.m, key)
	return true
}

//GetStatus   get the crd release status from clm.
func GetStatus(name, version string) (interface{}, bool) {
	defer statusRecorder.RUnlock()
	statusRecorder.RLock()
	key := name + ":" + version
	if s, ok := statusRecorder.m[key]; !ok {
		log.V(utils.Info).Info("crd release status does not exist", "name", name)
		return nil, false
	} else {
		return s, true
	}
}
