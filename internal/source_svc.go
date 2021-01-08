package internal

import (
	"cloudnativeapp/clm/pkg/implement"
	"cloudnativeapp/clm/pkg/utils"
	"encoding/json"
	"errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

var sLog = ctrl.Log.WithName("source")

var SourcesRegistered = struct {
	m map[string]implement.Implement
	sync.RWMutex
}{
	m: make(map[string]implement.Implement),
}

//AddSource  add source to clm memory.
func AddSource(name string, s implement.Implement) bool {
	defer SourcesRegistered.Unlock()
	SourcesRegistered.Lock()
	if _, ok := SourcesRegistered.m[name]; ok {
		sLog.V(utils.Warn).Info("source exist already", "name", name)
		return false
	}
	sLog.V(utils.Info).Info("source add success", "name", name)
	SourcesRegistered.m[name] = s
	return true
}

//DeleteSource  delete source from clm.
func DeleteSource(name string) bool {
	defer SourcesRegistered.Unlock()
	SourcesRegistered.Lock()
	if _, ok := SourcesRegistered.m[name]; !ok {
		sLog.V(utils.Warn).Info("source does not exist", "name", name)
		return false
	}
	sLog.V(utils.Info).Info("source delete success", "name", name)
	delete(SourcesRegistered.m, name)
	return true
}

//GetSource   get the source configuration from clm.
func GetSource(name string) (implement.Implement, bool) {
	defer SourcesRegistered.RUnlock()
	SourcesRegistered.RLock()
	if s, ok := SourcesRegistered.m[name]; !ok {
		sLog.V(utils.Warn).Info("source does not exist", "name", name)
		return implement.Implement{}, false
	} else {
		return s, true
	}
}

func installFromSource(source Source, targetName, targetVersion string) error {
	sLog.V(utils.Debug).Info("try to install from source", "source", source, "target name", targetName)
	if s, ok := GetSource(source.Name); !ok {
		err := errors.New(utils.ImplementNotFound)
		sLog.Error(err, "can not find source", "sourceName", source.Name)
		return err
	} else {
		var values map[string]interface{}
		if source.Values != nil && len(source.Values.Raw) != 0 {
			if err := json.Unmarshal(source.Values.Raw, &values); err != nil {
				sLog.Error(err, "can not unmarshal source value", "sourceName", source.Name)
				return err
			}
		}
		if err := s.Install(targetName, targetVersion, values); err != nil {
			sLog.Error(err, "install by implement failed", "sourceName", source.Name)
			return err
		} else {
			return nil
		}
	}
}

func uninstallFromSource(source Source, targetName, targetVersion string) error {
	sLog.V(utils.Debug).Info("try to uninstall from source", "source", source, "target name", targetName)
	if s, ok := GetSource(source.Name); !ok {
		err := errors.New(utils.ImplementNotFound)
		sLog.Error(err, "can not find source", "sourceName", source.Name)
		return err
	} else {
		var values map[string]interface{}
		if source.Values != nil && len(source.Values.Raw) != 0 {
			if err := json.Unmarshal(source.Values.Raw, &values); err != nil {
				sLog.Error(err, "can not unmarshal source value", "sourceName", source.Name)
				return err
			}
		}
		if err := s.Uninstall(targetName, targetVersion, values); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func recoverFromSource(source Source, targetName, targetVersion string) error {
	sLog.V(utils.Debug).Info("try to recover from source", "source", source, "target name", targetName)
	if s, ok := GetSource(source.Name); !ok {
		err := errors.New(utils.ImplementNotFound)
		sLog.Error(err, "can not find source", "sourceName", source.Name)
		return err
	} else {
		var values map[string]interface{}
		if source.Values != nil && len(source.Values.Raw) != 0 {
			if err := json.Unmarshal(source.Values.Raw, &values); err != nil {
				sLog.Error(err, "can not unmarshal source value", "sourceName", source.Name)
				return err
			}
		}
		if err := s.Recover(targetName, targetVersion, values); err != nil {
			sLog.Error(err, "recover by implement failed", "sourceName", source.Name)
			return err
		} else {
			return nil
		}
	}
}

func upgradeFromSource(source Source, targetName, targetVersion string) error {
	sLog.V(utils.Debug).Info("try to upgrade from source", "source", source, "target name", targetName)
	if s, ok := GetSource(source.Name); !ok {
		err := errors.New(utils.ImplementNotFound)
		sLog.Error(err, "can not find source", "sourceName", source.Name)
		return err
	} else {
		var values map[string]interface{}
		if source.Values != nil && len(source.Values.Raw) != 0 {
			if err := json.Unmarshal(source.Values.Raw, &values); err != nil {
				sLog.Error(err, "can not unmarshal source value", "sourceName", source.Name)
				return err
			}
		}
		if err := s.Upgrade(targetName, targetVersion, values); err != nil {
			sLog.Error(err, "upgrade by implement failed", "sourceName", source.Name)
			return err
		} else {
			return nil
		}
	}
}
