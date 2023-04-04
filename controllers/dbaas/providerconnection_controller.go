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
	"errors"
	"fmt"
	dbaasv1beta1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1beta1"
	"github.com/RHEcosystemAppEng/provider-operator-example/apis/dbaas/v1beta1"
	"github.com/RHEcosystemAppEng/provider-operator-example/controllers/dbaas/testutil"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProviderConnectionReconciler reconciles a ProviderConnection object
type ProviderConnectionReconciler struct {
	testutil.DBaaSProviderService
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerconnections,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerconnections/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=providerconnections/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;delete;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProviderConnection object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ProviderConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "ProviderConnection", req.NamespacedName)

	var connection v1beta1.ProviderConnection

	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			logger.Info("ProviderConnection resource not found, or has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch ProviderConnection for reconcile")
		return ctrl.Result{}, err
	}

	inventory := v1beta1.ProviderInventory{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: connection.Spec.InventoryRef.Namespace, Name: connection.Spec.InventoryRef.Name}, &inventory); err != nil {
		if apierrors.IsNotFound(err) {
			statusErr := r.updateStatus(ctx, &connection, metav1.ConditionFalse, InventoryNotFound, err.Error())
			if statusErr != nil {
				logger.Error(statusErr, "Error in updating connection status")
				return ctrl.Result{Requeue: true}, statusErr
			}
			logger.Info("inventory resource not found, has been deleted")
			return ctrl.Result{}, err

		}
		logger.Error(err, "Failed to fetch ProviderInventory")
		return ctrl.Result{}, err
	}

	logger.Info("Checking if Inventory is with a valid cluster")
	instance, err := getClusterInstance(inventory, connection.Spec.DatabaseServiceID)
	if err != nil {
		statusErr := r.updateStatus(ctx, &connection, metav1.ConditionFalse, ConnectionNotReady, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating connection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Invalid instance info")
		return ctrl.Result{}, err
	}

	logger.Info("Creating API client for provider cloud")
	secretSelector := client.ObjectKey{
		Namespace: inventory.Namespace,
		Name:      inventory.Spec.CredentialsRef.Name,
	}
	cloudService, err := r.CreateCloudService(ctx, secretSelector)
	if err != nil {
		statusErr := r.updateStatus(ctx, &connection, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating connection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Failed to create CloudClient")
		return ctrl.Result{}, err
	}

	logger.Info("Created CloudClient for cloud", "cloudService", cloudService)
	logger.Info("Create or get sql user for Connection", "instance", instance)

	logger.Info("Create or update secret for Connection")
	userSecret, err := r.createOrUpdateSecret(ctx, &connection, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, &connection, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating connection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		return ctrl.Result{}, err
	}

	logger.Info("Create or update config map for Connection")
	dbConfigMap, err := r.createOrUpdateConfigMap(ctx, &connection, instance, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, &connection, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating connection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		return ctrl.Result{}, err
	}

	logger.Info("Updating Connection status")
	connection.Status.CredentialsRef = &corev1.LocalObjectReference{Name: userSecret.Name}
	connection.Status.ConnectionInfoRef = &corev1.LocalObjectReference{Name: dbConfigMap.Name}
	statusErr := r.updateStatus(ctx, &connection, metav1.ConditionTrue, ConnectionReady, SuccessConnection)
	if statusErr != nil {
		logger.Error(statusErr, "Error in updating connection status")
		return ctrl.Result{Requeue: true}, statusErr
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProviderConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ProviderConnection{}).
		Complete(r)
}

func getClusterInstance(inventory v1beta1.ProviderInventory, instanceID string) (*dbaasv1beta1.DatabaseService, error) {
	var conSynced *metav1.Condition
	for i := range inventory.Status.Conditions {
		if inventory.Status.Conditions[i].Type == inventoryConditionTypeReady &&
			inventory.Status.Conditions[i].Status == metav1.ConditionTrue {
			conSynced = &inventory.Status.Conditions[i]
			break
		}
	}
	if conSynced == nil {
		return nil, errors.New("ProviderInventory is not yet in-synced, or is invalid")
	}
	for _, databaseService := range inventory.Status.DatabaseServices {
		if databaseService.ServiceID == instanceID {
			return &databaseService, nil
		}
	}
	return nil, fmt.Errorf("instance with id:%v not found in ProviderInventory", instanceID)
}

func (r *ProviderConnectionReconciler) createOrUpdateSecret(ctx context.Context, connection *v1beta1.ProviderConnection, logger logr.Logger) (*corev1.Secret, error) {

	secretName := fmt.Sprintf("cloud-user-credentials-%s", connection.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: connection.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.DBaaSProviderService, secret, func() error {
		secret.Type = corev1.SecretTypeOpaque
		secret.ObjectMeta.Labels = buildLabels(connection)
		secret.ObjectMeta.Labels[dbaasv1beta1.TypeLabelKey] = dbaasv1beta1.TypeLabelValue
		if err := ctrl.SetControllerReference(connection, secret, r.Scheme); err != nil {
			return err
		}
		secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		setSecret(secret)
		return nil
	})
	if err != nil {
		logger.Error(err, "Failed to create or update secret object for the sql user")
		return nil, err
	}
	return secret, nil
}

func buildLabels(connection *v1beta1.ProviderConnection) map[string]string {
	return map[string]string{
		"managed-by":      "dbaas-operator",
		"owner":           connection.Name,
		"owner.kind":      connection.Kind,
		"owner.namespace": connection.Namespace,
	}
}

func setSecret(secret *corev1.Secret) {
	data := map[string][]byte{
		"username": []byte("username1"),
		"password": []byte("Password"),
	}
	secret.Data = data
}

func (r *ProviderConnectionReconciler) createOrUpdateConfigMap(ctx context.Context, connection *v1beta1.ProviderConnection,
	instance *dbaasv1beta1.DatabaseService, logger logr.Logger) (*corev1.ConfigMap, error) {
	logger.Info("Saving this instance's connection info in a configMap")

	cmName := fmt.Sprintf("cloud-conn-cm-%s", connection.Name)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: connection.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.DBaaSProviderService, cm, func() error {
		cm.ObjectMeta.Labels = buildLabels(connection)
		if err := ctrl.SetControllerReference(connection, cm, r.Scheme); err != nil {
			return err
		}
		cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		setConfigMap(cm, instance)
		return nil
	})
	if err != nil {
		logger.Error(err, "Failed to create or update configmap object for the cluster")
		return nil, err
	}
	return cm, nil
}

func setConfigMap(cm *corev1.ConfigMap, instance *dbaasv1beta1.DatabaseService) {
	dataMap := map[string]string{
		"type":     databaseType,
		"provider": databaseProvider,
		"host":     databasehost,
		"port":     databasePort,
		"database": databaseName,
		"sslmode":  databaseSSLMode,
	}
	cm.Data = dataMap
}

func (r *ProviderConnectionReconciler) updateStatus(ctx context.Context, conn *v1beta1.ProviderConnection,
	status metav1.ConditionStatus, reason ConditionReason, msg string) error {

	curCondition := metav1.Condition{
		Type:    connectionConditionReadyType,
		Status:  status,
		Reason:  string(reason),
		Message: msg,
	}

	apimeta.SetStatusCondition(&conn.Status.Conditions, curCondition)
	if err := r.Status().Update(ctx, conn); err != nil {
		if apierrors.IsConflict(err) {
			return nil
		}
		return err
	}
	return nil
}
