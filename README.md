# CLM
CLM (CRD Lifecycle Management) is a tool to manage CRDs lifecycle.

## Quick Start
* `make install` Install CRD of CLM.

* `kubectl apply -f clm-server.yaml` Install CLM deployment to K8s cluster.

* `cd config/sample/; kubectl apply -f source` Install sources of CLM.

* `kubectl apply -f crdrelease/native-example.yaml` Install the example crd release of native source.
![crd](images/crd.jpg) 
![nginx](images/nginx-deploy.jpg)

* `kubectl apply -f crdrelease/helm-example.yaml` Install the example crd release of helm source.
![helm](images/helm.jpg)

* Check the crd releases installed.
![crdreleases](images/crdreleases.jpg)
