package controllers

import (
	"cloudnativeapp/clm/api/v1beta1"
	"cloudnativeapp/clm/internal"
	"cloudnativeapp/clm/pkg/check/condition"
	"cloudnativeapp/clm/pkg/dag"
	"cloudnativeapp/clm/pkg/plugin"
	"cloudnativeapp/clm/pkg/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var releaseLog = ctrl.Log.WithName("crd release")
var MGRClient client.Client
var EventRecorder record.EventRecorder

const lastStatus = "clm.cloudnativeapp.io/last-configuration-status"
const MaxRecordLen = 32 * 1024

// CheckCRDRelease : Check all crd release status.
func CheckCRDRelease(c *v1beta1.CRDRelease) (bool, error) {
	// Check the status of dependencies.
	if ok, err := checkDependencies(c); err != nil || !ok {
		return false, err
	}
	updateCRDReleaseCondition(c, internal.CRDReleasesDependenciesSatisfied, apiextensions.ConditionTrue)
	// Check the status of modules.
	if ok, err := checkModules(c); err != nil || !ok {
		return false, err
	}
	// 修改状态
	c.Status.CurrentVersion = c.Spec.Version
	updateCRDReleaseCondition(c, internal.CRDReleaseModulesReady, apiextensions.ConditionTrue)
	updateCRDReleaseCondition(c, internal.CRDReleaseReady, apiextensions.ConditionTrue)
	return true, nil
}

func getActivedDependencies(c *v1beta1.CRDRelease) ([]plugin.Iplugin, error) {
	releaseLog.V(utils.Debug).Info("get dependencies union", "name", c.Name, "version", c.Spec.Version)
	dmap := make(map[string]internal.Dependency)
	for _, i := range c.Spec.Dependencies {
		if _, ok := dmap[i.Name]; ok {
			return nil, errors.New("repeated crd release dependency spec")
		} else {
			dmap[i.Name] = i
		}
	}
	var activeDeps []internal.DependencyStatus
	for _, i := range c.Status.Dependencies {
		if _, ok := dmap[i.Name]; ok {
			// Delete from status and do not delete release
			activeDeps = append(activeDeps, i)
		} else {
			releaseLog.V(utils.Info).Info("orphan dependency omitted", "name", c.Name,
				"version", c.Spec.Version, "dependency name", i.Name, "dependency version", i.Version)
		}
	}
	c.Status.Dependencies = activeDeps
	dependencies := make([]plugin.Iplugin, len(c.Spec.Dependencies))
	k := 0
	for _, j := range dmap {
		dependencies[k] = j
		k++
	}
	return dependencies, nil
}

//checkDependencies  Check whether dependencies are ready.
func checkDependencies(c *v1beta1.CRDRelease) (bool, error) {
	releaseLog.V(utils.Debug).Info("check dependencies", "name", c.Name, "version", c.Spec.Version)
	updateCRDReleaseCondition(c, internal.CRDReleaseInitialized, apiextensions.ConditionTrue)
	dependencies, err := getActivedDependencies(c)
	if err != nil {
		return false, err
	}
	ready, err := plugin.CheckPlugins(dependencies, dependencySetStatus(c), dependencyCheckStatus())
	if !ready {
		releaseLog.V(utils.Warn).Info("not all dependencies ready", "name", c.Name, "version", c.Spec.Version)
		updateCRDReleaseCondition(c, internal.CRDReleasesDependenciesSatisfied, apiextensions.ConditionFalse)
	} else {
		releaseLog.V(utils.Info).Info("all dependencies ready", "name", c.Name, "version", c.Spec.Version)
		updateCRDReleaseCondition(c, internal.CRDReleasesDependenciesSatisfied, apiextensions.ConditionTrue)
	}
	return ready, err
}

//dependencySetStatus : Check whether dependency reach the target status.
func dependencySetStatus(c *v1beta1.CRDRelease) plugin.StatusSet {
	return func(name string, version string, phase interface{}) (ok bool, e error) {
		p, o := phase.(internal.DependencyStatus)
		if !o {
			return
		}
		if p.Phase != internal.DependencyRunning {
			EventRecorder.Eventf(c, corev1.EventTypeWarning, "Dependency:"+string(p.Phase), p.Reason)
		}
		releaseLog.V(utils.Debug).Info("set phase from check", "phase", p)
		if p.Phase == internal.DependencyAbnormal || p.Phase == internal.DependencyAbsentErr || p.Phase == internal.DependencyPullingErr {
			e = errors.New(utils.DependencyStateAbnormal)
		}
		for j, i := range c.Status.Dependencies {
			if i.Name == name {
				if i.Version != version {
					releaseLog.V(utils.Info).Info("upgrade module status", "previous", i.Version,
						"current", version)
				} else {
					releaseLog.V(utils.Debug).Info("update dependency phase", "target phase", p, "name", name)
				}
				c.Status.Dependencies[j] = p
				return p.Phase == internal.DependencyRunning, e
			}
		}
		c.Status.Dependencies = append(c.Status.Dependencies, p)
		releaseLog.V(utils.Debug).Info("add dependency status", "phase", p, "name", name)
		return p.Phase == internal.DependencyRunning, e
	}
}

//dependencyCheckStatus : return plugin phase.
func dependencyCheckStatus() plugin.StatusGet {
	return func(name string, version string) (act plugin.Action, phase string, e error) {
		release := &v1beta1.CRDRelease{}
		if err := MGRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ""}, release); err != nil {
			err = client.IgnoreNotFound(err)
			if err == nil {
				releaseLog.Info("crd release absent", "name", name, "version", version)
				return plugin.NeedInstall, string(internal.DependencyDontCare), nil
			} else {
				return plugin.NeedRecover, string(internal.DependencyDontCare), err
			}
		}
		if release.Status.Phase == internal.CRDReleaseRunning && !utils.VersionMatch(release.Status.CurrentVersion, version, "") {
			releaseLog.Info("crd release version mismatch", "name", name,
				"current", release.Spec.Version, "expected", version)
			// upgrade crd release to target version
			return plugin.NeedUpgrade, string(internal.DependencyDontCare), nil
		}

		return plugin.NeedConvert, string(release.Status.Phase), nil
	}
}

