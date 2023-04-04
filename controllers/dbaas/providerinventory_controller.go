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
	"fmt"
	"github.com/RHEcosystemAppEng/provider-operator-example/apis/dbaas/v1beta1"
	"github.com/RHEcosystemAppEng/provider-operator-example/controllers/dbaas/testutil"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

// ProviderInventoryReconciler reconciles a ProviderInventory object
type ProviderInventoryReconciler struct {
	testutil.DBaaSProviderService
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerinventories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerinventories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerinventories/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProviderInventory object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ProviderInventoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "ProviderDBaaSInventory", req.NamespacedName)

	var inventory v1beta1.ProviderInventory
	if err := r.Get(ctx, req.NamespacedName, &inventory); err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			logger.Info("ProviderInventory resource not found, may have been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch ProviderInventory for reconcile")
		return ctrl.Result{}, err
	}

	logger.Info("Creating API client for ProviderInventory cloud to get list of cloud clusters/database")
	secretSelector := client.ObjectKey{
		Namespace: inventory.Namespace,
		Name:      inventory.Spec.CredentialsRef.Name,
	}
	cloudService, err := r.CreateCloudService(ctx, secretSelector)
	if err != nil {
		if errUpdate := r.updateInventoryStatus(ctx, inventory, metav1.ConditionFalse, InputError, string(InputError), logger); errUpdate != nil {
			logger.Error(errUpdate, "Failed to update Inventory status")
		}
		logger.Error(err, "Failed to create CloudClient")
		return ctrl.Result{}, err
	}
	logger.Info("Created CloudClient for provider cloud")
	logger.Info("Discovering clusters from provider cloud")

	instanceLst, err := r.DiscoverClusters(ctx, cloudService)
	if err != nil {
		if errUpdate := r.updateInventoryStatus(ctx, inventory, metav1.ConditionFalse, BackendError, string(BackendError), logger); errUpdate != nil {
			logger.Error(errUpdate, "Failed to update Inventory status")
		}
		logger.Error(err, "Failed to discover Clusters")
		return ctrl.Result{}, err
	}
	logger.Info("Sync Instances of the Inventory")
	inventory.Status.DatabaseServices = instanceLst
	if err := r.updateInventoryStatus(ctx, inventory, metav1.ConditionTrue, InventorySyncOK, string(InventorySyncOK), logger); err != nil {
		logger.Error(err, "Failed to update Inventory status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ProviderInventoryReconciler) updateInventoryStatus(ctx context.Context, inventory v1beta1.ProviderInventory,
	status metav1.ConditionStatus, reason ConditionReason, reasonMsg string, logger logr.Logger) error {

	curCondition := metav1.Condition{
		Type:    inventoryConditionTypeReady,
		Status:  status,
		Reason:  string(reason),
		Message: reasonMsg}

	apimeta.SetStatusCondition(&inventory.Status.Conditions, curCondition)
	if err := r.Status().Update(ctx, &inventory); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("Inventory modified, retry reconciling")
			return nil
		}
		logger.Error(err, fmt.Sprintf("Could not update Inventory status:%v", inventory.Name))
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProviderInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ProviderInventory{}).
		Complete(r)
}
