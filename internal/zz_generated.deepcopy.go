// +build !ignore_autogenerated

/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package internal

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CRDReleaseCondition) DeepCopyInto(out *CRDReleaseCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CRDReleaseCondition.
func (in *CRDReleaseCondition) DeepCopy() *CRDReleaseCondition {
	if in == nil {
		return nil
	}
	out := new(CRDReleaseCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dependency) DeepCopyInto(out *Dependency) {
	*out = *in
	in.Registry.DeepCopyInto(&out.Registry)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dependency.
func (in *Dependency) DeepCopy() *Dependency {
	if in == nil {
		return nil
	}
	out := new(Dependency)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DependencyStatus) DeepCopyInto(out *DependencyStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DependencyStatus.
func (in *DependencyStatus) DeepCopy() *DependencyStatus {
	if in == nil {
		return nil
	}
	out := new(DependencyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Module) DeepCopyInto(out *Module) {
	*out = *in
	in.Conditions.DeepCopyInto(&out.Conditions)
	in.PreCheck.DeepCopyInto(&out.PreCheck)
	in.Source.DeepCopyInto(&out.Source)
	in.Readiness.DeepCopyInto(&out.Readiness)
	out.Recover = in.Recover
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Module.
func (in *Module) DeepCopy() *Module {
	if in == nil {
		return nil
	}
	out := new(Module)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ModuleCondition) DeepCopyInto(out *ModuleCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ModuleCondition.
func (in *ModuleCondition) DeepCopy() *ModuleCondition {
	if in == nil {
		return nil
	}
	out := new(ModuleCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ModuleState) DeepCopyInto(out *ModuleState) {
	*out = *in
	if in.Installing != nil {
		in, out := &in.Installing, &out.Installing
		*out = new(ModuleStateInternal)
		(*in).DeepCopyInto(*out)
	}
	if in.Running != nil {
		in, out := &in.Running, &out.Running
		*out = new(ModuleStateInternal)
		(*in).DeepCopyInto(*out)
	}
	if in.Recovering != nil {
		in, out := &in.Recovering, &out.Recovering
		*out = new(ModuleStateInternal)
		(*in).DeepCopyInto(*out)
	}
	if in.Abnormal != nil {
		in, out := &in.Abnormal, &out.Abnormal
		*out = new(ModuleStateInternal)
		(*in).DeepCopyInto(*out)
	}
	if in.Terminated != nil {
		in, out := &in.Terminated, &out.Terminated
		*out = new(ModuleStateInternal)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ModuleState.
func (in *ModuleState) DeepCopy() *ModuleState {
	if in == nil {
		return nil
	}
	out := new(ModuleState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ModuleStateInternal) DeepCopyInto(out *ModuleStateInternal) {
	*out = *in
	in.StartedAt.DeepCopyInto(&out.StartedAt)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ModuleStateInternal.
func (in *ModuleStateInternal) DeepCopy() *ModuleStateInternal {
	if in == nil {
		return nil
	}
	out := new(ModuleStateInternal)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ModuleStatus) DeepCopyInto(out *ModuleStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ModuleCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.State != nil {
		in, out := &in.State, &out.State
		*out = new(ModuleState)
		(*in).DeepCopyInto(*out)
	}
	if in.LastState != nil {
		in, out := &in.LastState, &out.LastState
		*out = new(ModuleState)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ModuleStatus.
func (in *ModuleStatus) DeepCopy() *ModuleStatus {
	if in == nil {
		return nil
	}
	out := new(ModuleStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Recover) DeepCopyInto(out *Recover) {
	*out = *in
	out.Recover = in.Recover
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Recover.
func (in *Recover) DeepCopy() *Recover {
	if in == nil {
		return nil
	}
	out := new(Recover)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Registry) DeepCopyInto(out *Registry) {
	*out = *in
	if in.Params != nil {
		in, out := &in.Params, &out.Params
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Registry.
func (in *Registry) DeepCopy() *Registry {
	if in == nil {
		return nil
	}
	out := new(Registry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Source) DeepCopyInto(out *Source) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Source.
func (in *Source) DeepCopy() *Source {
	if in == nil {
		return nil
	}
	out := new(Source)
	in.DeepCopyInto(out)
	return out
}