func getModulesUnion(current []internal.Module, lastModuleMap map[string]internal.Module,
	imported []internal.Module) ([]plugin.Iplugin, error) {
	releaseLog.V(utils.Debug).Info("get modules union")
	dmap := make(map[string]internal.Module)
	for _, i := range current {
		if _, ok := dmap[i.Name]; ok {
			return nil, errors.New("repeated crd release module spec")
		} else {
			dmap[i.Name] = i
		}
	}

	for _, k := range imported {
		delete(dmap, k.Name)
	}
	if len(lastModuleMap) > 0 {
		for k, v := range lastModuleMap {
			if _, ok := dmap[v.Name]; !ok {
				dmap[k] = v
			}
		}
	}

	modules := make([]plugin.Iplugin, len(dmap))
	k := 0
	for _, j := range dmap {
		modules[k] = j
		k++
	}
	return modules, nil
}

//checkModules Check whether all modules are ready.
func checkModules(c *v1beta1.CRDRelease) (bool, error) {
	releaseLog.Info("check modules", "name", c.Name, "version", c.Spec.Version)
	//第一次进来的时候status都为空，进行一次全量external判断, 同时module需要注意external状态的破坏
	var modulesExclude []internal.Module
	if len(c.Status.Modules) == 0 {
		releaseLog.V(utils.Debug).Info("check new release modules", "name", c.Name, "version", c.Spec.Version)
		for _, m := range c.Spec.Modules {
			if ok, err := m.ConditionCheck(); err != nil {
				return false, err
			} else if !ok {
				if !reflect.DeepEqual(m.Conditions, condition.Condition{}) && m.Conditions.Strategy == condition.Import {
					releaseLog.V(utils.Debug).Info("condition check failed, take it as an imported module",
						"name", m.Name)
					moduleSetStatus(c)(m.Name, "", internal.GenerateImportedModuleStatus(m))
					EventRecorder.Eventf(c, corev1.EventTypeNormal, "Imported", "module %v imported", m.Name)
				} else {
					releaseLog.V(utils.Debug).Info("condition check failed, take it as an external module",
						"name", m.Name)
					moduleSetStatus(c)(m.Name, "", internal.GenerateExternalModuleStatus(m.Name))
					EventRecorder.Eventf(c, corev1.EventTypeNormal, "External", "module %v external", m.Name)
				}
				modulesExclude = append(modulesExclude, m)
			}
			releaseLog.V(utils.Debug).Info("condition check passed", "name", m.Name)
		}
	}

	mmap, update, err := getLastConfigModuleMap(c)
	if err != nil {
		return false, err
	}
	if update {
		releaseLog.V(utils.Info).Info("crd release updated", "crd release", c.Name, "version", c.Spec.Version)
	}

	modules, err := getModulesUnion(c.Spec.Modules, mmap, modulesExclude)
	if err != nil {
		return false, err
	}
	ready, err := plugin.CheckPlugins(modules, moduleSetStatus(c), moduleCheckStatus(c, mmap, update))
	if !ready {
		releaseLog.Info("not all modules ready", "name", c.Name, "version", c.Spec.Version)
		updateCRDReleaseCondition(c, internal.CRDReleaseModulesReady, apiextensions.ConditionFalse)
	} else {
		releaseLog.V(utils.Debug).Info("all modules ready", "name", c.Name, "version", c.Spec.Version)
		updateCRDReleaseCondition(c, internal.CRDReleaseModulesReady, apiextensions.ConditionTrue)
	}
	return ready, err
}

