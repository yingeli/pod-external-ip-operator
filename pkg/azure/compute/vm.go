package compute

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/iam"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/network"
)

func getVMClient() compute.VirtualMachinesClient {
	//vmClient := compute.NewVirtualMachinesClient(config.SubscriptionID())
	vmClient := compute.NewVirtualMachinesClientWithBaseURI(
		config.Environment().ResourceManagerEndpoint, config.SubscriptionID())
	a, _ := iam.GetResourceManagementAuthorizer()
	vmClient.Authorizer = a
	vmClient.AddToUserAgent(config.UserAgent())
	return vmClient
}

func getVMExtensionsClient() compute.VirtualMachineExtensionsClient {
	extClient := compute.NewVirtualMachineExtensionsClient(config.SubscriptionID())
	a, _ := iam.GetResourceManagementAuthorizer()
	extClient.Authorizer = a
	extClient.AddToUserAgent(config.UserAgent())
	return extClient
}

// CreateVM creates a new virtual machine with the specified name using the specified NIC.
// Username, password, and sshPublicKeyPath determine logon credentials.
func CreateVM(ctx context.Context, vmName, nicName, username, password, sshPublicKeyPath string) (vm compute.VirtualMachine, err error) {
	// see the network samples for how to create and get a NIC resource
	nic, _ := network.GetNic(ctx, nicName)

	var sshKeyData string
	if _, err = os.Stat(sshPublicKeyPath); err == nil {
		sshBytes, err := ioutil.ReadFile(sshPublicKeyPath)
		if err != nil {
			log.Fatalf("failed to read SSH key data: %v", err)
		}
		sshKeyData = string(sshBytes)
	} else {
		sshKeyData = fakepubkey
	}

	vmClient := getVMClient()
	future, err := vmClient.CreateOrUpdate(
		ctx,
		config.GroupName(),
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(config.Location()),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypesBasicA0,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(publisher),
						Offer:     to.StringPtr(offer),
						Sku:       to.StringPtr(sku),
						Version:   to.StringPtr("latest"),
					},
				},
				OsProfile: &compute.OSProfile{
					ComputerName:  to.StringPtr(vmName),
					AdminUsername: to.StringPtr(username),
					AdminPassword: to.StringPtr(password),
					LinuxConfiguration: &compute.LinuxConfiguration{
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									Path: to.StringPtr(
										fmt.Sprintf("/home/%s/.ssh/authorized_keys",
											username)),
									KeyData: to.StringPtr(sshKeyData),
								},
							},
						},
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{
						{
							ID: nic.ID,
							NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
								Primary: to.BoolPtr(true),
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		return vm, fmt.Errorf("cannot create vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return vm, fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	return future.Result(vmClient)
}

// GetVM gets the specified VM info
func GetVM(ctx context.Context, vmName string) (compute.VirtualMachine, error) {
	vmClient := getVMClient()
	return vmClient.Get(ctx, config.GroupName(), vmName, compute.InstanceView)
}

// UpdateVM modifies the VM resource by getting it, updating it locally, and
// putting it back to the server.
func UpdateVM(ctx context.Context, vmName string, tags map[string]*string) (vm compute.VirtualMachine, err error) {

	// get the VM resource
	vm, err = GetVM(ctx, vmName)
	if err != nil {
		return
	}

	// update it
	vm.Tags = tags

	// PUT it back
	vmClient := getVMClient()
	future, err := vmClient.CreateOrUpdate(ctx, config.GroupName(), vmName, vm)
	if err != nil {
		return vm, fmt.Errorf("cannot update vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return vm, fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	return future.Result(vmClient)
}

// DeallocateVM deallocates the selected VM
func DeallocateVM(ctx context.Context, vmName string) (osr autorest.Response, err error) {
	vmClient := getVMClient()
	future, err := vmClient.Deallocate(ctx, config.GroupName(), vmName)
	if err != nil {
		return osr, fmt.Errorf("cannot deallocate vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return osr, fmt.Errorf("cannot get the vm deallocate future response: %v", err)
	}

	return future.Result(vmClient)
}

// StartVM starts the selected VM
func StartVM(ctx context.Context, vmName string) (osr autorest.Response, err error) {
	vmClient := getVMClient()
	future, err := vmClient.Start(ctx, config.GroupName(), vmName)
	if err != nil {
		return osr, fmt.Errorf("cannot start vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return osr, fmt.Errorf("cannot get the vm start future response: %v", err)
	}

	return future.Result(vmClient)
}

// RestartVM restarts the selected VM
func RestartVM(ctx context.Context, vmName string) (osr autorest.Response, err error) {
	vmClient := getVMClient()
	future, err := vmClient.Restart(ctx, config.GroupName(), vmName)
	if err != nil {
		return osr, fmt.Errorf("cannot restart vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return osr, fmt.Errorf("cannot get the vm restart future response: %v", err)
	}

	return future.Result(vmClient)
}

// StopVM stops the selected VM
func StopVM(ctx context.Context, vmName string) (osr autorest.Response, err error) {
	vmClient := getVMClient()
	// skipShutdown parameter is optional, we are taking its default value here
	future, err := vmClient.PowerOff(ctx, config.GroupName(), vmName, nil)
	if err != nil {
		return osr, fmt.Errorf("cannot power off vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return osr, fmt.Errorf("cannot get the vm power off future response: %v", err)
	}

	return future.Result(vmClient)
}

func AssociateVMPrivateIPWithPublicIP(ctx context.Context, vmName string, privateIPAddr string, publicIPAddr string) error {
	pip, found, err := network.LookupPublicIP(ctx, publicIPAddr)
	if err != nil {
		return fmt.Errorf("LookupPublicIP error: %v", err)
	}
	if !found {
		return fmt.Errorf("LookupPublicIP cannot find public ip %s", publicIPAddr)
	}

	vm, err := GetVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("GetVM error: %v", err)
	}

	for _, ni := range *vm.NetworkProfile.NetworkInterfaces {
		resource, err := azure.ParseResourceID(*ni.ID)
		if err != nil {
			return fmt.Errorf("ParseResourceID error: %v", err)
		}

		nic, err := network.GetNic(ctx, resource.ResourceName)
		if err != nil {
			return fmt.Errorf("GetNic error: %v", err)
		}

		if nic.Primary == nil || *nic.Primary {
			return network.AssociateNicPrivateIPWithPublicIP(ctx, nic, privateIPAddr, pip)
		}
	}
	return fmt.Errorf("cannot find primary nic on VM %s", vmName)
}

func DissociateVMPrivateIPWithPublicIP(ctx context.Context, vmName string, privateIPAddr string, publicIPAddr string) error {
	vm, err := GetVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("GetVM error: %v", err)
	}

	for _, ni := range *vm.NetworkProfile.NetworkInterfaces {
		resource, err := azure.ParseResourceID(*ni.ID)
		if err != nil {
			return fmt.Errorf("ParseResourceID error: %v", err)
		}

		nic, err := network.GetNic(ctx, resource.ResourceName)
		if err != nil {
			return fmt.Errorf("GetNic error: %v", err)
		}

		if nic.Primary == nil || *nic.Primary {
			return network.DissociateNicPrivateIPWithPublicIP(ctx, &nic, privateIPAddr, publicIPAddr)
		}
	}
	return fmt.Errorf("cannot find primary nic on VM %s", vmName)
}
