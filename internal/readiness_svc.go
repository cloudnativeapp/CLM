package internal

import (
	"cloudnativeapp/clm/pkg/probe"
	"cloudnativeapp/clm/pkg/prober"
	"cloudnativeapp/clm/pkg/utils"
	"reflect"
	"strings"
	"sync"
	"time"
)

var Prober *prober.Prober

type ModuleReadinessState struct {
	State string
	Ch    chan bool
}

var ModuleReadinessStateMap = struct {
	m map[string]*ModuleReadinessState
	sync.RWMutex
}{
	m: make(map[string]*ModuleReadinessState),
}

//addModule  add the module to clm readiness map.
func addModule(name string) bool {
	defer ModuleReadinessStateMap.Unlock()
	ModuleReadinessStateMap.Lock()
	key := name
	if _, ok := ModuleReadinessStateMap.m[key]; ok {
		mLog.V(utils.Warn).Info("module exist already", "name", name)
		return false
	}
	ModuleReadinessStateMap.m[key] = &ModuleReadinessState{
		State: ModuleInstalling,
	}
	return true
}

//deleteModule  delete the module from the clm readiness map.
func deleteModule(name string) bool {
	defer ModuleReadinessStateMap.Unlock()
	ModuleReadinessStateMap.Lock()
	key := name
	if v, ok := ModuleReadinessStateMap.m[key]; !ok {
		mLog.V(utils.Warn).Info("module does not exist", "name", name)
		return false
	} else {
		if v.Ch != nil {
			close(v.Ch)
			v.Ch = nil
		}
		delete(ModuleReadinessStateMap.m, key)
		return true
	}
}

//updateModule  update the module state to the clm readiness map.
func updateModule(name, s string, ch chan bool) bool {
	defer ModuleReadinessStateMap.Unlock()
	ModuleReadinessStateMap.Lock()
	key := name
	if v, ok := ModuleReadinessStateMap.m[key]; !ok {
		mLog.V(utils.Warn).Info("module does not exist", "name", name)
		return false
	} else {
		v.State = s
		if ch != nil {
			if v.Ch != nil && v.Ch != ch {
				close(v.Ch)
			}
			v.Ch = ch
		}
		mLog.V(utils.Info).Info("update module status", "name", name, "status", v.State)
		return true
	}
}

func getModule(name string) string {
	defer ModuleReadinessStateMap.RUnlock()
	ModuleReadinessStateMap.RLock()
	key := name
	if v, ok := ModuleReadinessStateMap.m[key]; !ok {
		mLog.V(utils.Warn).Info("module does not exist", "name", name)
		return ""
	} else {
		mLog.V(utils.Info).Info("module status got", "name", name, "status", v.State)
		return v.State
	}
}

func readinessCheck(ch chan bool, m Module) {
	if reflect.DeepEqual(m.Readiness, probe.Probe{}) {
		// update to running when no readiness setting.
		updateModule(m.Name, ModuleRunning, nil)
		return
	}
	interval := m.Readiness.PeriodSeconds
	if interval < 3 {
		interval = 3
	}
	failureThreshold := m.Readiness.FailureThreshold
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	successThreshold := m.Readiness.SuccessThreshold
	if successThreshold < 1 {
		successThreshold = 1
	}
	recoverThreshold := m.Readiness.RecoverThreshold
	if recoverThreshold < 1 {
		recoverThreshold = 1
	}

	successCount := 0
	failureCount := 0
	ticker := time.NewTicker(time.Second * time.Duration(interval))

	for {
		select {
		case <-ch:
			return
		case <-ticker.C:
			result, out, err := Prober.RunProbeWithRetries(3, &m.Readiness)
			mLog.V(utils.Info).Info("readiness output", "out", out)
			if err == nil && result == probe.Success {
				if strings.ToLower(out) == "ready" {
					failureCount = 0
					successCount++
					if successCount >= m.Readiness.SuccessThreshold {
						updateModule(m.Name, ModuleRunning, nil)
						continue
					}
				}
			}

			if err != nil {
				mLog.Error(err, "readiness failed", "name", m.Name)
			} else { // result != probe.Success
				mLog.V(utils.Warn).Info("readiness warning", "name", m.Name)
			}
			s := getModule(m.Name)
			successCount = 0
			failureCount++
			if s == ModuleRecovering && failureCount >= recoverThreshold {
				mLog.V(utils.Info).Info("readiness update module", "from", s, "to", ModuleAbnormal)
				updateModule(m.Name, ModuleAbnormal, nil)
			} else if s == ModuleRunning && failureCount >= failureThreshold {
				mLog.V(utils.Info).Info("readiness update module", "from", s, "to", ModuleAbnormal)
				updateModule(m.Name, ModuleAbnormal, nil)
			}
		}
	}
}