func getLastConfigModuleMap(c *v1beta1.CRDRelease) (map[string]internal.Module, bool, error) {
	mmap := make(map[string]internal.Module)
	var lastCRDRelease v1beta1.CRDRelease
	update := false
	str := c.Annotations[lastStatus]
	releaseLog.V(utils.Debug).Info(fmt.Sprintf("last release crd: %s", str), "crd release name",
		c.Name)
	if len(str) > 0 {
		if err := json.Unmarshal([]byte(str), &lastCRDRelease); err != nil {
			releaseLog.Error(err, "unmarshal last crd release failed")
			return nil, false, err
		}
		releaseLog.V(utils.Debug).Info(fmt.Sprintf("unmarshal last crd release %s", str))
		if c.Generation > lastCRDRelease.Generation+1 {
			update = true
		}
		for _, i := range lastCRDRelease.Spec.Modules {
			mmap[i.Name] = i
		}
	}
	return mmap, update, nil
}

func recordModuleState(c *v1beta1.CRDRelease, state *internal.ModuleState, name string) {
	if state.Abnormal != nil {
		EventRecorder.Eventf(c, corev1.EventTypeWarning, name+":Abnormal", "message:%v reason:%v",
			state.Abnormal.Message, state.Abnormal.Reason)
	}
	if state.Recovering != nil {
		EventRecorder.Eventf(c, corev1.EventTypeWarning, name+":Recovering", "message:%v reason:%v",
			state.Recovering.Message, state.Recovering.Reason)
	}
	if state.Terminated != nil {
		EventRecorder.Eventf(c, corev1.EventTypeNormal, name+":Terminated", "message:%v reason:%v",
			state.Terminated.Message, state.Terminated.Reason)
	}
	if state.Installing != nil {
		EventRecorder.Eventf(c, corev1.EventTypeNormal, name+":Installing", "message:%v reason:%v",
			state.Installing.Message, state.Installing.Reason)
	}
	if state.Running != nil {
		EventRecorder.Eventf(c, corev1.EventTypeNormal, name+":Running", "")
	}
}

//moduleSetStatus Set the module status.
func moduleSetStatus(c *v1beta1.CRDRelease) plugin.StatusSet {
	return func(name string, version string, status interface{}) (b bool, e error) {
		// e 来自conditioncheck，moduleCheckStatus，preCheck
		s, ok := status.(internal.ModuleStatus)
		if !ok {
			return
		}
		releaseLog.V(utils.Debug).Info(fmt.Sprintf("try to set module status %v", s),
			"crd release name", c.Name, "module", name)
		if s.State != nil && s.State.Abnormal != nil {
			e = errors.New(utils.ModuleStateAbnormal)
		}
		for i, j := range c.Status.Modules {
			if j.Name == name {
				releaseLog.V(utils.Debug).Info("update module status", "crd release name", name,
					"module", name)
				if s.State != nil && !internal.ModuleStateEqual(s.State, j.State) {
					recordModuleState(c, s.State, s.Name)
				}
				c.Status.Modules[i] = j.UpdateStatus(s)
				return s.Ready || (s.State != nil && s.State.Terminated != nil), e
			}
		}
		releaseLog.V(utils.Debug).Info("add module status", "crd release name", name, "module", name)
		recordModuleState(c, s.State, s.Name)
		c.Status.Modules = append(c.Status.Modules, status.(internal.ModuleStatus))
		return s.Ready || (s.State != nil && s.State.Terminated != nil), e
	}
}

