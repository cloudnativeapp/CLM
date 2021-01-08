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
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clmv1beta1 "cloudnativeapp/clm/api/v1beta1"
)

// SourceReconciler reconciles a Source object
type SourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const sourceFinalizer = "finalizer.clm.cloudnativeapp.io"

// +kubebuilder:rbac:groups=clm.cloudnativeapp.io,resources=sources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clm.cloudnativeapp.io,resources=sources/status,verbs=get;update;patch

func (r *SourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("source", req.NamespacedName)

	// your logic here
	source := &clmv1beta1.Source{}
	if err := r.Get(ctx, req.NamespacedName, source); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(utils.Info).Info("succeed get source", "name", source.Name, "type", source.Spec.Type)
	if source.GetDeletionTimestamp() != nil {
		log.V(utils.Debug).Info("try to finalize source", "name", source.Name)
		if utils.Contains(source.GetFinalizers(), sourceFinalizer) {
			err := r.finalizeSource(log, source)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	if !utils.Contains(source.GetFinalizers(), sourceFinalizer) {
		if err := r.addFinalizer(log, source); err != nil {
			return reconcile.Result{}, err
		}
	}

	if ok := internal.AddSource(source.Name, source.Spec.Implement); !ok {
		log.V(utils.Warn).Info("source add failed", "name", source.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: CycleDelay * time.Second}, errors.New("source add failed")
	}
	log.V(utils.Debug).Info("source add success", "name", source.Name)

	if !source.Status.Ready {
		source.Status.Ready = true
		if err := r.Update(ctx, source); err != nil {
			log.Error(err, "update source status failed", "name", source.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *SourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clmv1beta1.Source{}).
		Complete(r)
}

func (r *SourceReconciler) finalizeSource(reqLogger logr.Logger, instance *clmv1beta1.Source) error {
	reqLogger.V(utils.Debug).Info("source finalizer", "name", instance.Name)
	ok := internal.DeleteSource(instance.Name)
	if !ok {
		reqLogger.V(utils.Warn).Info("source delete failed")
	}
	instance.SetFinalizers(utils.Remove(instance.GetFinalizers(), sourceFinalizer))
	if err := r.Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "failed to update ecs in finalizer", "name", instance.Name)
		return err
	}
	return nil
}

func (r *SourceReconciler) addFinalizer(reqLogger logr.Logger, instance *clmv1beta1.Source) error {
	finalizers := instance.GetFinalizers()
	finalizers = append(finalizers, sourceFinalizer)
	reqLogger.V(utils.Info).Info("Adding Finalizer for the feature", "feature", instance.Name)
	instance.SetFinalizers(finalizers)
	return nil
}
