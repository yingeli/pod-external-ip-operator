package azurecni

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/yingeli/pod-external-ip-operator/pkg/azure/compute"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/imds"
)

type Associater struct {
	hostName string
}

func NewAssociater() Associater {
	return Associater{}
}

func (p *Associater) Initialize(ctx context.Context, localNetworks []string) error {
	hostName, err := initializeAzure()
	if err != nil {
		return err
	}
	p.hostName = hostName

	if err := SetupIptables(localNetworks); err != nil {
		return err
	}

	return nil
}

func (p *Associater) Associate(ctx context.Context, pod *corev1.Pod, publicIPAddr string) (bool, error) {
	podIP := pod.Status.PodIP
	if podIP == "" {
		return false, nil
	}
	if err := compute.AssociateVMPrivateIPWithPublicIP(ctx, p.hostName, podIP, publicIPAddr); err != nil {
		return false, err
	}
	if err := AddPodIPRules(pod); err != nil {
		return false, err
	}
	return true, nil
}

func (p *Associater) Dissociate(ctx context.Context, pod *corev1.Pod) error {
	return RemovePodIPRules(pod)
}

type Finalizer struct {
}

func NewFinalizer() Finalizer {
	return Finalizer{}
}

func (p *Finalizer) Initialize(ctx context.Context, localNetworks []string) error {
	_, err := initializeAzure()
	if err != nil {
		return err
	}

	return nil
}

func (p *Finalizer) Finalize(ctx context.Context, pod *corev1.Pod) error {
	podIP := pod.Status.PodIP
	nodeName := pod.Spec.NodeName
	return compute.DissociateVMPrivateIPWithPublicIP(ctx, nodeName, podIP)
}

func initializeAzure() (hostName string, err error) {
	if err := config.ParseEnvironment(); err != nil {
		return hostName, fmt.Errorf("config.ParseEnvironment error: %v", err)
	}

	metadata, err := imds.GetMetadata()
	if err != nil {
		return hostName, fmt.Errorf("imds.GetMetadata error: %v", err)
	}
	compute := metadata.Compute

	config.SetGroup(compute.AzEnvironment, compute.SubscriptionId, compute.ResourceGroupName)

	return compute.Name, nil
}