//moduleCheckStatus Check the status of module, return the act and phase.
func moduleCheckStatus(c *v1beta1.CRDRelease, lastModuleMap map[string]internal.Module, crdUpdate bool) plugin.StatusGet {
	return func(name string, version string) (act plugin.Action, s string, e error) {
		var lastState *internal.ModuleState
		external := false
		lastApplied := lastModuleMap[name]
		for _, i := range c.Status.Modules {
			if i.Name == name {
				lastState = i.State
				if i.GetConditionStatus(internal.ModuleExternal) == apiextensions.ConditionTrue {
					external = true
					releaseLog.V(utils.Debug).Info("check external module", "crd release name", c.Name,
						"module", name)
				}
			}
		}
		releaseLog.V(utils.Debug).Info(fmt.Sprintf("last module %v", lastApplied),
			"crd release name", c.Name, "module", name)
		for _, i := range c.Spec.Modules {
			if i.Name == name {
				return i.CheckStatus(lastApplied, lastState, external, crdUpdate, moduleUpdateCondition(c))
			}
		}
		releaseLog.V(utils.Warn).Info("can not find module to check status, delete it", "name", c.Name,
			"version", c.Spec.Version, "module", name)
		return plugin.NeedUninstall, internal.ModuleDontCare, nil
	}
}

//moduleUpdateCondition Update the module condition.
func moduleUpdateCondition(c *v1beta1.CRDRelease) func(internal.ModuleCondition, string) {
	return func(m internal.ModuleCondition, name string) {
		releaseLog.V(utils.Debug).Info(fmt.Sprintf("try to update condition %v", m), "crd release name",
			c.Name, "module", name)
		for _, j := range c.Status.Modules {
			if j.Name == name {
				for _, i := range j.Conditions {
					if i.Type == m.Type {
						releaseLog.V(utils.Debug).Info(fmt.Sprintf("update module condition from %s to %s", i.Status, m.Status),
							"crd release name", c.Name, "module", name)
						if i.Status != m.Status {
							i.Status = m.Status
							i.LastTransitionTime = m.LastTransitionTime
							EventRecorder.Eventf(c, corev1.EventTypeNormal, "Module:"+string(m.Type),
								"module %v condition %v", name, m.Status)
						}
						return
					}
				}
				j.Conditions = append(j.Conditions, m)
				releaseLog.V(utils.Debug).Info("add module condition",
					"crd release name", c.Name, "module", name)
				EventRecorder.Eventf(c, corev1.EventTypeNormal, "Module:"+string(m.Type),
					"module %v condition %v", name, m.Status)
				return
			}
		}
		releaseLog.V(utils.Debug).Info("add module status", "crd release name", c.Name, "module", name)
		EventRecorder.Eventf(c, corev1.EventTypeNormal, "Module:"+string(m.Type),
			"module %v condition %v", name, m.Status)
		t := internal.ModuleStatus{}
		t.Name = name
		t.Conditions = append(t.Conditions, m)
		c.Status.Modules = append(c.Status.Modules, t)
	}
}

//UninstallCRDRelease Uninstall the crd release.
func UninstallCRDRelease(c *v1beta1.CRDRelease) (bool, error) {
	releaseLog.Info("try to uninstall feature", "name", c.Name, "version", c.Spec.Version)
	updateCRDReleaseCondition(c, internal.CRDReleaseReady, apiextensions.ConditionFalse)
	if ok, err := uninstallDependencies(c); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}
	updateCRDReleaseCondition(c, internal.CRDReleasesDependenciesSatisfied, apiextensions.ConditionFalse)
	// 检查module状态
	if err := uninstallModules(c); err != nil {
		return false, err
	}
	updateCRDReleaseCondition(c, internal.CRDReleaseModulesReady, apiextensions.ConditionFalse)

	// 修改状态
	c.Status.CurrentVersion = ""
	return true, nil
}

