package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/m15ch4/nsx-tag-controller/pkg/handlers"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type event struct {
	key       string
	eventType string
}

type Controller struct {
	kubeclientset kubernetes.Interface
	workqueue     workqueue.RateLimitingInterface
	eventHandler  handlers.Handler

	servicesLister corelisters.ServiceLister
	servicesSynced cache.InformerSynced
	informer       cache.SharedIndexInformer
}

func NewController(
	kubeclientset *kubernetes.Clientset,
	serviceInformer coreinformers.ServiceInformer,
	eventHandler handlers.Handler) *Controller {

	controller := &Controller{
		kubeclientset:  kubeclientset,
		servicesLister: serviceInformer.Lister(),
		servicesSynced: serviceInformer.Informer().HasSynced,
		workqueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		informer:       serviceInformer.Informer(),
		eventHandler:   eventHandler,
	}

	klog.Info("Setting up event handlers")

	var event event

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			event.key = key
			event.eventType = "create"
			if err == nil {
				controller.workqueue.Add(event)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newSvc := new.(*corev1.Service)
			oldSvc := old.(*corev1.Service)
			if newSvc.ResourceVersion == oldSvc.ResourceVersion {
				// Periodic resync will send update events for all known Services.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}

			klog.Infof("OldSVC: '%s'\n", oldSvc)
			klog.Infof("NewSVC: '%s'\n", newSvc)

			key, err := cache.MetaNamespaceKeyFunc(old)
			event.key = key
			event.eventType = "update"
			if err == nil {
				controller.workqueue.Add(event)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			event.key = key
			event.eventType = "delete"
			if err == nil {
				controller.workqueue.Add(event)
			}
		},
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.servicesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	//wait.Until(c.runWorker, time.Second, stopCh)

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		//var key string
		//var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		/*
			if key, ok = obj.(string); !ok {
				// As the item in the workqueue is actually invalid, we call
				// Forget here else we'd go into a loop of attempting to
				// process a work item that is invalid.
				c.workqueue.Forget(obj)
				utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
				return nil
			}
		*/
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.processItem(obj.(event)); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(obj)
			return fmt.Errorf("error syncing '%s': %s, requeuing", obj.(event).key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", obj.(event).key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) processItem(e event) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(e.key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", e.key))
		return nil
	}

	if e.eventType == "delete" {
		c.eventHandler.ObjectDeleted(e.key)
	}

	// Get the Foo resource with this namespace/name
	service, err := c.servicesLister.Services(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("Service '%s' in work queue no longer exists", e.key))
			return nil
		}
		return err
	}

	if e.eventType == "create" {
		c.eventHandler.ObjectCreated(service)
	}
	if e.eventType == "update" {
		c.eventHandler.ObjectUpdated(service)
	}

	return nil
}
