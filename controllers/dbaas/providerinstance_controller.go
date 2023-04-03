/*
Copyright 2023.

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

package dbaas

import (
	"context"
	errors1 "errors"
	dbaasv1beta1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1beta1"
	"github.com/RHEcosystemAppEng/provider-operator-example/controllers/dbaas/testutil"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/RHEcosystemAppEng/provider-operator-example/apis/dbaas/v1beta1"
)

// ProviderInstanceReconciler reconciles a ProviderInstance object
type ProviderInstanceReconciler struct {
	testutil.DBaaSProviderService
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProviderInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ProviderInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "ProviderInstance", req.NamespacedName)

	var instance v1beta1.ProviderInstance

	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ProviderInstance resource not found, may have been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch ProviderInstance for reconcile")
		return ctrl.Result{}, err
	}

	instance.Status.Phase = dbaasv1beta1.InstancePhaseUnknown
	inventory := v1beta1.ProviderInventory{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: instance.Spec.InventoryRef.Namespace, Name: instance.Spec.InventoryRef.Name}, &inventory); err != nil {
		if errors.IsNotFound(err) {
			statusErr := r.updateStatus(ctx, &instance, metav1.ConditionFalse, InventoryNotFound, err.Error())
			if statusErr != nil {
				logger.Error(statusErr, "Error in updating instance status")
				return ctrl.Result{Requeue: true}, statusErr
			}
			logger.Info("inventory resource not found, has been deleted")
			return ctrl.Result{}, err

		}
		logger.Error(err, "Failed to fetch ProviderInventory")
		return ctrl.Result{}, err
	}

	instance.Status.Phase = dbaasv1beta1.InstancePhasePending
	logger.Info("Creating API client for provider cloud")
	secretSelector := client.ObjectKey{
		Namespace: inventory.Namespace,
		Name:      inventory.Spec.CredentialsRef.Name,
	}
	cloudService, err := r.CreateCloudService(ctx, secretSelector)
	if err != nil {
		statusErr := r.updateStatus(ctx, &instance, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating instance status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Failed to create CloudClient")
		return ctrl.Result{}, err
	}

	instance.Status.Phase = dbaasv1beta1.InstancePhaseCreating
	logger.Info("Creating  cloud cluster")
	cluster, err := r.CreateCluster(ctx, cloudService, &instance)
	if err != nil {
		statusErr := r.updateStatus(ctx, &instance, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating instance status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Failed to create a cluster at provider cloud")
		return ctrl.Result{}, err
	}
	if err := r.updateClusterDetails(cluster, &instance.Status); err != nil {
		statusErr := r.updateStatus(ctx, &instance, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating instance status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Could not update Instance status")

	}
	logger.Info("updating  cluster details")
	if err := r.Status().Update(ctx, &instance); err != nil {
		if errors.IsConflict(err) {
			logger.Info("Instance modified, retry reconciling")
			return ctrl.Result{Requeue: true}, nil
		}
		instance.Status.Phase = dbaasv1beta1.InstancePhaseReady
		statusErr := r.updateStatus(ctx, &instance, metav1.ConditionTrue, InstanceReady, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating instance status")
			return ctrl.Result{Requeue: true}, statusErr
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProviderInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ProviderInstance{}).
		Complete(r)
}

func (r *ProviderInstanceReconciler) updateStatus(ctx context.Context, conn *v1beta1.ProviderInstance,
	status metav1.ConditionStatus, reason ConditionReason, msg string) error {

	curCondition := metav1.Condition{
		Type:    connectionConditionReadyType,
		Status:  status,
		Reason:  string(reason),
		Message: msg,
	}

	apimeta.SetStatusCondition(&conn.Status.Conditions, curCondition)
	if err := r.Status().Update(ctx, conn); err != nil {
		if errors.IsConflict(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *ProviderInstanceReconciler) updateClusterDetails(clusterDetails *testutil.Cluster,
	instanceStatus *dbaasv1beta1.DBaaSInstanceStatus) error {
	if clusterDetails.Id == "" {
		return errors1.New("received cluster details with no ID")
	}
	instanceStatus.InstanceID = clusterDetails.Id
	instanceStatus.InstanceInfo = testutil.PopulateInstanceInfo(clusterDetails)
	return nil
}
