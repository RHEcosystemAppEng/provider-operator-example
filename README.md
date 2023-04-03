# Provider Operator Example
The GitHub repository provides an operator example for integrating database providers with the [OpenShift Database Access/DBaaS Operator](https://github.com/RHEcosystemAppEng/dbaas-operator). The examples is intended to help developers understand how to create their operator that is integrated with DBaaS operator.

# Create your Operator 

``` 
operator-sdk init --domain redhat.com --repo github.com/RHEcosystemAppEng/provider-operator-example.git

operator-sdk edit --multigroup=true


operator-sdk create api --group dbaas --version v1beta1 --kind ProviderInventory --resource --controller
operator-sdk create api --group dbaas --version v1beta1 --kind ProviderConnection --resource --controller
operator-sdk create api --group dbaas --version v1beta1 --kind ProviderInstance --resource --controller

```

## Import DBaaS operator api in your Operator

Add the DBaaS API to your go.mod file `github.com/RHEcosystemAppEng/dbaas-operator v0.4.1-0.20230403142057-6112a98be1a6` 
and add DBaaSInventorySpec/DBaaSInventoryStatus, DBaaSInstanceSpec/DBaaSInstanceStatus, and DBaaSConnectionSpec/DBaaSConnectionStatus as the Spec/Status to the `ProviderInventory`, `ProviderInstance`, and `ProviderConnection` data structures, respectively.

## Build the Operator 

run the commands to build operator, bundle,catalog and push on quay.io or other registry,  make sure the images in the registry have public access.  
``` 
make docker-build
make docker-push
make bundle
make bundle-build
make bundle-push
make catalog-build
make catalog-push
```

## Add Provider Registration controller and cr details to register with DBaaS Operator

