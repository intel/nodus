# nodus ponens

Simulated large clusters for Kubernetes scheduler validation.

## quick start

**Build binaries**

`make`

**Start k8s control plane services**

`make k8s-up`

**Build a kubeconfig that points to the local cluster**

`make kconfig`

**Bring up a simulated fleet of nodes**

`npsim --nodes=examples/simple/nodes.yml`

**Run a test scenario**

`nptest --scenario=examples/simple/scenario.yml --pods=examples/simple/pods.yml`

**View test results and session statistics**

`open my-test-result.html`

**Tear down k8s control plane**

`make k8s-down`


**Step/Grammar Matrix**

| Step              | Grammar                                                                         | Objects supported |
|-------------------|---------------------------------------------------------------------------------|-------------------|
| Assert            | `"assert" <count> [<class>] <object> [<is> <phase>] [<within> <count> seconds]` | Pod, Node         |
| Create            | `"create" <count> <class> <object>`                                             | Pod, Node, Job    |
| Change            | `"change" <count> <class> <object> "from" <phase> "to" <phase>`                 | Pod, Node         |
| Delete            | `"delete" <count> <class> <object>`                                             | Pod, Node, Job    |


| Predicate         | Syntax                                                        |
|-------------------|---------------------------------------------------------------|
| is                | `"is" | "are"`                                                |
| count             | `[1-9][0-9]`                                                  |
| class             | `[A-Za-z0-9\-]+`                                              |
| object            | `"pod[s]" | "node[s]" | "job[s]"`                             |
| phase             | `"Pending" | "Running" | "Succeeded" | "Failed" | "Unknown"`  |