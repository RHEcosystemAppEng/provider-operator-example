package testutil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

func NewFakeAPIClient() *FakeAPIClient {
	client := &FakeAPIClient{
		FakeClusters: fakeClusters,
		FakeSQLUsers: fakeUsers,
	}
	return client
}

var (
	aDate, _ = time.Parse(time.RFC3339, "22021-12-14T14:38:46.133610Z")
	cluster1 = Cluster{
		Id:            "a-cluster-instance-1-id",
		Name:          "a-cluster-test-1",
		CloudProvider: APICLOUDPROVIDER_GCP,
		State:         CLUSTERSTATETYPE_CREATED,
		Regions: []Region{
			{
				Name:   "region-1",
				SqlDns: "free-tier4.cloud",
			},
		},
		CreatedAt: &aDate,
		UpdatedAt: &aDate,
	}
	cluster2 = Cluster{
		Id:            "a-cluster-instance-2-id",
		Name:          "a-cluster-test-2",
		CloudProvider: APICLOUDPROVIDER_AWS,
		State:         CLUSTERSTATETYPE_CREATED,
		Regions: []Region{
			{
				Name:   "region-2",
				SqlDns: "free-tier5.cloud",
			},
		},
		CreatedAt: &aDate,
		UpdatedAt: &aDate,
	}
	cluster3 = Cluster{
		Id:            "a-cluster-instance-id-a-provider-test-instance-name",
		Name:          "provider-test-instance-name",
		CloudProvider: APICLOUDPROVIDER_AWS,
		State:         CLUSTERSTATETYPE_CREATED,
		Regions: []Region{
			{
				Name:   "region-3",
				SqlDns: "free-tier6.cloud",
			},
		},
		CreatedAt: &aDate,
		UpdatedAt: &aDate,
	}
)

var fakeClusters = NewFakeClusters()
var fakeUsers = NewFakeUsers()

type FakeAPIClient struct {
	*FakeClusters
	*FakeSQLUsers
}

type FakeClusters struct {
	addCluster    chan Cluster
	updateCluster chan Cluster
	deleteCluster chan string
	clusters      *ListClustersResponse
	clusterMutex  *sync.Mutex
}

type FakeSQLUsers struct {
	addUser    chan SQLUser
	deleteUser chan string
	users      *ListSQLUsersResponse
	userMutex  *sync.Mutex
}

type Cluster struct {
	Id                   string            `json:"id"`
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Plan                 Plan              `json:"plan"`
	CloudProvider        ApiCloudProvider  `json:"cloud_provider"`
	AccountId            *string           `json:"account_id,omitempty"`
	State                ClusterStateType  `json:"state"`
	CreatorId            string            `json:"creator_id"`
	OperationStatus      ClusterStatusType `json:"operation_status"`
	Regions              []Region          `json:"regions"`
	CreatedAt            *time.Time        `json:"created_at,omitempty"`
	UpdatedAt            *time.Time        `json:"updated_at,omitempty"`
	DeletedAt            *time.Time        `json:"deleted_at,omitempty"`
	AdditionalProperties map[string]interface{}
}

// ListClustersResponse struct for ListClustersResponse.
type ListClustersResponse struct {
	Clusters []Cluster `json:"clusters"`
}

func NewFakeClusters() *FakeClusters {
	clusters := &FakeClusters{
		addCluster:    make(chan Cluster, 5),
		deleteCluster: make(chan string, 5),
		updateCluster: make(chan Cluster, 5),
		clusterMutex:  &sync.Mutex{},
		clusters:      &ListClustersResponse{Clusters: []Cluster{cluster1, cluster2, cluster3}}, //fixed
	}

	go func() {
		for c := range clusters.addCluster {
			clusters.clusterMutex.Lock()
			clusters.clusters.Clusters = append(clusters.clusters.Clusters, c)
			clusters.clusterMutex.Unlock()
		}
	}()

	go func() {
		for id := range clusters.deleteCluster {
			clusters.clusterMutex.Lock()
			for i, c := range clusters.clusters.Clusters {
				if c.Id == id {
					clusters.clusters.Clusters[i] = clusters.clusters.Clusters[len(clusters.clusters.Clusters)-1]
					clusters.clusters.Clusters = clusters.clusters.Clusters[:len(clusters.clusters.Clusters)-1]
					break
				}
			}
			clusters.clusterMutex.Unlock()
		}
	}()

	go func() {
		for cluster := range clusters.updateCluster {
			clusters.clusterMutex.Lock()
			for i, c := range clusters.clusters.Clusters {
				if c.Id == cluster.Id {
					clusters.clusters.Clusters[i] = cluster
					break
				}
			}
			clusters.clusterMutex.Unlock()
		}
	}()

	return clusters
}

