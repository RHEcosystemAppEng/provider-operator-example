# Provider Operator Example
The GitHub repository provides an operator example for integrating database providers with the [OpenShift Database Access/DBaaS Operator](https://github.com/RHEcosystemAppEng/dbaas-operator). The examples is intended to help developers understand how to create their operator that is integrated with DBaaS operator.

## Create Your Operator

Reqs:

- go v1.18
- operator-sdk-v1.24

``` 
operator-sdk init --domain redhat.com --repo github.com/RHEcosystemAppEng/provider-operator-example.git

operator-sdk edit --multigroup=true


operator-sdk create api --group dbaas --version v1beta1 --kind ProviderInventory --resource --controller
operator-sdk create api --group dbaas --version v1beta1 --kind ProviderConnection --resource --controller
operator-sdk create api --group dbaas --version v1beta1 --kind ProviderInstance --resource --controller

```

## Import [DBaaS Operator API](https://github.com/RHEcosystemAppEng/dbaas-operator/tree/main/api/v1beta1) in Your Operator

Add the DBaaS API to your go.mod file `github.com/RHEcosystemAppEng/dbaas-operator v0.4.1-0.20230403142057-6112a98be1a6` 
and add DBaaSInventorySpec/DBaaSInventoryStatus, DBaaSInstanceSpec/DBaaSInstanceStatus, and DBaaSConnectionSpec/DBaaSConnectionStatus as the Spec/Status to the [ProviderInventory](https://github.com/RHEcosystemAppEng/provider-operator-example/blob/main/apis/dbaas/v1beta1/providerinventory_types.go#L35-L36), [ProviderInstance](https://github.com/RHEcosystemAppEng/provider-operator-example/blob/main/apis/dbaas/v1beta1/providerinstance_types.go#L50-L51), and [ProviderConnection](https://github.com/RHEcosystemAppEng/provider-operator-example/blob/main/apis/dbaas/v1beta1/providerconnection_types.go#L37-L38) data structures, respectively.

## Build Your Operator 

Run the following commands to build operator, bundle, catalog and push on quay.io or another registry, and make sure the images in the registry have public access:

``` 
make docker-build
make docker-push
make bundle
make bundle-build
make bundle-push
make catalog-build
make catalog-push
```

## Implement Your Operator Controllers

Please read the DBaaS Provider guide for more details [here](https://github.com/RHEcosystemAppEng/dbaas-operator/tree/main/docs/provider-guide/dbaas-provider-guide.md) and you can follow this operator controller implementation for your reference, 


1- Add [Provider Controller](https://github.com/RHEcosystemAppEng/provider-operator-example/blob/main/controllers/dbaas/dbaasprovider_reconciler.go) and CR details to register with DBaaS Operator.
    This will be a [new controller](https://github.com/RHEcosystemAppEng/provider-operator-example/blob/main/main.go#L98-L113) you can follow or copied the same controller, and update the [registration CR](https://github.com/RHEcosystemAppEng/provider-operator-example/blob/main/controllers/dbaas/dbaasprovider_reconciler.go#L229-L680) accordingly your provider details. 

2- Inventory Controller [Implementation reference](controllers/dbaas/providerinventory_controller.go)

3- Connection Controller [Implementation reference](controllers/dbaas/providerconnection_controller.go)

4- Instance Controller [Implementation reference](controllers/dbaas/providerinstance_controller.go) 

## Test Your Operator
Read these reference docs to understand the flow of DBaaS Operator:

- [Quick Start Guide](https://github.com/RHEcosystemAppEng/dbaas-operator/tree/main/docs/quick-start-guide)
- [Reference Guide](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/docs/reference-guide/main.adoc) 

**Test With DBaaS Operator**

Before testing your operator, make sure to deploy the DBaaS Operator from OLM. Once the DBaaS Operator is installed, you can proceed to install your own operator.

- Verify DBaaS Registration CR: once the operator deployed it will automatically create a cluster level DBaaSProvider custom resource (CR) object and register itself with the DBaaS Operator.
- Create the Provider Account: using DBaaS UI as explained [here](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/docs/quick-start-guide/main.adoc#accessing-the-database-access-menu-for-configuring-and-monitoring)
- Create new Instance: you can create the new Instance by going DBaaS UI by clicking Create Database Instance
- Create the Connection with Instance : using DBaaS UI as explained [here](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/docs/quick-start-guide/main.adoc#accessing-the-developer-workspace-and-adding-a-database-instance) 
- Create your provider application sample: You need to create the application according to [Service binding libraries](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/docs/reference-guide/main.adoc#service-binding-libraries) structure : 

**Test Standalone Operator**

- Create the API Secret 

 ```kubectl create secret generic dbaas-vendor-credentials  --from-literal="CredentialField1=<Application ID>"   --from-literal="CredentialField2=<Application Secret>"   -n openshift-dbaas-operator```

- Create the Provider Account like [here](config/samples/dbaas_v1beta1_providerinventory.yaml) 
- Create connection Object like [here](config/samples/dbaas_v1beta1_providerconnection.yam)
- Create the Instance Object like [here](config/samples/dbaas_v1beta1_providerinstance.yaml)