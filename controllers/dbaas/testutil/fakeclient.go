package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

func NewFakeAPIClient() *FakeAPIClient {
	client := &FakeAPIClient{
		FakeClusters: fakeClusters,
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

type FakeAPIClient struct {
	*FakeClusters
}

type FakeClusters struct {
	addCluster    chan Cluster
	updateCluster chan Cluster
	deleteCluster chan string
	clusters      *ListClustersResponse
	clusterMutex  *sync.Mutex
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

func (f FakeAPIClient) CreateCluster(ctx context.Context, createClusterRequest *CreateClusterRequest) (*Cluster, *http.Response, error) {
	if strings.HasSuffix(createClusterRequest.Name, "invalid-cluster-creation-request") {
		resp := &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(strings.NewReader("{\"code\": 0, \"message\": \"creation failed\"}")),
		}
		return nil, resp, fmt.Errorf("{\"code\": 0, \"message\": \"creation failed\"}")
	}

	f.clusterMutex.Lock()
	clusters := f.clusters
	f.clusterMutex.Unlock()

	clusterID := "a-cluster-instance-id-" + createClusterRequest.Name
	for _, cluster := range clusters.Clusters {
		if cluster.Id == clusterID {
			resp := &http.Response{
				StatusCode: 409,
				Body:       io.NopCloser(strings.NewReader("{\"code\": 6, \"message\": \"code = AlreadyExists\"}")),
			}
			return nil, resp, fmt.Errorf("{\"code\": 6, \"message\": \"code = AlreadyExists\"}")
		}
	}

	cluster := Cluster{
		Id:            clusterID,
		Name:          createClusterRequest.Name,
		CloudProvider: createClusterRequest.Provider,
		Regions: []Region{
			{
				Name:   "region-2",
				SqlDns: "free-tier5.cloud",
			},
		},
		CreatedAt: &aDate,
		UpdatedAt: &aDate,
	}
	f.addCluster <- cluster
	return &cluster, buildFakeResponse(), nil
}

func (f FakeAPIClient) GetCluster(ctx context.Context, clusterID string) (*Cluster, *http.Response, error) {
	f.clusterMutex.Lock()
	clusters := f.clusters
	f.clusterMutex.Unlock()

	for _, cluster := range clusters.Clusters {
		if cluster.Id == clusterID {
			return &cluster, buildFakeResponse(), nil
		}
	}
	return nil, buildFakeResponse(), fmt.Errorf("could not find cluster")
}

// Plan  - DEDICATED: A paid plan that offers dedicated hardware in any location.  - CUSTOM: A plan option that is used for clusters whose machine configs are not  supported in self-service. All INVOICE clusters are under this plan option.  - SERVERLESS: A paid plan that runs on shared hardware and caps the users' maximum monthly spending to a user-specified (possibly 0) amount.
type Plan string

// ApiCloudProvider  - GCP: The Google Cloud Platform cloud provider.  - AWS: The Amazon Web Services cloud provider.
type ApiCloudProvider string

// List of api.CloudProvider.
const (
	APICLOUDPROVIDER_GCP ApiCloudProvider = "GCP"
	APICLOUDPROVIDER_AWS ApiCloudProvider = "AWS"
)

// ClusterStateType  - LOCKED: An exclusive operation is being performed on this cluster. Other operations should not proceed if they did not set a cluster into the LOCKED state.
type ClusterStateType string

// List of ClusterStateType.
const (
	CLUSTERSTATETYPE_CREATED ClusterStateType = "CREATED"
)

// ClusterStatusType the model 'ClusterStatusType'.
type ClusterStatusType string

// Region struct for Region.
type Region struct {
	Name   string `json:"name"`
	SqlDns string `json:"sql_dns"`
	UiDns  string `json:"ui_dns"`
	// NodeCount will be 0 for serverless clusters.
	NodeCount            int32 `json:"node_count"`
	AdditionalProperties map[string]interface{}
}
