// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package network

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/iam"
)

func getIPClient() network.PublicIPAddressesClient {
	ipClient := network.NewPublicIPAddressesClientWithBaseURI(
		config.Environment().ResourceManagerEndpoint, config.SubscriptionID())
	auth, _ := iam.GetResourceManagementAuthorizer()
	ipClient.Authorizer = auth
	ipClient.AddToUserAgent(config.UserAgent())
	return ipClient
}

// CreatePublicIP creates a new public IP
func CreatePublicIP(ctx context.Context, ipName string) (ip network.PublicIPAddress, err error) {
	ipClient := getIPClient()
	future, err := ipClient.CreateOrUpdate(
		ctx,
		config.GroupName(),
		ipName,
		network.PublicIPAddress{
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(config.Location()),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)

	if err != nil {
		return ip, fmt.Errorf("cannot create public ip address: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, ipClient.Client)
	if err != nil {
		return ip, fmt.Errorf("cannot get public ip address create or update future response: %v", err)
	}

	return future.Result(ipClient)
}

// GetPublicIP returns an existing public IP
func GetPublicIP(ctx context.Context, ipName string) (network.PublicIPAddress, error) {
	ipClient := getIPClient()
	return ipClient.Get(ctx, config.GroupName(), ipName, "")
}

// DeletePublicIP deletes an existing public IP
func DeletePublicIP(ctx context.Context, ipName string) (result network.PublicIPAddressesDeleteFuture, err error) {
	ipClient := getIPClient()
	return ipClient.Delete(ctx, config.GroupName(), ipName)
}

// ListPublicIPs lists public IPs
func ListPublicIPs(ctx context.Context) (result network.PublicIPAddressListResultPage, err error) {
	ipClient := getIPClient()
	return ipClient.List(ctx, config.GroupName())
}

// LookupPublicIP lookup public IP by address
func LookupPublicIP(ctx context.Context, address string) (ip network.PublicIPAddress, found bool, err error) {
	result, err := ListPublicIPs(ctx)
	if err != nil {
		return ip, false, err
	}
	for result.NotDone() {
		for _, ip := range result.Values() {
			if *ip.IPAddress == address {
				return ip, true, nil
			}
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return ip, false, err
		}
	}
	return ip, false, nil
}

func DissociatePublicIP(ctx context.Context, publicIPAddr string) error {
	pip, found, err := LookupPublicIP(ctx, publicIPAddr)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if pip.IPConfiguration != nil {
		r, err := ParseIPConfigurationID(*pip.IPConfiguration.ID)
		if err != nil {
			return fmt.Errorf("ParseIPConfigurationID error: %v", err)
		}

		nic, err := GetNic(ctx, r.NicName)
		if err != nil {
			return fmt.Errorf("GetNic error: %v", err)
		}

		err = DissociateNicPublicIP(ctx, &nic, *pip.IPConfiguration.ID)
		if err != nil {
			return fmt.Errorf("DissociateNicWithPublicIP error: %v", err)
		}
	}
	return nil
}

/*
// GetPrivateIP detach public IP from VM and remove the ip configutation
func GetPrivateIP(ctx context.Context, pipAddress string) (privateIPAddress string, err error) {
	pip, err := LookupPublicIP(ctx, pipAddress)
	if err != nil {
		return privateIPAddress, fmt.Errorf("LookupPublicIP error: %v", err)
	}

	if pip.IPConfiguration == nil {
		return privateIPAddress, fmt.Errorf("no IPConfiguration for public ip: %s", pipAddress)
	}

	ipconfig, err := GetIPConfiguration(ctx, *pip.IPConfiguration.ID)
	if err != nil {
		return privateIPAddress, err
	}

	if ipconfig.PrivateIPAddress == nil {
		return privateIPAddress, fmt.Errorf("no PrivateIPAddress for public ip: %s", pipAddress)
	}

	return *ipconfig.PrivateIPAddress, nil
}
*/
