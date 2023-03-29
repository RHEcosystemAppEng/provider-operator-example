package testutil

import (
	"context"
	"errors"
	"fmt"
	dbaasv1beta1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1beta1"
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
		APIKey: string(secret.Data["APIKey"]),
	}
	if cred.APIKey == "" {
		return nil, errors.New("failed to retrieve API credential")
	}

	return cred, nil
}

type Service interface {
	ListClusters(ctx context.Context) (*ListClustersResponse, *http.Response, error)
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
