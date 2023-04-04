package testutil

import (
	"context"
	"errors"
	"fmt"
	dbaasv1beta1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1beta1"
	"github.com/RHEcosystemAppEng/provider-operator-example/apis/dbaas/v1beta1"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var _ Service = &FakeAPIClient{}

type DBaaSProviderService interface {
	client.Client
	CreateCloudService(ctx context.Context, selector client.ObjectKey) (Service, error)
	DiscoverClusters(ctx context.Context, cloudService Service) ([]dbaasv1beta1.DatabaseService, error)
	CreateCluster(ctx context.Context, cloudService Service, instance *v1beta1.ProviderInstance) (*Cluster, error)
}

type FakeProviderService struct {
	client.Client
}

func (s *FakeProviderService) CreateCloudService(ctx context.Context, selector client.ObjectKey) (Service, error) {
	s.retrieveClientCredential(ctx, selector)
	return NewFakeAPIClient(), nil
}

func (s *FakeProviderService) DiscoverClusters(ctx context.Context, cloudService Service) ([]dbaasv1beta1.DatabaseService, error) {
	clusters, _, err := cloudService.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	var instanceLst []dbaasv1beta1.DatabaseService
	if clusters != nil {
		for i := range clusters.Clusters {
			c := clusters.Clusters[i]
			cur := dbaasv1beta1.DatabaseService{
				ServiceID:   c.Id,
				ServiceName: c.Name,
				ServiceInfo: PopulateInstanceInfo(&c),
			}
			instanceLst = append(instanceLst, cur)
		}
	}
	return instanceLst, nil
}

func (s *FakeProviderService) retrieveClientCredential(ctx context.Context, selector client.ObjectKey) (*credential, error) {
	secret := &v1.Secret{}
	if err := s.Get(ctx, selector, secret); err != nil {
		return nil, err
	}
	cred := &credential{
		CredentialField1: string(secret.Data["CredentialField1"]),
		CredentialField2: string(secret.Data["CredentialField2"]),
	}
	if cred.CredentialField1 == "" {
		return nil, errors.New("failed to retrieve API credential1")
	}
	if cred.CredentialField2 == "" {
		return nil, errors.New("failed to retrieve API credential2")
	}

	return cred, nil
}

func (s *FakeProviderService) CreateCluster(ctx context.Context, cloudService Service, instance *v1beta1.ProviderInstance) (*Cluster, error) {

	cloudProvider := getClusterParameter(instance, dbaasv1beta1.ProvisioningCloudProvider)
	if len(cloudProvider) == 0 {
		cloudProvider = "AWS"
	}
	clusterName := getClusterParameter(instance, dbaasv1beta1.ProvisioningName)
	if len(clusterName) == 0 {
		err := fmt.Errorf("parameter %v is required", dbaasv1beta1.ProvisioningName)
		return nil, err
	}

	clusterDetails := &CreateClusterRequest{
		Name:     clusterName,
		Provider: ApiCloudProvider(cloudProvider),
	}

	cluster, _, err := cloudService.CreateCluster(ctx, clusterDetails)

	return cluster, err
}

func getClusterParameter(providerInstance *v1beta1.ProviderInstance, key dbaasv1beta1.ProvisioningParameterType) string {
	if len(providerInstance.Spec.ProvisioningParameters) == 0 {
		return ""
	}
	if value, ok := providerInstance.Spec.ProvisioningParameters[key]; ok {
		return value
	}
	return ""
}

type Service interface {
	ListClusters(ctx context.Context) (*ListClustersResponse, *http.Response, error)
	CreateCluster(ctx context.Context, createClusterRequest *CreateClusterRequest) (*Cluster, *http.Response, error)
}

// Client manages communication with the provider Cloud API v2022-03-31.
type Client struct {
	cfg *Configuration
}

// Configuration stores the configuration of the API client.
type Configuration struct {
	Host          string            `json:"host,omitempty"`
	Scheme        string            `json:"scheme,omitempty"`
	DefaultHeader map[string]string `json:"defaultHeader,omitempty"`
	UserAgent     string            `json:"userAgent,omitempty"`
	Debug         bool              `json:"debug,omitempty"`
	ServerURL     string
	HTTPClient    *http.Client
	apiKey        string
}

func PopulateInstanceInfo(cluster *Cluster) map[string]string {
	data := map[string]string{
		"numOfRegions":    strconv.Itoa(len(cluster.Regions)),
		"Version":         cluster.Version,
		"operationStatus": string(cluster.OperationStatus),
		"creatorId":       cluster.CreatorId,
		"cloudProvider":   string(cluster.CloudProvider),
		"plan":            string(cluster.Plan),
		"state":           string(cluster.State),
		"createAt":        cluster.CreatedAt.String(),
		"updateAt":        cluster.UpdatedAt.String(),
	}
	for i := range cluster.Regions {
		key := fmt.Sprintf("regions.%v.name", strconv.Itoa(i+1))
		data[key] = cluster.Regions[i].Name
		key = fmt.Sprintf("regions.%v.sqlDns", strconv.Itoa(i+1))
		data[key] = cluster.Regions[i].SqlDns
	}

	return data
}

type CreateClusterRequest struct {
	Name     string           `json:"name"`
	Provider ApiCloudProvider `json:"provider"`
}
