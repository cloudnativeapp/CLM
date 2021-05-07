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

## Support Helm Repository

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: Source
metadata:
  name: helm-source
spec:
  type: helm
  implement:
    helm:
      wait: true
      timeout: 120
      repositories:
        - name: bitnami                             ### Localname
          url: https://charts.bitnami.com/bitnami   ### Repository url
        - name: nginx-stable
          url: https://helm.nginx.com/stable
        - name: private
          url: https://xxx
          username: yourname                        ### Username of private repository
          password: yourpasswd                      ### Password of private repository
```

## Usage In CRDRelease
```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: CRDRelease
metadata:
  name: test-helm-repo
spec:
  version: 1.0.0
  modules:
    - name: nginx.module
      source:
        name: helm-source
        values:
          chartPath: "bitnami/nginx"
          namespace: default
          releaseName: bitnginx
```