package compute

import (
	"context"
	//"fmt"
	//"regexp"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	//"github.com/Azure/go-autorest/autorest/azure"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/iam"
	//"github.com/yingeli/egress-ip-operator/gateway/azpip/network"
	//"github.com/Azure/go-autorest/autorest/to"
	//"honnef.co/go/tools/config"
)

func getVMSSVMsClient() compute.VirtualMachineScaleSetVMsClient {
	//vmClient := compute.NewVirtualMachinesClient(config.SubscriptionID())
	vmsClient := compute.NewVirtualMachineScaleSetVMsClientWithBaseURI(
		config.Environment().ResourceManagerEndpoint, config.SubscriptionID())
	a, _ := iam.GetResourceManagementAuthorizer()
	vmsClient.Authorizer = a
	vmsClient.AddToUserAgent(config.UserAgent())
	return vmsClient
}

// GetVMSSVM gets the specified VMSS VM info
func GetVMSSVM(ctx context.Context, vmssName string, instanceId string) (vmssvm compute.VirtualMachineScaleSetVM, err error) {
	vmsClient := getVMSSVMsClient()
	return vmsClient.Get(ctx, config.GroupName(), vmssName, instanceId, compute.InstanceView)
}

/*
// AssociateVMWithPublicIP attach public IP to VM. If public IP is already attached to VM, it will be detached by removing the ip configutation
func AssociateVMSSVMWithPublicIP(ctx context.Context, vmName string, address string) (privateIPAddress string, err error) {
	ip, err := network.LookupPublicIP(ctx, address)
	if err != nil {
		return privateIPAddress, fmt.Errorf("LookupPublicIP error: %v", err)
	}

	if ip.IPConfiguration != nil {
		err = network.DeleteNicIPConfiguration(ctx, *ip.IPConfiguration.ID)
		if err != nil {
			return privateIPAddress, fmt.Errorf("DeleteNicIPConfiguration error: %v", err)
		}
	}

	const vmssvmPatternText = `(.*)[_](.*)`
	configIDPattern := regexp.MustCompile(vmssvmPatternText)
	match := configIDPattern.FindStringSubmatch(vmName)
	if len(match) != 3 {
		return privateIPAddress, fmt.Errorf("wrong vmName: %s", vmName)
	}
	vmssName := match[1]
	vmIndex := match[2]

	vm, err := GetVMSSVM(ctx, vmssName, vmIndex)
	if err != nil {
		return privateIPAddress, fmt.Errorf("compute.GetVM error: %v", err)
	}

	for _, ni := range *vm.NetworkProfile.NetworkInterfaces {
		resource, e := azure.ParseResourceID(*ni.ID)
		if e != nil {
			return privateIPAddress, fmt.Errorf("azure.ParseResourceID error: %v", e)
		}

		nic, e := network.GetVMSSNic(ctx, vmssName, vmIndex, resource.ResourceName)
		if e != nil {
			return privateIPAddress, fmt.Errorf("network.GetVMSSNic error: %v", e)
		}
		if nic.Primary == nil || *nic.Primary {
			fmt.Printf("network.AssociateNicWithPublicIP: %s %s\n", *nic.Name, *ip.IPAddress)
			addr, err := network.AssociateVMSSNicWithPublicIP(ctx, vmssName, vmIndex, nic, ip)
			if e != nil {
				return privateIPAddress, fmt.Errorf("network.AttachPublicIP error: %v", err)
			}
			return addr, nil
		}
	}
	return privateIPAddress, fmt.Errorf("cannot find primary nic on VM %s", vmName)
}
*/
