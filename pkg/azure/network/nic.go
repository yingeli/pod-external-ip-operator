// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package network

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/iam"
)

func getNicClient() network.InterfacesClient {
	//nicClient := network.NewInterfacesClient(config.SubscriptionID())
	nicClient := network.NewInterfacesClientWithBaseURI(
		config.Environment().ResourceManagerEndpoint, config.SubscriptionID())
	auth, _ := iam.GetResourceManagementAuthorizer()
	nicClient.Authorizer = auth
	nicClient.AddToUserAgent(config.UserAgent())
	return nicClient
}

// CreateNIC creates a new network interface. The Network Security Group is not a required parameter
func CreateNIC(ctx context.Context, vnetName, subnetName, nsgName, ipName, nicName string) (nic network.Interface, err error) {
	subnet, err := GetVirtualNetworkSubnet(ctx, vnetName, subnetName)
	if err != nil {
		log.Fatalf("failed to get subnet: %v", err)
	}

	ip, err := GetPublicIP(ctx, ipName)
	if err != nil {
		log.Fatalf("failed to get ip address: %v", err)
	}

	nicParams := network.Interface{
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(config.Location()),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ipConfig1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet:                    &subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           &ip,
					},
				},
			},
		},
	}

	if nsgName != "" {
		nsg, err := GetNetworkSecurityGroup(ctx, nsgName)
		if err != nil {
			log.Fatalf("failed to get nsg: %v", err)
		}
		nicParams.NetworkSecurityGroup = &nsg
	}

	nicClient := getNicClient()
	future, err := nicClient.CreateOrUpdate(ctx, config.GroupName(), nicName, nicParams)
	if err != nil {
		return nic, fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return nic, fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	return future.Result(nicClient)
}

// CreateNICWithLoadBalancer creats a network interface, wich is set up with a loadbalancer's NAT rule
func CreateNICWithLoadBalancer(ctx context.Context, lbName, vnetName, subnetName, nicName string, natRule int) (nic network.Interface, err error) {
	subnet, err := GetVirtualNetworkSubnet(ctx, vnetName, subnetName)
	if err != nil {
		return
	}

	lb, err := GetLoadBalancer(ctx, lbName)
	if err != nil {
		return
	}

	nicClient := getNicClient()
	future, err := nicClient.CreateOrUpdate(ctx,
		config.GroupName(),
		nicName,
		network.Interface{
			Location: to.StringPtr(config.Location()),
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				IPConfigurations: &[]network.InterfaceIPConfiguration{
					{
						Name: to.StringPtr("pipConfig"),
						InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
							Subnet: &network.Subnet{
								ID: subnet.ID,
							},
							LoadBalancerBackendAddressPools: &[]network.BackendAddressPool{
								{
									ID: (*lb.BackendAddressPools)[0].ID,
								},
							},
							LoadBalancerInboundNatRules: &[]network.InboundNatRule{
								{
									ID: (*lb.InboundNatRules)[natRule].ID,
								},
							},
						},
					},
				},
			},
		})
	if err != nil {
		return nic, fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return nic, fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	return future.Result(nicClient)
}

// GetNic returns an existing network interface
func GetNic(ctx context.Context, nicName string) (network.Interface, error) {
	nicClient := getNicClient()
	return nicClient.Get(ctx, config.GroupName(), nicName, "")
}

// DeleteNic deletes an existing network interface
func DeleteNic(ctx context.Context, nic string) (result network.InterfacesDeleteFuture, err error) {
	nicClient := getNicClient()
	return nicClient.Delete(ctx, config.GroupName(), nic)
}

// AssociateNicPrivateIPWithPublicIP associate public IP to network interface
func AssociateNicPrivateIPWithPublicIP(ctx context.Context, nic network.Interface, privateIPAddr string, ip network.PublicIPAddress) error {
	found := false
	for _, ifconfig := range *nic.IPConfigurations {
		if *ifconfig.PrivateIPAddress == privateIPAddr {
			ifconfig.PublicIPAddress = &ip
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("private ip not found")
	}

	nicClient := getNicClient()

	future, err := nicClient.CreateOrUpdate(ctx, config.GroupName(), *nic.Name, nic)
	if err != nil {
		return fmt.Errorf("failed to update nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get nic update future response: %v", err)
	}

	nic, err = future.Result(nicClient)
	if err != nil {
		return fmt.Errorf("error loading update result: %v", err)
	}

	return nil
}

func DissociateNicPublicIP(ctx context.Context, nic *network.Interface, ipconfigID string) error {
	ipconfigs := nic.IPConfigurations
	l := len(*ipconfigs)
	for i := 0; i < l; i++ {
		ipconfig := (*ipconfigs)[i]
		if *ipconfig.ID == ipconfigID {
			ipconfig.PublicIPAddress = nil
			break
		}
	}

	fmt.Printf("getNicClient\n")

	nicClient := getNicClient()

	future, err := nicClient.CreateOrUpdate(ctx, config.GroupName(), *nic.Name, *nic)
	if err != nil {
		return fmt.Errorf("cannot update nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get nic update future response: %v", err)
	}

	_, err = future.Result(nicClient)
	return err
}

func DissociateNicPrivateIPWithPublicIP(ctx context.Context, nic *network.Interface, privateIPAddr string) error {
	ipconfigs := nic.IPConfigurations
	l := len(*ipconfigs)
	for i := 0; i < l; i++ {
		ifconfig := (*ipconfigs)[i]
		if ifconfig.PrivateIPAddress != nil && *ifconfig.PrivateIPAddress == privateIPAddr {
			ifconfig.PublicIPAddress = nil
			break
		}
	}

	nicClient := getNicClient()

	future, err := nicClient.CreateOrUpdate(ctx, config.GroupName(), *nic.Name, *nic)
	if err != nil {
		return fmt.Errorf("cannot update nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get nic update future response: %v", err)
	}

	_, err = future.Result(nicClient)
	return err
}
