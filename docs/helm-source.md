# Helm Source

## Source Definition

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: Source
metadata:
  name: helm-source
spec:
  type: helm
  implement:
    helm:
      wait: true    ### Whether wait helm action result.
      timeout: 120  ### Timeout for waiting.
```

## Usage In CRDRelease

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: CRDRelease
metadata:
  name: test-helm
spec:
  version: 1.0.0
  modules:
    - name: nginx.module
      source:
        name: helm-source
        values:
          chartPath: "https://cloudnativeapp.oss-cn-shenzhen.aliyuncs.com/clm/nginx-ingress-0.7.1.tgz"  ### Helm package URL.
          namespace: default   ###  The namespace to install helm package.
          releaseName: nginx   ###  Helm release name .
          chartValues: {"imageUrl":"...","imageTag":"1.0.0"}  ### Helm package charts value.
```

## TODO
* Support helm registry setting
