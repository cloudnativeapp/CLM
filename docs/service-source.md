# Service Source

Service Source means CLM find a K8s service to do lifecycle management.

## Source Definition

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: Source
metadata:
  name: service-source
spec:
  type: service
  implement:
    localService:
      name: adapter-slb                 ### Find a K8s Service named adapter-slb.
      namespace: edas-oam-system        ### The namespace of K8s Service.
      install:                          ### Install action.
        relativePath: plugin/module     ### Relative path of install action request.
        values:                         ### Parameter of request.
          action:
            - install
        method: post                    ### Method of request.
      uninstall:
        relativePath: plugin/module
        values:
          action:
            - uninstall
        method: delete
      upgrade:
        relativePath: plugin/module
        values:
          action:
            - upgrade
        method: post
      recover:
        relativePath: plugin/module
        values:
          action:
            - recover
        method: post
      status:
        relativePath: plugin/module
        values:
          action:
            - status
        method: get
```

## Usage In CRDRelease

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: CRDRelease
metadata:
  name: test-service
spec:
  version: 1.0.0
  modules:
    - name: service.module
      source:
        name: service-source
      values:    ### Any values can be handle by K8s Service pods
        ...
```

## TODO