func NewFakeUsers() *FakeSQLUsers {
	users := &FakeSQLUsers{
		addUser:    make(chan SQLUser, 5),
		deleteUser: make(chan string, 5),
		userMutex:  &sync.Mutex{},
		users:      &ListSQLUsersResponse{},
	}

	go func() {
		for c := range users.addUser {
			users.userMutex.Lock()
			users.users.Users = append(users.users.Users, c)
			users.userMutex.Unlock()
		}
	}()

	go func() {
		for name := range users.deleteUser {
			users.userMutex.Lock()
			for i, c := range users.users.Users {
				if c.Name == name {
					users.users.Users[i] = users.users.Users[len(users.users.Users)-1]
					users.users.Users = users.users.Users[:len(users.users.Users)-1]
					break
				}
			}
			users.userMutex.Unlock()
		}
	}()

	return users
}

type SQLUser struct {
	Name                 string `json:"name"`
	AdditionalProperties map[string]interface{}
}

// ListSQLUsersResponse struct for ListSQLUsersResponse.
type ListSQLUsersResponse struct {
	Users                []SQLUser `json:"users"`
	AdditionalProperties map[string]interface{}
}

func (f FakeAPIClient) ListClusters(ctx context.Context) (*ListClustersResponse, *http.Response, error) {
	f.clusterMutex.Lock()
	clusters := f.clusters
	f.clusterMutex.Unlock()
	return clusters, buildFakeResponse(), nil
}

func buildFakeResponse() *http.Response {
	r := &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		Body:          io.NopCloser(bytes.NewBufferString(" ")),
		ContentLength: int64(len(" ")),
		Header:        make(http.Header),
	}
	//r.Write(bytes.NewBuffer(nil))
	return r
}

// Plan  - DEDICATED: A paid plan that offers dedicated hardware in any location.  - CUSTOM: A plan option that is used for clusters whose machine configs are not  supported in self-service. All INVOICE clusters are under this plan option.  - SERVERLESS: A paid plan that runs on shared hardware and caps the users' maximum monthly spending to a user-specified (possibly 0) amount.
type Plan string

// List of Plan.
const (
	PLAN_PLAN_UNSPECIFIED Plan = "PLAN_UNSPECIFIED"
	PLAN_DEDICATED        Plan = "DEDICATED"
	PLAN_CUSTOM           Plan = "CUSTOM"
	PLAN_SERVERLESS       Plan = "SERVERLESS"
)

// ApiCloudProvider  - GCP: The Google Cloud Platform cloud provider.  - AWS: The Amazon Web Services cloud provider.
type ApiCloudProvider string

// List of api.CloudProvider.
const (
	APICLOUDPROVIDER_CLOUD_PROVIDER_UNSPECIFIED ApiCloudProvider = "CLOUD_PROVIDER_UNSPECIFIED"
	APICLOUDPROVIDER_GCP                        ApiCloudProvider = "GCP"
	APICLOUDPROVIDER_AWS                        ApiCloudProvider = "AWS"
)

// ClusterStateType  - LOCKED: An exclusive operation is being performed on this cluster. Other operations should not proceed if they did not set a cluster into the LOCKED state.
type ClusterStateType string

