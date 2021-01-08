package internal

import (
	"cloudnativeapp/clm/pkg/check/condition"
	"cloudnativeapp/clm/pkg/check/precheck"
	"cloudnativeapp/clm/pkg/plugin"
	"cloudnativeapp/clm/pkg/probe"
	"cloudnativeapp/clm/pkg/recover"
	"cloudnativeapp/clm/pkg/utils"
	"errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Module struct {
	Name string `json:"name,omitempty"`
	// Indicates whether the module should be managed by controller.
	Conditions condition.Condition `json:"conditions,omitempty"`
	// Check before do crd release installation from source, the installation blocks until check success.
	PreCheck condition.Condition `json:"preCheck,omitempty"`
	// The source of module installation.
	Source Source `json:"source,omitempty"`
	// Readiness prober after module installs successfully, the probe result will change the status of module.
	Readiness probe.Probe `json:"readiness,omitempty"`
	// Indicates whether do source recovery.
	Recover Recover `json:"recover,omitempty"`
}

type Recover struct {
	// Indicates whether retry recover work
	Retry bool `json:"retry,omitempty"`
	// Specify the action to recover. Do source recover when omitted.
	Recover recover.Recover `json:"action,omitempty"`
}

type Source struct {
	Name string `json:"name,omitempty"`
	// Values to do installation from source.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Values *runtime.RawExtension `json:"values,omitempty"`
}

type ModuleStatus struct {
	Name string `json:"name"`
	// Indicates whether module install success and ready to work.
	Ready bool `json:"ready"`
	// Conditions during installation.
	Conditions []ModuleCondition `json:"conditions,omitempty"`
	// Recover count.
	RecoverCount int `json:"recoverCount,omitempty"`
	// Current state of the module.
	State *ModuleState `json:"state,omitempty"`
	// Last state of the module.
	LastState *ModuleState `json:"lastState,omitempty"`
}

