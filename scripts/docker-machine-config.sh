set -o nounset errexit pipefail

K8S_URI=http://`docker-machine ip`:8080

cat <<CFG > $PWD/kconfig
apiVersion: v1
clusters:
- cluster:
    server: $K8S_URI
  name: docker-machine
contexts:
- context:
    cluster: docker-machine
  name: docker-machine
current-context: docker-machine
kind: Config
preferences: {}
users: []
CFG
