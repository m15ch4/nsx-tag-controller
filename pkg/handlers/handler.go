package handlers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
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
	fmt.Printf("Sync/Add/Update for Service %s\n", obj.(*corev1.Service).GetName())
	fmt.Printf("Service type: %s\n", obj.(*corev1.Service).Spec.Type)
	//fmt.Printf("Service status: %s\n", obj.(*corev1.Service).Status.LoadBalancer.Ingress[0].IP)
	fmt.Printf("Service status: %s\n", obj.(*corev1.Service))

	//fmt.Println("ObjectCreated")
}

func (h *dummy) ObjectUpdated(obj interface{}) {
	fmt.Printf("Update for Service %s\n", obj.(*corev1.Service).GetName())
}

func (h *dummy) ObjectDeleted(key string) {
	fmt.Printf("Serice %s does not exist anymore\n", key)
	//fmt.Println("ObjectDeleted")
}
