# rtpengine-manager

NOTE:  this is currently a:
  **NON-FUNCTIONAL WORK IN PROGRESS!**

rtpengine-manager keeps the set of RTPEngine references in kamailio in sync with
the Endpoints of an RTPEngine Service set in Kubernetes.

RTPEngine-manager watches a set of Kubernetes Services (one per rtpengine set in
kamailio), and when changes occur to the set of Endpoints for that Service, it
updates the Kamailio RTPEngine list for the corresponding set using the Kamailio
RPC system.

## Usage

In general, `rtpengine-manager` is meant to run as a container within the same Pod as
the kamailio container.

## Options

Command-line options are available to customize and configure the operation of
`rtpengine-manager`:

- `-b <host:port>`: specifies the address on which kamailio is running its binrpc service.  It defaults to `localhost:9998`.
- `-set <set-number>=[namespace:]<service-name>[:port]`: Specifies an RTPEngine set.  This may be passed multiple times for multiple sets.  Namespace is optional.  If not specified, namespace is `default` or the value of the `POD_NAMESPACE` environment variable.  Port is optional and if not specified, `22222` or the value of the `RTPENGINE_PORT` environment variable is used.  Port may also be a name corresponding the to the name of the port in the corresponding Service to be used for the control protocol of RTPEngine.

For simple systems where the monitored services are in the same namespace as
`rtpengine-manager`, you can set the `POD_NAMESPACE` environment variable to
automatically use the same namespace in which `rtpengine-manager` runs.

## RBAC

`rtpengine-manager` needs to run under a service account with access to the `endpoints` resource
for the namespace(s) in which your RTPEngine services exist.

Example RBAC Role for services in the `sip` namespace:

```yaml

kind: ServiceAccount
apiVersion: v1
metadata:
  name: voip-manager
  namespace: sip

--

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: endpoints-reader
rules:
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "watch", "list"]

--

kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: sip
  name: rtpengine-manager
subjects:
  - kind: ServiceAccount
    name: voip-manager
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: endpoints-reader
```

