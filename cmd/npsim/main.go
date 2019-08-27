package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"		
	"github.com/IntelAI/nodus/pkg/client"
	"github.com/IntelAI/nodus/pkg/config"
	"github.com/IntelAI/nodus/pkg/node"
)

func main() {
	usage := `npsim - Kubernetes Node Simulator.

Usage:
  npsim --nodes=<config> [--master=<url> | --kubeconfig=<kconfig>]
		[--verbose]
  npsim -h | --help

Options:
  -h --help              Show this screen.
  --nodes=<config>       Nodes config file.
  --master=<url>         Kubernetes API server URL.
  --kubeconfig=<config>  Kubernetes client config file [default: kconfig].
  --verbose              Enable debug logs.`

	args, _ := docopt.ParseDoc(usage)

	verbose, _ := args.Bool("--verbose")
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	nodeConfigPath, _ := args.String("--nodes")
	nodeConfig, err := config.NodeConfigFromFile(nodeConfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to read node config")
		os.Exit(1)
	}
	conf, _ := nodeConfig.AsYaml()
	log.Debugf("using node config:\n%s", conf)

	// Construct apiserver client
	master, _ := args.String("--master")
	kubeconfigPath, _ := args.String("--kubeconfig")
	cl, err := client.NewK8sClient(master, kubeconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct kubernetes client")
		os.Exit(1)
	}

	heartbeat, err := client.NewK8sHeartbeatClient(master, kubeconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct kubernetes client")
		os.Exit(1)
	}

	events, err := client.NewK8sEventClient(master, kubeconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct events client")
		os.Exit(1)
	}
	
	
	// Subscribe to interrupt and terminate signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Creating nodes...")

	nodes := makeNodes(nodeConfig)
	err = start(nodes, cl, heartbeat, events)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to start nodes")
		os.Exit(1)
	}

	defer stop(nodes)

	log.Infof("Registered %d fake nodes", len(nodes))
	log.Info("Waiting for shutdown signal")
	<-shutdown
	fmt.Println("")
	log.Info("Shutting down (deleting fake nodes)")
}

func makeNodes(nodeConfig *config.NodeConfig) []node.FakeNode {
	nodes := []node.FakeNode{}
	for _, class := range nodeConfig.NodeClasses {
		log.WithFields(log.Fields{"class": class.Name}).Debug("making node class")
		for i := uint(0); i < class.Count; i++ {
			log.WithFields(log.Fields{"class": class.Name, "id": i}).Debug("making node")
			name := fmt.Sprintf("%s-%d", class.Name, i)
			n := node.NewFakeNode(name, class.Name, class.Labels, class.Resources)
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func start(nodes []node.FakeNode, client *kubernetes.Clientset, heartbeat *kubernetes.Clientset, events v1core.EventsGetter) (err error) {
	for _, n := range nodes {
		if err = n.Start(client, heartbeat, events); err != nil {
			log.WithFields(log.Fields{
				"node":  n.Name(),
				"error": err.Error(),
			}).Error("failed to start node")
			return
		}
		log.WithFields(log.Fields{
			"node": n.Name(),
		}).Debug("started node")
	}
	return nil
}

func stop(nodes []node.FakeNode) {
	for _, n := range nodes {
		log.WithFields(log.Fields{
			"node": n.Name(),
		}).Debug("stopping node")
		n.Stop()
	}
}
