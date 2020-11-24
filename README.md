[![Go Report Card](https://goreportcard.com/badge/github.com/JRBANCEL/ResourceRequests)](https://goreportcard.com/report/github.com/JRBANCEL/ResourceRequests)
[![License](https://img.shields.io/badge/License-BSD%202--Clause-orange.svg)](https://opensource.org/licenses/BSD-2-Clause)

# ResourceRequests
Calculates the total resource requirements (just CPU and memory for now) of a set of Kubernetes YAML files.

# Why?
When releasing a project with many `Pod`, `Deployment` and `Job` objects, it is useful to provide the minimum amount of resource needed for it to be scheduled by Kubernetes.

# How?
The tools go recursively though the provided path and looks at YAML files and sums the resource requests:
* In a `Deployment`, `spec.template.spec.containers[*].resources.requests` is multiplied by `spec.replicas`
* In a `Job`, `spec.template.spec.containers[*].resources.requests` is multiplied by `spec.parallelism

# Install

```
go get -u github.com/JRBANCEL/ResourceRequests/cmd/requests
```

# Example

Let's compute the resource required by Knative:

```
pushd $(mktemp -d)
git clone --depth 1 -b master https://github.com/knative/serving.git
requests serving/config/core 
popd
```

returns

```
-> serving/config/core/deployments/activator.yaml
        Kind: Deployment, Object: knative-serving/activator CPU: 300m, Memory: 63M
-> serving/config/core/deployments/autoscaler.yaml
        Kind: Deployment, Object: knative-serving/autoscaler CPU: 30m, Memory: 42M
-> serving/config/core/deployments/controller.yaml
        Kind: Deployment, Object: knative-serving/controller CPU: 100m, Memory: 105M
-> serving/config/core/deployments/webhook.yaml
        Kind: Deployment, Object: knative-serving/webhook CPU: 100m, Memory: 105M
---
Total - CPU: 530m, Memory: 315M
```

[![asciicast](https://asciinema.org/a/7A9HM38NMmiVPgLnxVfctyGp8.svg)](https://asciinema.org/a/7A9HM38NMmiVPgLnxVfctyGp8)
