# CRDRelease

## Typical CRDRelease File

```$xslt
apiVersion: clm.cloudnativeapp.io/v1beta1
kind: CRDRelease
metadata:
  name: applications.oam-domain.alibabacloud.com    ### crd release name
spec:
  version: 1.0.0                                    ### crd release version
  dependencies:                                     ### crd release dependencies
    - name: applicationconfigurations.core.oam.dev
      version: 1.0.0
      strategy: WaitIfAbsent
  modules:                                          ### crd release modules
    - name: applications-crd
      source:
        name: native-source
        values:
          urls:
            - https://.../applications-crd.yaml
    - name: appmodel-module
      precheck:
        resourceExist:
          - type: CustomResourceDefinition
            name: applicationconfigurations.core.oam.dev
      recover:
        retry: true
      readiness:
        failureThreshold: 3
        periodSeconds: 10
        recoverThreshold: 120
        httpGet:
          path: plugin/module?name=configuration-module&crds=applications.oam-domain.alibabacloud.com
          host: adapter-slb
          port: 80
          scheme: http
      source:
        name: service-source
        values:
          container: |
            command:
            - /manager
            - -metrics-addr
            - 127.0.0.1:8082
            image: xxx:latest
            imagePullPolicy: Always
            volumeMounts:
              - mountPath: /home/admin/
                name: ack-config
                readOnly: true
              - mountPath: /var/log/edas
                name: edas-log
            name: configuration

status:                                             ### crd release status
  conditions:
  - lastTransitionTime: "2021-02-07T14:44:16Z"
    status: "True"
    type: Initialized
  - lastTransitionTime: "2021-02-07T14:44:16Z"
    status: "True"
    type: DependenciesSatisfied
  - lastTransitionTime: "2021-02-07T14:44:43Z"
    status: "True"
    type: ModulesReady
  - lastTransitionTime: "2021-02-07T14:45:03Z"
    status: "True"
    type: Ready
  currentVersion: 1.0.0
  phase: Running
  dependencies:
  - name: applicationconfigurations.core.oam.dev
    phase: Running
    version: 1.0.0
  modules:
  - conditions:
    - lastTransitionTime: "2021-02-07T14:44:43Z"
      status: "True"
      type: Initialized
    - lastTransitionTime: "2021-02-07T14:44:43Z"
      status: "True"
      type: PreChecked
    - lastTransitionTime: "2021-02-07T14:44:43Z"
      status: "True"
      type: SourceReady
    - lastTransitionTime: "2021-02-07T14:44:43Z"
      status: "True"
      type: Ready
    lastState:
      installing:
        startedAt: "2021-02-07T14:44:43Z"
    name: applications-crd
    ready: true
    state:
      running:
        startedAt: "2021-02-07T14:45:03Z"
```

### CRDRelease Dependencies

```$xslt
  dependencies:                                     
    - name: applicationconfigurations.core.oam.dev
      version: 1.0.0
      strategy: WaitIfAbsent  (PullIfAbsent | WaitIfAbsent| ErrIfAbsent)
```
* strategy : Strategy when dependency not found in cluster.
    * PullIfAbsent: Pull dependency from registry when it not found in cluster, error will be throw when pull failed.
    * WaitIfAbsent: Default strategy. CRDRelease will wait until dependency appears.
    * ErrIfAbsent: Throw an error simply.

### CRDRelease Module
```$xslt
    - name: appmodel-module
      conditions:                                       ### module conditions
        resourceExist:
          - type: CustomResourceDefinition
            name: applicationconfigurations.core.oam.dev
      preCheck:                                         ### module precheck
        resourceExist:
          - type: CustomResourceDefinition
            name: applicationconfigurations.core.oam.dev
      recover:                                          ### module recover
        retry: true
      readiness:                                        ### module readiness
        failureThreshold: 3
        periodSeconds: 10
        recoverThreshold: 120
        httpGet:
          path: plugin/module?name=configuration-module&crds=applications.oam-domain.alibabacloud.com
          host: adapter-slb
          port: 80
          scheme: http
      source:                                           ### module source
        name: service-source
```
* conditions: Condition check of the module. When condition check failed, module will not be managed by CLM.
    * ResourceNotExist: All resources should not exist.
    * ResourceExist: All resources should exist.
    * Both ResourceNotExist and ResourceExist should meets.   
    
