package main

import (
	"flag"

	"github.com/m15ch4/nsx-tag-controller/pkg/handlers"
	"github.com/m15ch4/nsx-tag-controller/pkg/signals"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	masterURL  string
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatal(err)
	}

	factory := informers.NewSharedInformerFactory(kubeClient, 0)

	eventHandler := handlers.NewDummyHandler()

	controller := NewController(kubeClient, factory.Core().V1().Services(), eventHandler)

	factory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatal("Error running controller: %s", err.Error())
	}

}
