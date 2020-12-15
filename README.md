# (experimental) unofficial [Img](https://github.com/genuinetools/img) CRD controller

Simple CRD that uses [img](https://github.com/genuinetools/img) to build docker images on Kubernetes. It doesn't require `privileged` permissions, and builds the image as user `1000` in the workload pod.

## Install

To install it in your k8s cluster:

```bash
$ kubectl apply -f https://raw.githubusercontent.com/mudler/img-controller/master/hack/kube.yaml
```

## Build images

The controller expose a new `ImageBuild` Kubernetes resource type, which can be used to build docker images with `img`.

To build an image, for example:

```bash

$ cat <<EOF | kubectl apply -f -
apiVersion: img.k8s.io/v1alpha1
kind: ImageBuild
metadata:
  name: test
spec:
  imageName: mocaccino/extra:latest
  repository: 
    url: "https://github.com/mocaccinoOS/mocaccino-extra"
EOF
```


### Full example


```yaml
apiVersion: img.k8s.io/v1alpha1
kind: ImageBuild
metadata:
  name: test
spec:
  annotations:
    # Annotations to apply to workload pod
  labels:
    # Labels to apply to workload pod
  nodeSelector:
    # node Selector labels
  imageName: container/img
  context: "./foo/bar"
  dockerfile: "./foo/bar/Dockerfile"
  privileged: false # Wether to run privileged builds
  registry: # If enabled, will push the docker image
    enabled: true
    username: "user"
    password: "pass"
    registry: "quay.io"
    fromSecret: "secret-key" # Only if using credentials from secret
  repository: 
    url: "https://github.com/mocaccinoOS/mocaccino-extra"
    checkout: "hash_or_branch"

```

If storage and registry credentials are sourced from secrets, the secret should have the following fields and live in the same namespace of the workload:

```yaml
registryUri: ""
registryPassword: ""
registryUsername: ""
```

## Uninstall

First delete all the workload from the cluster, by deleting all the `imagebuild` resources.

Then run:

```bash

$ kubectl delete -f https://raw.githubusercontent.com/mudler/img-controller/master/hack/kube.yaml

```