* preCheck: Check before do crd release installation from source, the installation blocks until check success.
    * ResourceNotExist: All resources should not exist.
    * ResourceExist: All resources should exist.
    * Both ResourceNotExist and ResourceExist should meets. 
 
* source: See `helm-source`, `native-source`, `service-source`

* readiness: Readiness prober after module installs successfully, the probe result will change the status of module.
    * recoverThreshold: The failed probe result threshold of turning a module status from recover to abnormal.
    * successThreshold: The success probe result threshold of turning a module status to running.
    * failureThreshold: The failed probe result threshold of turning a module status to abnormal.
    * periodSeconds: Probe periods.
    * timeoutSeconds: Probe timeOut seconds.
    * httpGet/tcpSocket: Please see pkg/probe/probe.go
    
* recover: Indicates whether and how to do source recovery.
    * retry: Indicates whether retry recover work, request to source to do recovery action.
    
    
### CRDRelease Status

* conditions: Conditions of CRD Release.
    * Initialized: Start handle the crd release by clm.
    * DependenciesSatisfied: All dependencies are satisfied, and begin to install modules.
    * ModulesReady: All modules are ready to work.
    * Ready: It means crd release is ready to work now.
    
* phase: 
    * Running.
    * Abnormal. 
    * Installing.
    
* dependencies: Dependencies status of CRD Release. Phase below:
    * Pulling: Pulling dependency from registry.
    * Waiting: Waiting for dependency to be installed successfully.
    * AbsentError: Error when strategy is ErrorIfAbsent.
    * PullError: Pull dependency from registry error.
    * Running: Only when dependency CRDRelease phase is running.
    * Abnormal: Dependency phase abnormal.
    
* events: Events list of handle CRD Release.    
```
Type    Reason                   Age   From        Message
│   ----    ------                   ----  ----        -------
│   Normal  Initialized              93s   CRDRelease  True
│   Normal  DependenciesSatisfied    93s   CRDRelease  True
│   Normal  Module:Initialized       93s   CRDRelease  module nginx.module condition True
│   Normal  Module:PreChecked        93s   CRDRelease  module nginx.module condition True
│   Normal  nginx.module:Installing  86s   CRDRelease  message: reason:
│   Normal  ModulesReady             86s   CRDRelease  False
│   Normal  nginx.module:Running     86s   CRDRelease
│   Normal  ModulesReady             86s   CRDRelease  True
│   Normal  Ready                    86s   CRDRelease  True

```   

### Manage Existing Resources
Using native source:
* `cd config/sample/import; kubectl apply -f nginx-deployment.yaml` Install nginx deployment.

* `kubectl apply -f native-import-example.yaml` Install CRD Release to K8s cluster.

* `kubectl describe crdrelease test-native-import` Check CRD Release status: Status->Modules->Conditions->Type:Imported.

* `kubectl edit crdrelease test-native-import` Scale deployment replicas to 1 and check deployment status.

Using helm source:

* `helm install bitnginx bitnami/nginx` Install helm charts to K8s cluster.

* `kubectl apply -f helm-import-example.yaml` Install CRD Release to K8s cluster.

* `kubectl describe crdrelease test-helm-import` Check CRD Release status: Status->Modules->Conditions->Type:Imported.

* `kubectl delete crdrelease test-helm-import` Delete CRD Release and check result using `helm ls -A`.


### Upgrade CRD Release

* `cd config/sample/crdrelease; kubectl apply -f native-example.yaml` Install native CRD Release.

* `kubectl edit crdrelease test-native` Scale deployment replicas to 1 and check deployment status.

