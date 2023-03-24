# Provider Operator Example
The GitHub repository provides a operator example for integrating database providers with the OpenShift Database Access/DBaaS Operator. The examples are intended to help developers understand how to create their operator and use the operator to with DBaaS operator.

# Create your Operator 

``` 
operator-sdk init --domain provider.com --repo github.com/RHEcosystemAppEng/provider-operator-example.git

operator-sdk edit --multigroup=true


operator-sdk create api --group dbaas --version v1beta1 --kind ProviderInventory --resource --controller
operator-sdk create api --group dbaas --version v1beta1 --kind ProviderConnection --resource --controller
operator-sdk create api --group dbaas --version v1beta1 --kind ProviderInstance --resource --controller

```

## Import DBaaS operator api in your Operator

In your go.mod and added spec status in `providerconnection_types.go` 
`providerinstance_types.go`, `providerinventory_types.go` from dbaas operaor



## Build the Operator 

run the commands to build operator, bundle,catalog and push on quay.io

make docker-build
make docker-push
make bundle
make bundle-build
make bundle-push
make catalog-build
make catalog-push


## Add Provider Registration controller and cr details to register with DBaaS Operator

