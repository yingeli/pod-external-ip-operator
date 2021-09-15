package azurecni

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/yingeli/pod-external-ip-operator/pkg/azure/compute"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/imds"
)

var (
	log = ctrl.Log.WithName("azurecni")
)

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

func (a *Associater) Associate(ctx context.Context, pod *corev1.Pod, localIP string, publicIP string) (bool, error) {
	if err := compute.AssociateVMPrivateIPWithPublicIP(ctx, pod.Spec.NodeName, localIP, publicIP); err != nil {
		log.Error(err, "error asscociating vm private ip with public ip", "err.Error()", err.Error())
		if isPublicIPReferencedByMultipleIPConfigsError(err) || isPublicIPAddressInUseError(err) {
			return true, nil
		} else {
			return false, err
		}
	}
	if err := AddOrUpdatePodIPRules(pod, localIP); err != nil {
		return false, err
	}
	return false, nil
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

func isPublicIPAddressInUseError(err error) bool {
	/*
		Sample PublicIPAddressInUse error
		"err.Error()": "failed to update nic: network.InterfacesClient#CreateOrUpdate: Failure sending request:
		StatusCode=0 -- Original Error: Code=\"PublicIPAddressInUse\" Message=\"Resource
		/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/MC_rg-aks-cni-standard_aks-cni-standard_eastasia/providers/Microsoft.Network/networkInterfaces/aks-agentpool-93984122-nic-2/ipConfigurations/ipconfig73
		is referencing public IP address /subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/MC_rg-aks-cni-standard_aks-cni-standard_eastasia/providers/Microsoft.Network/publicIPAddresses/pip-externelip-003
		that is already allocated to resource /subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/MC_rg-aks-cni-standard_aks-cni-standard_eastasia/providers/Microsoft.Network/networkInterfaces/aks-agentpool-93984122-nic-0/ipConfigurations/ipconfig5.\"
	*/
	return strings.Contains(err.Error(), "PublicIPAddressInUse")
}

func isPublicIPReferencedByMultipleIPConfigsError(err error) bool {
	return strings.Contains(err.Error(), "PublicIPReferencedByMultipleIPConfigs")
}
