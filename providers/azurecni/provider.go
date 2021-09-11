package azurecni

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/yingeli/pod-external-ip-operator/pkg/azure/compute"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/imds"
)

const ()

type Associater struct {
	//hostName string
}

func NewAssociater() Associater {
	return Associater{}
}

func (a *Associater) Initialize(ctx context.Context, localNetworks []string) error {
	if err := initializeAzure(); err != nil {
		return err
	}
	//a.hostName = hostName

	if err := SetupIptables(localNetworks); err != nil {
		return err
	}

	return nil
}

func (a *Associater) Associate(ctx context.Context, pod *corev1.Pod, localIP string, publicIP string) error {
	if err := compute.AssociateVMPrivateIPWithPublicIP(ctx, pod.Spec.NodeName, localIP, publicIP); err != nil {
		return err
	}
	if err := AddOrUpdatePodIPRules(pod, localIP); err != nil {
		return err
	}
	return nil
}

func (p *Associater) Dissociate(ctx context.Context, pod *corev1.Pod, localIP string, publicIP string) error {
	if err := RemovePodIPRules(pod); err != nil {
		return err
	}
	return nil
}

type Finalizer struct {
}

func NewFinalizer() Finalizer {
	return Finalizer{}
}

func (p *Finalizer) Initialize(ctx context.Context) error {
	if err := initializeAzure(); err != nil {
		return err
	}

	return nil
}

func (p *Finalizer) Finalize(ctx context.Context, pod *corev1.Pod, localIP string, publicIP string) error {
	return compute.DissociateVMPrivateIPWithPublicIP(ctx, pod.Spec.NodeName, localIP, publicIP)
}

func initializeAzure() (err error) {
	if err := config.ParseEnvironment(); err != nil {
		return fmt.Errorf("config.ParseEnvironment error: %v", err)
	}

	metadata, err := imds.GetMetadata()
	if err != nil {
		return fmt.Errorf("imds.GetMetadata error: %v", err)
	}
	compute := metadata.Compute

	config.SetGroup(compute.AzEnvironment, compute.SubscriptionId, compute.ResourceGroupName)

	return nil
}