//uninstallDependencies Uninstall the dependencies using DAG.
func uninstallDependencies(c *v1beta1.CRDRelease) (bool, error) {
	releaseLog.V(utils.Debug).Info("try to uninstall dependencies", "crd release name", c.Name,
		"version", c.Spec.Version)
	releases := &v1beta1.CRDReleaseList{}
	if err := MGRClient.List(context.Background(), releases); err != nil {
		releaseLog.Error(err, "unable to fetch crd release list")
		return false, err
	}
	// 此处需要新增版本的条件
	d := new(dag.DAG)
	d.Init()
	for _, release := range releases.Items {
		deps, err := getDependencies(release.Spec.Dependencies)
		if err != nil {
			return false, err
		}
		d.AddNode(release.Name, deps)
	}
	d.Shape()
	q, ok := d.GetUninstallQueueOfNode(c.Name)
	if !ok {
		err := errors.New("can not find a queue for dependencies deleting")
		releaseLog.Error(err, "delete dependency failed")
		return false, err
	}
	releaseLog.V(utils.Debug).Info("uninstall sequence", "values", q)
	for _, n := range q {
		if n == c.Name {
			// 只需要删除自己, 直接去删除module
			releaseLog.V(utils.Info).Info("uninstall myself", "name", n)
			break
		}
		for _, release := range releases.Items {
			if release.Name == n && release.GetDeletionTimestamp() != nil {
				//正在删除中， =
				releaseLog.V(utils.Info).Info("release is deleting", "name", release.Name)
				return false, nil
			}
		}
		// Delete directly
		release, err := getRelease(releases.Items, n)
		if err != nil {
			return false, err
		}
		if err := MGRClient.Delete(context.Background(), &release); err != nil {
			return false, err
		}

		// 重新调度
		releaseLog.V(utils.Debug).Info("crd release starts to delete", "name", n)
		return false, nil
	}
	return true, nil
}

//uninstallModules Uninstall modules using plugin management.
func uninstallModules(c *v1beta1.CRDRelease) error {
	releaseLog.V(utils.Debug).Info("try to uninstall modules", "crd release name", c.Name,
		"version", c.Spec.Version)
	modules := make([]plugin.Iplugin, len(c.Spec.Modules))
	for i, j := range c.Spec.Modules {
		modules[i] = j
	}
	deleted, err := plugin.CheckPlugins(modules, moduleSetStatus(c), moduleDeleteCheck(c))
	if err != nil {
		return err
	}
	if !deleted {
		releaseLog.V(utils.Warn).Info("not all modules deleted", "name", c.Name, "version", c.Spec.Version)
		return errors.New("not all modules deleted")
	}
	releaseLog.V(utils.Debug).Info("all modules deleted", "crd release name", c.Name,
		"version", c.Spec.Version)
	return nil
}

//moduleDeleteCheck StatusGet function when delete module using plugin management.
func moduleDeleteCheck(c *v1beta1.CRDRelease) plugin.StatusGet {
	return func(name string, version string) (act plugin.Action, s string, e error) {
		for _, i := range c.Status.Modules {
			if i.Name == name {
				if i.GetConditionStatus(internal.ModuleExternal) == apiextensions.ConditionTrue {
					releaseLog.V(utils.Debug).Info("external module does not need uninstall", "crd release name",
						c.Name, "module", name)
					return plugin.NeedNothing, internal.ModuleRunning, nil
				}
				if i.State != nil && i.State.Terminated == nil {
					releaseLog.V(utils.Debug).Info("module check need uninstall", "crd release name",
						c.Name, "module", name)
					return plugin.NeedUninstall, internal.ModuleDontCare, nil
				}
			}
		}
		releaseLog.V(utils.Debug).Info("module check need nothing", "crd release name",
			c.Name, "module", name)
		return plugin.NeedNothing, internal.ModuleRunning, nil
	}
}

func getDependencies(ds []internal.Dependency) ([]string, error) {
	var result []string
	for _, d := range ds {
		result = append(result, d.Name)
	}

	return result, nil
}

func getRelease(fs []v1beta1.CRDRelease, name string) (v1beta1.CRDRelease, error) {
	for _, f := range fs {
		if name == f.Name {
			return f, nil
		}
	}

	return v1beta1.CRDRelease{}, errors.New(fmt.Sprintf("can not find feature %s", name))
}

//updateCRDReleaseStatus
func updateCRDReleaseStatus(release *v1beta1.CRDRelease, phase internal.CRDReleasePhase, reason string) {
	releaseLog.V(utils.Debug).Info("update crd release status", "name", release.Name,
		"phase", phase, "reason", reason)
	release.Status.Phase = phase
	release.Status.Reason = reason
}

