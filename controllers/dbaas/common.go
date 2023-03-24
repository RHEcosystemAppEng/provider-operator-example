package dbaas

import (
	"os"
	"strconv"
	"time"
)

const (
	// DefaultRetryDelay applies to situations where we want to wait for certain time for some resources to be available
	DefaultRetryDelay      = time.Second * 5
	DefaultSyncPeriod      = time.Minute * 180
	InstallNamespaceEnvVar = "INSTALL_NAMESPACE"
	instanceFinalizer      = "providerdbaasinstance.dbaas.redhat.com/cluster"

	databaseType     = "postgresql"
	databaseProvider = "provider Cloud"
	databasePort     = "26257"
	databaseName     = "defaultdb"
	databaseSSLMode  = "verify-full"

	inventoryConditionTypeReady  string = "SpecSynced"
	connectionConditionReadyType string = "ReadyForBinding"
	instanceConditionReadyType   string = "ProvisionReady"
	providerConditionReadyType   string = "ProviderReady"

	InstanceCreating          ConditionReason = "Creating"
	InstanceCreationFailed    ConditionReason = "CreationFailed"
	InstanceReady             ConditionReason = "Ready"
	InstanceDeleted           ConditionReason = "Deleted"
	InventorySyncOK           ConditionReason = "SyncOK"
	InventoryNotFound         ConditionReason = "InventoryNotFound"
	ConnectionReady           ConditionReason = "Ready"
	ConnectionNotReady        ConditionReason = "ConnectionNotReady"
	ProviderReady             ConditionReason = "Ready"
	ProviderProcessingPending ConditionReason = "ProcessingPending"
)

// GetSyncPeriod get the sync period for next reconciliation
func GetSyncPeriod() time.Duration {
	if sp, ok := os.LookupEnv("SYNC_PERIOD_MIN"); ok {
		spInt, err := strconv.Atoi(sp)
		if err == nil {
			return time.Duration(spInt) * time.Minute
		}
	}
	return DefaultSyncPeriod
}