// List of ClusterStateType.
const (
	CLUSTERSTATETYPE_CLUSTER_STATE_UNSPECIFIED ClusterStateType = "CLUSTER_STATE_UNSPECIFIED"
	CLUSTERSTATETYPE_CREATING                  ClusterStateType = "CREATING"
	CLUSTERSTATETYPE_CREATED                   ClusterStateType = "CREATED"
	CLUSTERSTATETYPE_CREATION_FAILED           ClusterStateType = "CREATION_FAILED"
	CLUSTERSTATETYPE_DELETED                   ClusterStateType = "DELETED"
	CLUSTERSTATETYPE_LOCKED                    ClusterStateType = "LOCKED"
)

// ClusterStatusType the model 'ClusterStatusType'.
type ClusterStatusType string

// List of ClusterStatusType.
const (
	CLUSTERSTATUSTYPE_CLUSTER_STATUS_UNSPECIFIED       ClusterStatusType = "CLUSTER_STATUS_UNSPECIFIED"
	CLUSTERSTATUSTYPE_provider_MAJOR_UPGRADE_RUNNING   ClusterStatusType = "provider_MAJOR_UPGRADE_RUNNING"
	CLUSTERSTATUSTYPE_provider_MAJOR_UPGRADE_FAILED    ClusterStatusType = "provider_MAJOR_UPGRADE_FAILED"
	CLUSTERSTATUSTYPE_provider_MAJOR_ROLLBACK_RUNNING  ClusterStatusType = "provider_MAJOR_ROLLBACK_RUNNING"
	CLUSTERSTATUSTYPE_provider_MAJOR_ROLLBACK_FAILED   ClusterStatusType = "provider_MAJOR_ROLLBACK_FAILED"
	CLUSTERSTATUSTYPE_provider_PATCH_RUNNING           ClusterStatusType = "provider_PATCH_RUNNING"
	CLUSTERSTATUSTYPE_provider_PATCH_FAILED            ClusterStatusType = "provider_PATCH_FAILED"
	CLUSTERSTATUSTYPE_provider_SCALE_RUNNING           ClusterStatusType = "provider_SCALE_RUNNING"
	CLUSTERSTATUSTYPE_provider_SCALE_FAILED            ClusterStatusType = "provider_SCALE_FAILED"
	CLUSTERSTATUSTYPE_MAINTENANCE_RUNNING              ClusterStatusType = "MAINTENANCE_RUNNING"
	CLUSTERSTATUSTYPE_provider_INSTANCE_UPDATE_RUNNING ClusterStatusType = "provider_INSTANCE_UPDATE_RUNNING"
	CLUSTERSTATUSTYPE_provider_INSTANCE_UPDATE_FAILED  ClusterStatusType = "provider_INSTANCE_UPDATE_FAILED"
	CLUSTERSTATUSTYPE_provider_EDIT_CLUSTER_RUNNING    ClusterStatusType = "provider_EDIT_CLUSTER_RUNNING"
	CLUSTERSTATUSTYPE_provider_EDIT_CLUSTER_FAILED     ClusterStatusType = "provider_EDIT_CLUSTER_FAILED"
	CLUSTERSTATUSTYPE_provider_CMEK_OPERATION_RUNNING  ClusterStatusType = "provider_CMEK_OPERATION_RUNNING"
	CLUSTERSTATUSTYPE_provider_CMEK_OPERATION_FAILED   ClusterStatusType = "provider_CMEK_OPERATION_FAILED"
	CLUSTERSTATUSTYPE_TENANT_RESTORE_RUNNING           ClusterStatusType = "TENANT_RESTORE_RUNNING"
	CLUSTERSTATUSTYPE_TENANT_RESTORE_FAILED            ClusterStatusType = "TENANT_RESTORE_FAILED"
)

// Region struct for Region.
type Region struct {
	Name   string `json:"name"`
	SqlDns string `json:"sql_dns"`
	UiDns  string `json:"ui_dns"`
	// NodeCount will be 0 for serverless clusters.
	NodeCount            int32 `json:"node_count"`
	AdditionalProperties map[string]interface{}
}

type region Region
