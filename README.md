# nodus podens

### _P → Q, P_ ⊢ _Q_

Simulated large clusters for Kubernetes scheduler validation.

![nodus podens](https://user-images.githubusercontent.com/379372/55267148-baaea080-523d-11e9-9c63-fec89ed663a5.png)

## quick start

**Build binaries**

`$ make`

**Start k8s control plane services**

`$ make k8s-up`

**Build a kubeconfig that points to the local cluster**

`$ make kconfig`

**Run a test scenario**

```
$ nptest --scenario=examples/simple/scenario.yml \
  --pods=examples/simple/pods.yml \
  --nodes=examples/simple/nodes.yml
INFO[0000] run scenario                                  name="cpu resource test"
INFO[0000] run step [1 / 25]                             description="assert 0 pods"
INFO[0000] run step [2 / 25]                             description="create 1 large node"
INFO[0000] run step [3 / 25]                             description="assert 1 large node"
INFO[0000] run step [4 / 25]                             description="create 2 small nodes"
INFO[0000] run step [5 / 25]                             description="assert 2 small nodes"
INFO[0000] run step [6 / 25]                             description="create 1 4-cpu pod"
INFO[0000] run step [7 / 25]                             description="assert 1 4-cpu pod is Running within 4s"
INFO[0002] run step [8 / 25]                             description="delete 1 4-cpu pod"
INFO[0002] run step [9 / 25]                             description="create 3 1-cpu pods"
INFO[0002] run step [10 / 25]                            description="assert 3 1-cpu pods are Running within 4s"
INFO[0004] run step [11 / 25]                            description="change 1 1-cpu pod from Running to Succeeded"
INFO[0004] run step [12 / 25]                            description="assert 0 1-cpu pods are Pending"
INFO[0004] run step [13 / 25]                            description="assert 2 1-cpu pods are Running"
INFO[0004] run step [14 / 25]                            description="assert 1 1-cpu pod is Succeeded"
INFO[0004] run step [15 / 25]                            description="assert 0 1-cpu pods are Failed"
INFO[0004] run step [16 / 25]                            description="change 1 1-cpu pod from Running to Failed"
INFO[0004] run step [17 / 25]                            description="assert 0 1-cpu pods are Pending"
INFO[0004] run step [18 / 25]                            description="assert 1 1-cpu pod is Running"
INFO[0005] run step [19 / 25]                            description="assert 1 1-cpu pod is Succeeded"
INFO[0005] run step [20 / 25]                            description="assert 1 1-cpu pod is Failed"
INFO[0005] run step [21 / 25]                            description="change 1 1-cpu pod from Running to Pending"
INFO[0005] run step [22 / 25]                            description="assert 1 1-cpu pod is Pending"
INFO[0006] run step [23 / 25]                            description="assert 0 1-cpu pods are Running"
INFO[0006] run step [24 / 25]                            description="assert 1 1-cpu pod is Succeeded"
INFO[0006] run step [25 / 25]                            description="assert 1 1-cpu pod is Failed"
```

**View test results and session statistics**

`$ open my-test-result.html`

**Tear down k8s control plane**

`make k8s-down`
