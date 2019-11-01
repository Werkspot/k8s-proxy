# K8s proxy
Bypasses existing ingress controllers to send a request over a list of matching endpoints

## Usage

```
$ k8s-proxy --help
Allows to proxy an HTTP request to all matching endpoints

Usage:
  k8s-proxy [flags]

Flags:
  -h, --help                  help for k8s-proxy
      --http-port string      HTTP port (default "8080")
      --kube-config string    Path to kubeconfig file
      --kube-context string   Context to use
  -l, --label strings         K8s endpoint matching label
      --method strings        HTTP methods allowed
  -n, --namespace strings     K8s namespace
  -p, --port-name strings     K8s endpoint matching port name
      --timeout duration      Proxy timeout (default 5s)
```

Multiple labels, namespaces and port-names can be defined at same time.

This application has been designed to live inside the cluster, it uses the injected service-account tokens to interact 
with the API.
Optionally can be executed from outside the cluster, `--kube-config` and `--kube-context` are mandatory on those cases.

Example:
```
$ k8s-proxy 
    --method PURGE 
    -n my-app-ns 
    -p http 
    -l app.kubernetes.io/name=varnish 
    -l app.kubernetes.io/part-of=my-awesome-application
```
A new HTTP server will be available at port 80, the application will look for all endpoints with label 
`app.kubernetes.io/name=varnish` and `app.kubernetes.io/part-of=my-awesome-application` inside `my-app-ns` namespace
and will forward the request to port `http`.

```
$ curl -X PURGE 'localhost:8080/*'
```

As `PURGE` is a valid method, the request will be forwarded.

Otherwise, an error will be raised.

```
$ curl --head "localhost:8080/*" -X POST
HTTP/1.1 405 Method Not Allowed
Allow: PURGE
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Date: Fri, 01 Nov 2019 11:40:42 GMT
Content-Length: 19
```

### Use cases

Currently is actively used in our platform to purge all varnish instances inside our cluster at once.

## Installation
An specific role must be used in this application.
```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: k8s-proxy
rules:
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["list"]
```

## Configuration

This application can be configured via flags (see usage section) or using environment variables. 
CLI flags will be prioritized if environment variables and flags are defined. 
Environment variables are a composition of `K8S_PROXY` prefix and all available CLI flags, in uppercase and replacing dashes by underscores.

Example:
```
K8S_PROXY_METHOD="PURGE"
K8S_PROXY_NAMESPACE="my-app-ns"
K8S_PROXY_PORT_NAME="http"
K8S_PROXY_LABEL="app.kubernetes.io/name=varnish,app.kubernetes.io/part-of=my-awesome-application"
```
Is equivalent to
```
k8s-proxy 
    --method PURGE 
    -n my-app-ns 
    -p http 
    -l app.kubernetes.io/name=varnish 
    -l app.kubernetes.io/part-of=my-awesome-application
```
