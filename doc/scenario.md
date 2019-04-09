**Grammar**:

```
<step>        => <assertStep> | <createStep> | <changeStep> | <deleteStep>
<assertStep>  => "assert" ( <count> [<class>] <object> [<is> <phase>] | api  <version> <kind> [<group>] ) [<within> <duration>]
<createStep>  => "create" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
<changeStep>  => "change" <count> <class> <object> "from" <phase> "to" <phase>
<deleteStep>  => "delete" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
<is>         => "is" | "are"
<count>      => [1-9][0-9]*
<class>      => [A-Za-z0-9\-]+
<object>     => "pod[s]" | "node[s]"
<phase>      => "Pending" | "Running" | "Succeeded" | "Failed" | "Unknown"
<duration>   => time.Duration
```

**Supported steps**:
1. Assert
2. Create
3. Change
4. Delete

***1. Assert***: 
Assert can be used to assert the state of a node, a pod or an API within a specific timeout. For example:
- Node:
    - `"assert 2 small nodes within 5s"`: This would assert that 2 small nodes are available within 5 seconds
- Pod
    - `"assert 2 1-cpu pods are Running within 5s"`: This would assert that 2 pods of class `1-cpu` are Runnning within 5 seconds
- Api: 
    - `"assert api v1 Test example.com within 5s"`: This would assert that the api endpoint for `Group: example.com` `Version: v1` and `Kind: Test` is available within 5 seconds

***2. Create***: 
This step creates the specified resources. For example:
- Node:
    - `"create 1 large node"`: This would create 1 instance of a node of class `large` (definition of the class specified as a `--nodeConfig` to `nptest`)
- Pod:
    - `"create 1 4-cpu pod"`: This would create 1 instance of a pod of class `4-cpu` (definition of the class specified as a `--podConfig` to `nptest`)
- Yaml: 
    - `"create 1 instance of example.yml"`: This creates 1 instance of all the objects specified in the yaml

***3. Change***: 
This step can be used to change the state of a pod or set of pods from one state to another. Example:
- `"change 1 1-cpu pod from Running to Failed"`: Changes the state of 1 pod of class `1-cpu` (definition of the class specified as a `--podConfig` to `nptest`) from `Running` to `Failed`

***4. Delete***: 
This step can be used to delete the specified resource. For example:
- Node:
    - `"delete 1 large node"`: This deletes 1 instance of a node of class `large` (definition of the class specified as a `--nodeConfig` to `nptest`)
- Pod:
    - `"delete 1 4-cpu pod"`: This deletes 1 instance of a pod of class `4-cpu` (definition of the class specified as a `--podConfig` to `nptest`)
- Yaml: 
    - `"delete 1 instance of example.yml"`: This deletes 1 instance of all the objects specified in the yaml


**Note**:
Fore more examples, check all the scenario yamls [here](../examples/simple/).