func genRecordRelease(release v1beta1.CRDRelease) (v1beta1.CRDRelease, bool) {
	var result v1beta1.CRDRelease
	var record bool
	var lastCRDRelease v1beta1.CRDRelease
	str := release.Annotations[lastStatus]
	if len(str) > 0 {
		if err := json.Unmarshal([]byte(str), &lastCRDRelease); err != nil {
			releaseLog.Error(err, "unmarshal last crd release failed")
		}
	}
	releaseLog.V(utils.Debug).Info("last release state", "lastCRDRelease", lastCRDRelease,
		"crd release name", release.Name, "version", release.Spec.Version)
	applied := false
	if len(release.Status.Conditions) != 0 {
		for _, c := range release.Status.Conditions {
			if c.Type == internal.CRDReleasesDependenciesSatisfied && c.Status == apiextensions.ConditionTrue {
				applied = true
			}
		}
	}
	if !applied {
		result.Spec.Modules = lastCRDRelease.Spec.Modules
	} else {
		result.Spec.Modules = release.Spec.Modules
	}

	// Record them for backup.
	result.Name = release.Name
	result.Namespace = release.Namespace
	result.Generation = release.Generation
	result.Spec.Version = release.Spec.Version
	result.Spec.Dependencies = release.Spec.Dependencies

	if reflect.DeepEqual(lastCRDRelease.Spec.Modules, result.Spec.Modules) {
		record = false
	} else {
		releaseLog.V(utils.Debug).Info("spec diff from last applied", "last", lastCRDRelease.Spec,
			"current", release.Spec, "crd release name", release.Name)
		record = true
	}

	return result, record
}

//updateReleaseCheck Check whether spec.module and status changed
func updateReleaseCheck(release *v1beta1.CRDRelease) (bool, error) {
	releaseLog.V(utils.Debug).Info("try to compare with record status", "crd release name", release.Name,
		"version", release.Spec.Version)
	var statusRecord v1beta1.CRDReleaseStatus
	tmp, ok := internal.GetStatus(release.Name, release.Spec.Version)
	if ok {
		statusRecord = tmp.(v1beta1.CRDReleaseStatus)
	} else {
		releaseLog.V(utils.Warn).Info("no statusRecorded crd release got", "crd release name", release.Name,
			"version", release.Spec.Version)
		return false, errors.New("no statusRecorded crd release got from clm memory")
	}
	statusEqual := reflect.DeepEqual(release.Status, statusRecord)
	if !statusEqual {
		releaseLog.V(utils.Debug).Info("status changed since last apply", "last", statusRecord,
			"current", release.Status, "crd release name", release.Name)
	}

	target, record := genRecordRelease(*release)
	if !record && statusEqual {
		releaseLog.V(utils.Debug).Info("crd release no changes", "crd release name", release.Name,
			"version", release.Spec.Version)
		return false, nil
	}

	if record {
		delete(release.Annotations, lastStatus)
		bytes, err := json.Marshal(target)
		if err != nil {
			releaseLog.Error(err, "marshal release failed")
			return false, err
		}
		if release.Annotations == nil {
			release.Annotations = make(map[string]string)
		} else {
			delete(release.Annotations, lastStatus)
		}
		if len(bytes) > MaxRecordLen {
			return false, errors.New("crd release spec too long to record")
		}
		release.Annotations[lastStatus] = string(bytes)
	}

	return true, nil
}

//updateCRDReleaseCondition
func updateCRDReleaseCondition(release *v1beta1.CRDRelease, conditionType internal.CRDReleaseConditionType,
	status apiextensions.ConditionStatus) {
	releaseLog.V(utils.Debug).Info("try to update crd release conditions", "crd release name",
		release.Name, "type", conditionType, "status", status)
	for i, c := range release.Status.Conditions {
		if c.Type == conditionType {
			releaseLog.V(utils.Debug).Info("update crd release conditions", "type", conditionType,
				"from", c.Status, "to", status)
			if c.Status != status {
				EventRecorder.Eventf(release, corev1.EventTypeNormal, string(conditionType), string(status))
			}
			release.Status.Conditions[i].Status = status
			return
		}
	}
	releaseLog.V(utils.Debug).Info("add crd release conditions", "crd release name", release.Name,
		"version", release.Spec.Version)
	EventRecorder.Eventf(release, corev1.EventTypeNormal, string(conditionType), string(status))
	release.Status.Conditions = append(release.Status.Conditions,
		internal.CRDReleaseCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: v1.Now()})
	return
}