type ModuleCondition struct {
	Type   ModuleConditionType           `json:"type,omitempty"`
	Status apiextensions.ConditionStatus `json:"status,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime v1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
}

type ModuleConditionType string

const (
	// ModuleInitialized means that module did not pass condition check and not be managed by controller
	ModuleExternal ModuleConditionType = "External"
	// ModuleInitialized means that module passed condition check and to be managed by controller
	ModuleInitialized ModuleConditionType = "Initialized"
	// ModulePreChecked means that module passed preChecked and continue to install
	ModulePreChecked ModuleConditionType = "PreChecked"
	// ModuleSourceReady means that module source is ready to work
	ModuleSourceReady ModuleConditionType = "SourceReady"
	// ModuleReady means that module finished installation
	ModuleReady ModuleConditionType = "Ready"
)

const (
	// Module running phase.
	ModuleRunning string = "Running"
	// Module installing phase, Corresponds to install and upgrade.
	ModuleInstalling string = "Installing"
	// Module recovering phase, try to recover from abnormal phase.
	ModuleRecovering string = "Recovering"
	// Module abnormal phase.
	ModuleAbnormal string = "Abnormal"
	// Module terminated phase, after the module has been uninstalled.
	ModuleTerminated string = "Terminated"
	// Do not care.
	ModuleDontCare string = "DontCare"
	// Module did not pass the preCheck.
	PreCheckWaiting string = "PreCheckWaiting"
)

type ModuleState struct {
	// Module should be managed by controller and start to install itself
	Installing *ModuleStateInternal `json:"installing,omitempty"`
	// ModuleRunning means that module finished installation and passed the readiness probe
	Running *ModuleStateInternal `json:"running,omitempty"`
	// Module readiness probe failed and start to recovery according to specify strategy
	Recovering *ModuleStateInternal `json:"recovering,omitempty"`
	// Module installation failed or readiness probe failed without recovery strategy
	Abnormal *ModuleStateInternal `json:"abnormal,omitempty"`
	// Module deleted success
	Terminated *ModuleStateInternal `json:"terminated,omitempty"`
}

type ModuleStateInternal struct {
	StartedAt v1.Time `json:"startedAt,omitempty"`
	Message   string  `json:"message,omitempty"`
	Reason    string  `json:"reason,omitempty"`
}

var mLog = ctrl.Log.WithName("module")

//ConditionCheck do condition check , when the result is false, module will not be managed by clm.
func (m Module) ConditionCheck() (bool, error) {
	mLog.V(utils.Debug).Info("try to check condition", "module", m.Name)
	if reflect.DeepEqual(m.Conditions, condition.Condition{}) {
		mLog.V(utils.Debug).Info("empty condition setting", "module", m.Name)
		return true, nil
	}

	return m.Conditions.Check(mLog)
}

// CheckStatus: return the action needed.
func (m Module) CheckStatus(last Module, s *ModuleState, external bool,
	updateCondition func(ModuleCondition, string)) (plugin.Action, string, error) {
	mLog.V(utils.Debug).Info("try to check module status", "module", m.Name, "last config",
		last, "external", external)
	if external {
		// recheck in case external state changed
		mLog.V(utils.Debug).Info("try to recheck condition", "module", m.Name)
		if ok, err := m.ConditionCheck(); err != nil {
			return plugin.NeedNothing, ModuleAbnormal, err
		} else if !ok {
			mLog.V(utils.Debug).Info("module external condition established", "module", m.Name)
			return plugin.NeedNothing, ModuleRunning, nil
		} else {
			mLog.V(utils.Warn).Info("module external condition broken, start install module", "name", m.Name)
			updateCondition(
				ModuleCondition{Type: ModuleExternal, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()},
				m.Name)
			s = nil
		}
	}
	updateCondition(
		ModuleCondition{Type: ModuleInitialized, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()}, m.Name)
	// Start a complete process
	if s == nil {
		return m.emptyStateProc(updateCondition)
	} else if s.Abnormal != nil {
		mLog.V(utils.Debug).Info("abnormal module need recover", "module", m.Name)
		return plugin.NeedRecover, ModuleDontCare, nil
	} else if s.Terminated != nil {
		mLog.V(utils.Debug).Info("check terminated module", "module", m.Name)
		return plugin.NeedNothing, ModuleTerminated, nil
	} else {
		mLog.V(utils.Info).Info("try to get module status", "name", m.Name)
		state := getModule(m.Name)
		if len(state) == 0 {
			// maybe system reboot
			state = m.resetMemAfterReboot(s)
		} else {
			mLog.V(utils.Debug).Info("module found in controller memory", "module", m.Name, "state", state)
			if state == ModuleAbnormal {
				return plugin.NeedRecover, ModuleDontCare, nil
			}
			// check last apply config
			if !reflect.DeepEqual(m.Source, last.Source) {
				return plugin.NeedUpgrade, ModuleDontCare, nil
			}
		}

		return plugin.NeedConvert, state, nil
	}
}

func (m Module) resetMemAfterReboot(s *ModuleState) (state string) {
	mLog.V(utils.Debug).Info("can not find module in controller memory, maybe system reboot",
		"module", m.Name)
	ch := make(chan bool)
	addModule(m.Name)
	if s.Installing != nil {
		updateModule(m.Name, ModuleInstalling, ch)
		state = ModuleInstalling
	} else if s.Recovering != nil {
		updateModule(m.Name, ModuleRecovering, ch)
		state = ModuleRecovering
	} else if s.Running != nil {
		updateModule(m.Name, ModuleRunning, ch)
		state = ModuleRunning
	}
	go readinessCheck(ch, m)
	return
}

func (m Module) emptyStateProc(updateCondition func(ModuleCondition, string)) (plugin.Action, string, error) {
	if ok, err := m.preCheck(); err != nil {
		mLog.Error(err, "pre check failed", "name", m.Name)
		return plugin.NeedNothing, ModuleAbnormal, err
	} else if !ok {
		mLog.V(utils.Debug).Info("module pre-check failed", "module", m.Name)
		updateCondition(
			ModuleCondition{Type: ModulePreChecked, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()},
			m.Name)
		return plugin.NeedNothing, PreCheckWaiting, nil
	}
	updateCondition(
		ModuleCondition{Type: ModulePreChecked, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()},
		m.Name)
	if ok := addModule(m.Name); ok {
		mLog.V(utils.Debug).Info("module add success", "module", m.Name)
		return plugin.NeedInstall, ModuleDontCare, nil
	} else {
		mLog.V(utils.Debug).Info("module add failed", "module", m.Name)
		return plugin.NeedNothing, ModuleAbnormal, nil
	}
}

func (m Module) preCheck() (bool, error) {
	mLog.V(utils.Debug).Info("try to do pre-check", "module", m.Name)
	if !reflect.DeepEqual(m.PreCheck, precheck.Precheck{}) {
		return m.PreCheck.Check(mLog)
	}
	mLog.V(utils.Debug).Info("empty pre-check configuration", "module", m.Name)
	return true, nil
}

//Install  do install with source and return module status to be updated.
func (m Module) Install() (interface{}, error) {
	mLog.V(utils.Debug).Info("try to install module", "module", m.Name)
	result := ModuleStatus{}
	result.Name = m.Name
	result.Conditions = append(result.Conditions,
		ModuleCondition{Type: ModuleInitialized, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()})
	result.Conditions = append(result.Conditions,
		ModuleCondition{Type: ModulePreChecked, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()})
	if reflect.DeepEqual(m.Source, Source{}) {
		mLog.V(utils.Warn).Info("source not configured", "module", m.Name, "source", m.Source.Name)
		deleteModule(m.Name)
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleSourceReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		return result, errors.New(utils.ModuleStateAbnormal)
	} else {
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleSourceReady, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()})
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		if err := installFromSource(m.Source, m.Name, ""); err != nil {
			mLog.Error(err, "install from source error", "name", m.Name)
			result.State = GenModuleState(ModuleAbnormal, "install from source failed", err.Error())
			return result, err
		} else {
			// 开启监控协程
			mLog.V(utils.Debug).Info("install from source success", "module", m.Name, "source", m.Source)
			ch := make(chan bool)
			addModule(m.Name)
			updateModule(m.Name, ModuleInstalling, ch)
			go readinessCheck(ch, m)
			result.State = GenModuleState(ModuleInstalling, "", "")
			return result, nil
		}
	}
}

//Uninstall do uninstall with source and return module status to be updated.
func (m Module) Uninstall() (interface{}, error) {
	mLog.V(utils.Debug).Info("try to uninstall module", "module", m.Name)
	deleteModule(m.Name)
	result := ModuleStatus{}
	result.Name = m.Name
	if reflect.DeepEqual(m.Source, Source{}) {
		mLog.V(utils.Warn).Info("source not configured", "module", m.Name, "source", m.Source.Name)
		deleteModule(m.Name)
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleSourceReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		return result, nil
	} else {
		if err := uninstallFromSource(m.Source, m.Name, ""); err != nil {
			mLog.Error(err, "uninstall from source error", "name", m.Name)
			result.Conditions = append(result.Conditions,
				ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
			result.State = GenModuleState(ModuleAbnormal, "uninstall from source failed", err.Error())
			return result, nil
		} else {
			mLog.V(utils.Debug).Info("uninstall from source success", "module", m.Name, "source", m.Source)
			result.Conditions = append(result.Conditions,
				ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
			result.State = GenModuleState(ModuleTerminated, "", "")
			return result, nil
		}
	}
}

//DoRecover recover the module to normal phase, return module status to be updated.
func (m Module) DoRecover() (interface{}, error) {
	mLog.V(utils.Debug).Info("try to recover module", "module", m.Name)
	result := ModuleStatus{}
	result.Name = m.Name
	result.Conditions = append(result.Conditions,
		ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
	s := getModule(m.Name)
	if s == ModuleRecovering {
		// 正在恢复中
		mLog.V(utils.Debug).Info("module already in recover mode", "module", m.Name)
		return ModuleStatus{}, nil
	}

	if len(s) == 0 {
		addModule(m.Name)
	}

	if reflect.DeepEqual(m.Recover, Recover{}) {
		mLog.V(utils.Debug).Info("do source recover", "name", m.Name, "source", m.Source)
		if err := recoverFromSource(m.Source, m.Name, ""); err != nil {
			if !updateModule(m.Name, ModuleAbnormal, nil) {
				return nil, errors.New("update module to recovering failed")
			}
			result.State = GenModuleState(ModuleAbnormal, err.Error(), "update module to recovering failed")
			return result, err
		}
	} else if m.Recover.Retry == false {
		mLog.V(utils.Warn).Info("do not retry to recover", "name", m.Name)
		if !updateModule(m.Name, ModuleAbnormal, nil) {
			return nil, errors.New("update module to recovering failed")
		}
		result.State = GenModuleState(ModuleAbnormal, "strategy is not retry", "")
		return result, nil
	} else if reflect.DeepEqual(m.Recover.Recover, recover.Recover{}) {
		mLog.V(utils.Warn).Info("do source recover", "name", m.Name, "source", m.Source)
		if err := recoverFromSource(m.Source, m.Name, ""); err != nil {
			if !updateModule(m.Name, ModuleAbnormal, nil) {
				return nil, errors.New("update module to recovering failed")
			}
			result.State = GenModuleState(ModuleAbnormal, err.Error(), "update module to recovering failed")
			return result, err
		}
	} else {
		mLog.V(utils.Debug).Info("module recover", "module", m.Name, "recover", m.Recover)
		if err := m.Recover.Recover.DoRecover(mLog); err != nil {
			if !updateModule(m.Name, ModuleAbnormal, nil) {
				return nil, errors.New("update module to recovering failed")
			}
			result.State = GenModuleState(ModuleAbnormal, err.Error(), "update module to recovering failed")
			return result, err
		}
	}

	mLog.V(utils.Debug).Info("recover from source success", "module", m.Name, "source", m.Source)
	ch := make(chan bool)
	updateModule(m.Name, ModuleRecovering, ch)
	go readinessCheck(ch, m)
	result.State = GenModuleState(ModuleRecovering, "", "")
	result.RecoverCount = 1
	return result, nil
}

//DoUpgrade  upgrade the module to different version or configuration.
func (m Module) DoUpgrade() (interface{}, error) {
	mLog.V(utils.Debug).Info("try to upgrade module", "module", m.Name)
	result := ModuleStatus{}
	result.Name = m.Name
	s := getModule(m.Name)
	if s == ModuleRecovering {
		// 正在恢复中
		mLog.V(utils.Debug).Info("module already in recover mode", "module", m.Name)
		return ModuleStatus{}, nil
	}
	if reflect.DeepEqual(m.Source, Source{}) {
		mLog.V(utils.Warn).Info("source not configured", "module", m.Name, "source", m.Source.Name)
		deleteModule(m.Name)
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleSourceReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		return result, errors.New(utils.ModuleStateAbnormal)
	} else {
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleSourceReady, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()})
		result.Conditions = append(result.Conditions,
			ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		if err := upgradeFromSource(m.Source, m.Name, ""); err != nil {
			mLog.Error(err, "upgrade from source error", "name", m.Name)
			result.State = GenModuleState(ModuleAbnormal, "upgrade from source failed", err.Error())
			return result, err
			//return result, errors.New(utils.ModuleStateAbnormal)
		} else {
			mLog.V(utils.Debug).Info("upgrade from source success", "module", m.Name, "source", m.Source)
			ch := make(chan bool)
			updateModule(m.Name, ModuleInstalling, ch)
			go readinessCheck(ch, m)
			result.State = GenModuleState(ModuleInstalling, "", "")
			return result, nil
		}
	}
}

//Attributes  return the module name and version.
func (m Module) Attributes() (name, version string) {
	return m.Name, ""
}

//ConvertStatus  convert the module phase and error to module status.
func (m Module) ConvertStatus(modulePhase string, err error) (interface{}, error) {
	mLog.V(utils.Debug).Info("try to convert module phase to status", "module", m.Name,
		"phase", modulePhase)
	var reason string
	if err != nil {
		reason = err.Error()
	}
	s := ModuleStatus{}
	s.Name = m.Name
	switch modulePhase {
	case PreCheckWaiting:
		mLog.V(utils.Info).Info("module preCheck waiting", "module", m.Name, "reason", reason)
		s.Conditions = append(s.Conditions,
			ModuleCondition{Type: ModulePreChecked, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		return s, errors.New(utils.PreCheckWaiting)
	case ModuleInstalling:
		s.Conditions = append(s.Conditions,
			ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionFalse, LastTransitionTime: v1.Now()})
		s.State = GenModuleState(ModuleInstalling, "", reason)
	case ModuleRunning:
		s.Conditions = append(s.Conditions,
			ModuleCondition{Type: ModuleReady, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()})
		s.State = GenModuleState(ModuleRunning, "", reason)
		s.Ready = true
	case ModuleRecovering:
		s.State = GenModuleState(ModuleRecovering, "", reason)
	case ModuleTerminated:
		s.State = GenModuleState(ModuleTerminated, "", reason)
	case ModuleAbnormal:
		s.State = GenModuleState(ModuleAbnormal, "from convert status", reason)
	default:
		mLog.V(utils.Warn).Info("error module phase when convert to status", "module", m.Name,
			"phase", modulePhase)
		return ModuleStatus{}, nil
	}
	return s, nil
}

//UpdateStatus  Update module status with new status configurations.
func (m ModuleStatus) UpdateStatus(new ModuleStatus) ModuleStatus {
	mLog.V(utils.Debug).Info("try to update module status", "module", m.Name,
		"current", m, "new", new)
	result := ModuleStatus{}
	if len(new.Name) > 0 {
		result.Name = new.Name
	} else {
		result.Name = m.Name
	}
	result.Ready = new.Ready
	if new.State != nil && !moduleStateEqual(new.State, m.State) {
		result.LastState = m.State
		result.State = new.State
	} else {
		result.State = m.State
		result.LastState = m.LastState
	}

	result.RecoverCount = m.RecoverCount + new.RecoverCount

	for _, i := range m.Conditions {
		result.Conditions = append(result.Conditions, i)
	}

	for _, i := range new.Conditions {
		find := false
		for k, j := range result.Conditions {
			if i.Type == j.Type {
				if i.Status != j.Status {
					result.Conditions[k].Status = i.Status
				}
				find = true
			}
		}
		if !find {
			result.Conditions = append(result.Conditions, i)
		}
	}
	return result
}

func moduleStateEqual(new, old *ModuleState) bool {
	if new == nil && old == nil {
		return true
	}
	if new == nil || old == nil {
		return false
	}
	if reflect.DeepEqual(new, old) {
		return true
	}
	if !moduleStateInternalEqual(new.Running, old.Running) {
		return false
	}
	if !moduleStateInternalEqual(new.Installing, old.Installing) {
		return false
	}
	if !moduleStateInternalEqual(new.Terminated, old.Terminated) {
		return false
	}
	if !moduleStateInternalEqual(new.Recovering, old.Recovering) {
		return false
	}
	if !moduleStateInternalEqual(new.Abnormal, old.Abnormal) {
		return false
	}
	return true
}

func moduleStateInternalEqual(new, old *ModuleStateInternal) bool {
	if new == nil && old == nil {
		return true
	}
	if new == nil || old == nil {
		return false
	}
	if reflect.DeepEqual(new, old) {
		return true
	}
	if new.Reason != old.Reason {
		return false
	}
	if new.Message != old.Message {
		return false
	}
	return true
}

//GetConditionStatus  return the condition status of module condition type.
func (m ModuleStatus) GetConditionStatus(t ModuleConditionType) apiextensions.ConditionStatus {
	for _, c := range m.Conditions {
		if c.Type == t {
			return c.Status
		}
	}
	return apiextensions.ConditionFalse
}

//GenerateExternalModuleStatus  generate the external module status.
func GenerateExternalModuleStatus(name string) ModuleStatus {
	status := ModuleStatus{}
	status.Name = name
	status.Ready = true
	status.Conditions = append(status.Conditions,
		ModuleCondition{Type: ModuleExternal, Status: apiextensions.ConditionTrue, LastTransitionTime: v1.Now()})
	return status
}

//GenModuleState  generate all kinds of module state according to phase.
func GenModuleState(phase, message, reason string) *ModuleState {
	switch phase {
	case ModuleRunning:
		return &ModuleState{
			Running: &ModuleStateInternal{
				StartedAt: v1.Now(),
				Message:   message,
				Reason:    reason,
			},
		}
	case ModuleRecovering:
		return &ModuleState{
			Recovering: &ModuleStateInternal{
				StartedAt: v1.Now(),
				Message:   message,
				Reason:    reason,
			},
		}
	case ModuleInstalling:
		return &ModuleState{
			Installing: &ModuleStateInternal{
				StartedAt: v1.Now(),
				Message:   message,
				Reason:    reason,
			},
		}
	case ModuleTerminated:
		return &ModuleState{
			Terminated: &ModuleStateInternal{
				StartedAt: v1.Now(),
				Message:   message,
				Reason:    reason,
			},
		}
	case ModuleAbnormal:
		fallthrough
	default:
		return &ModuleState{
			Abnormal: &ModuleStateInternal{
				StartedAt: v1.Now(),
				Message:   message,
				Reason:    reason,
			},
		}
	}
}
