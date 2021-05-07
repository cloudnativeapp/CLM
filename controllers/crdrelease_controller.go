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

package controllers

import (
	"cloudnativeapp/clm/internal"
	"cloudnativeapp/clm/pkg/utils"
	"context"
	"errors"
	v1 "k8s.io/api/core/v1"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clmv1beta1 "cloudnativeapp/clm/api/v1beta1"
)

// CRDReleaseReconciler reconciles a CRDRelease object
type CRDReleaseReconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Eventer record.EventRecorder
}

const ReleaseFinalizer = "finalizer.clm.cloudnativeapp.io"
const CycleDelay = 10

// +kubebuilder:rbac:groups=clm.cloudnativeapp.io,resources=crdreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clm.cloudnativeapp.io,resources=crdreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clm.cloudnativeapp.io,resources=events,verbs=get;list;watch;create;update;patch

func (r *CRDReleaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("namespace", req.Namespace,
		"crd release", req.Name)

	// your logic here
	release := &clmv1beta1.CRDRelease{}
	if err := r.Get(ctx, req.NamespacedName, release); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(utils.Info).Info("succeed get release", "name", release.Name, "version", release.Spec.Version)
	log.V(utils.Debug).Info("source values", "value", release.Spec.Modules[0].Source.Values)
	// Record crd release status.
	if !internal.RecordStatus(release.Name, release.Spec.Version, *release.Status.DeepCopy()) {
		// retry later
		releaseLog.V(utils.Info).Info("crd release is processing, retry later.")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}
	defer func() {
		internal.DeleteStatus(release.Name, release.Spec.Version)
		if p := recover(); p != nil {
			log.Error(errors.New("panic occurs"), "panic occurs", "panic:", p)
			r.updateRelease(log, internal.CRDReleaseAbnormal, "panic", release)
			r.Eventer.Eventf(release, v1.EventTypeWarning, "Panic", "panic:%v", p)
		}
	}()

	if release.GetDeletionTimestamp() != nil {
		log.V(utils.Info).Info("crd release is going to be deleted", "name", release.Name,
			"version", release.Spec.Version)
		r.Eventer.Eventf(release, v1.EventTypeNormal, "Deleting", "try to finalize")
		if utils.Contains(release.GetFinalizers(), ReleaseFinalizer) {
			ok, err := r.finalizeRelease(log, release)
			if err != nil {
				r.Eventer.Eventf(release, v1.EventTypeWarning, "Error", "finalize error:%v", err)
				return reconcile.Result{}, err
			}
			if !ok {
				log.V(utils.Warn).Info("finalize crdRelease retry later")
				return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
			}
		}
		return reconcile.Result{}, nil
	}

	if !utils.Contains(release.GetFinalizers(), ReleaseFinalizer) {
		if err := r.addFinalizer(log, release); err != nil {
			return reconcile.Result{}, err
		}
	}

	var reason string
	var crdReleasePhase internal.CRDReleasePhase
	if ok, err := CheckCRDRelease(release); err != nil {
		log.Error(err, "crd release check error", "spec", release.Spec, "status", release.Status)
		r.Eventer.Eventf(release, v1.EventTypeWarning, "Error", "crd release check error:%v", err)
		reason = err.Error()
		// not all error should turn to crd release abnormal
		if abnormalCheck(*release, err) {
			crdReleasePhase = internal.CRDReleaseAbnormal
		} else {
			crdReleasePhase = internal.CRDReleaseInstalling
		}
	} else if !ok {
		log.V(utils.Info).Info("crd release not ready", "name", release.Name,
			"version", release.Spec.Version)
		crdReleasePhase = internal.CRDReleaseInstalling
	} else {
		log.V(utils.Info).Info("crd release is running", "name", release.Name,
			"version", release.Spec.Version)
		crdReleasePhase = internal.CRDReleaseRunning
	}

	if updated, err := r.updateRelease(log, crdReleasePhase, reason, release); err != nil {
		log.Error(err, "updateRelease error")
		r.Eventer.Eventf(release, v1.EventTypeWarning, "Error", "updateRelease error:%v", err)
		// 出现无法控制的故障，requeue也毫无意义
		return ctrl.Result{}, err
	} else if updated {
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}
}

// Think twice before turn crd release phase to abnormal
func abnormalCheck(release clmv1beta1.CRDRelease, err error) bool {
	if err.Error() == utils.ModuleStateAbnormal {
		for _, m := range release.Status.Modules {
			if m.State.Abnormal != nil && m.State.Abnormal.Reason != utils.ImplementNotFound {
				return true
			}
		}
	}
	if err.Error() == utils.DependencyStateAbnormal {
		for _, d := range release.Status.Dependencies {
			if d.Phase == internal.DependencyAbnormal && d.Reason != utils.ImplementNotFound {
				return true
			}
		}
	}
	return false
}

func (r *CRDReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clmv1beta1.CRDRelease{}).
		Complete(r)
}

func (r *CRDReleaseReconciler) finalizeRelease(reqLogger logr.Logger, release *clmv1beta1.CRDRelease) (bool, error) {
	reqLogger.V(utils.Debug).Info("crdRelease finalizer", "name", release.Name)
	ok, err := UninstallCRDRelease(release)
	if err != nil {
		reqLogger.Error(err, "uninstall crd release failed", "name", release.Name)
		r.Update(context.TODO(), release)
		return false, err
	}
	if !ok {
		return false, nil
	}
	release.SetFinalizers(utils.Remove(release.GetFinalizers(), ReleaseFinalizer))
	if err := r.Update(context.TODO(), release); err != nil {
		reqLogger.Error(err, "failed to update crd release in finalizer", "name", release.Name)
		return false, err
	}
	return true, nil
}

func (r *CRDReleaseReconciler) addFinalizer(reqLogger logr.Logger, release *clmv1beta1.CRDRelease) error {
	finalizers := release.GetFinalizers()
	finalizers = append(finalizers, ReleaseFinalizer)
	reqLogger.V(utils.Debug).Info("Adding Finalizer for the feature", "feature", release.Name)

	release.SetFinalizers(finalizers)
	return nil
}

func (r *CRDReleaseReconciler) updateRelease(reqLogger logr.Logger, phase internal.CRDReleasePhase,
	reason string, release *clmv1beta1.CRDRelease) (bool, error) {
	updateCRDReleaseStatus(release, phase, reason)
	if changed, err := updateReleaseCheck(release); err != nil {
		return false, err
	} else if changed {
		if err := r.Update(context.Background(), release); err != nil {
			reqLogger.Error(err, "update crdRelease failed", "name", release.Name, "version", release.Spec.Version,
				"phase", phase, "reason", reason)
			return true, err
		}
		return true, nil
	}
	return false, nil
}
