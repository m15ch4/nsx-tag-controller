package handlers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type Handler interface {
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(key string)
}

type dummy struct {
}

func NewDummyHandler() Handler {
	return &dummy{}
}

func (h *dummy) ObjectCreated(obj interface{}) {
	klog.Infof("Sync/Add/Update for Service %s\n", obj.(*corev1.Service).GetName())
	klog.Infof("Service '%s' is of type: %s\n", obj.(*corev1.Service).GetName(), obj.(*corev1.Service).Spec.Type)
	//fmt.Printf("Service status: %s\n", obj.(*corev1.Service).Status.LoadBalancer.Ingress[0].IP)
	//fmt.Printf("Service status: %s\n", obj.(*corev1.Service))

	//fmt.Println("ObjectCreated")
}

func (h *dummy) ObjectUpdated(obj interface{}) {
	klog.Infof("Update for Service %s\n", obj.(*corev1.Service).GetName())
	if obj.(*corev1.Service).Spec.Type == "LoadBalancer" && len(obj.(*corev1.Service).Status.LoadBalancer.Ingress) > 0 {
		klog.Infof("LoadBalancer IP: %s\n", obj.(*corev1.Service).Status.LoadBalancer.Ingress[0].IP)
	}
}

func (h *dummy) ObjectDeleted(key string) {
	klog.Infof("Service %s does not exist anymore\n", key)
}
