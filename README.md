# `kubectl commatrix` Plugin

The `kubectl commatrix` plugin enhances your Kubernetes CLI experience by
providing an easy-to-use command for generating a detailed and up-to-date
communication matrix. This tool leverages the [commatrix](https://github.com/openshift-kni/commatrix)
project to simplify the process of visualizing and documenting network
communication flows in OpenShift clusters.

---

## Overview

The `kubectl commatrix` plugin integrates the powerful capabilities of the
commatrix library directly into your Kubernetes command-line interface. It
enables users to automatically generate a communication flows matrix for OpenShift
deployments, including both **multi-node** and **single-node OpenShift (SNO)**
clusters. This communication matrix can be used for:

- Understanding and documenting ingress traffic flows.
- Assisting with troubleshooting network communication.
- Generating product documentation for customers.

---

## How It Works

The `kubectl commatrix` plugin uses the commatrix library to analyze the `EndpointSlice`
resource in your cluster. It inspects the following:

- **Host-networked Pods**: Identifies host-networked pods and their ingress flows.
- **NodePort Services**: Collects information about NodePort services.
- **LoadBalancer Services**: Tracks traffic entering the cluster through
LoadBalancer services.

By combining these data sources, the plugin generates a detailed communication matrix
for all ingress traffic in your cluster.

---

## Installation

### Prerequisites

- Kubernetes CLI (`kubectl`) installed and configured to access your cluster.
- Go installed for building the plugin, or download a pre-built binary (if available).

---

## Running

```sh
# assumes you have a working KUBECONFIG
$ go build cmd/kubectl-commatrix.go
# place the built binary somewhere in your PATH
$ cp ./kubectl-commatrix /usr/local/bin

# you can now begin using this plugin as a regular kubectl command:
$ kubectl commatrix generate 
```

---

## Example Output

Once you run the `kubectl commatrix generate` command, the plugin will
generate a communication matrix based on the ingress flows in your
OpenShift cluster. The output will be displayed in a tabular format,
similar to the following:

| Direction | Protocol | Port | Namespace              | Service              |
|-----------|----------|------|------------------------|----------------------|
| Pod       | Container | Node Role | Optional              |
| Ingress   | TCP      | 22   | Host system service    | sshd                 |
| Ingress   | TCP      | 111  | Host system service    | rpcbind              |
