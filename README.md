# Varnish Purger
To avoid having varnish as stateful component in our cluster, we need a way to purge all varnish instances.

## Installation

```
$ varnish-purger --help
Allows to purge all varnish instances living in a cluster

Usage:
  varnish-purger [flags] namespace port-name label1 label2 label3

Flags:
  -h, --help                  help for varnish-purger
      --kube-config string    Path to kubeconfig file
      --kube-context string   Context to use
      --port string           HTTP port (default "8080")
```

This application has been designed to live inside the cluster, it uses the injected service-account tokens to interact with the API. \ 
Optionally can be executed from outside the cluster, `--kube-config` and `--kube-context` are mandatory on those cases.

## Usage 
A new HTTP server will be bootstrapped in the application that will listen to `8080` (or the port defined via `--port`) and will accept `PURGE` methods.
Path will be proxy to all varnishes.

Example:
```
$ curl -X PURGE 'localhost:8080/*'
```
