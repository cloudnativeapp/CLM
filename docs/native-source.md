# Native Source

## Source Definition

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: Source
metadata:
  name: native-source
spec:
  type: native
  implement:
    native:
      ignoreError: false  ### Whether ignore single error when handle multiple resource.
```

## Usage In CRDRelease

```
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: CRDRelease
metadata:
  name: test-native
spec:
  version: 1.0.0
  modules:
    - name: native.module
      source:
        name: native-source
        values:
          urls:
            - https://cloudnativeapp.oss-cn-shenzhen.aliyuncs.com/clm/scaledobjects-crd.yaml
          yaml: |
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: nginx-deployment
              namespace: default
            spec:
              selector:
                matchLabels:
                  app: nginx
              replicas: 2
              template:
                metadata:
                  labels:
                    app: nginx
                spec:
                  containers:
                  - name: nginx
                    image: nginx:1.14.2
                    ports:
                    - containerPort: 80
```
* urls: remote resource file.
* yaml: yaml file to apply directly.

## TODO
