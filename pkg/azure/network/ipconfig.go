// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package network

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/config"
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/iam"
)

func getIPConfigurationClient() network.InterfaceIPConfigurationsClient {
	ipcClient := network.NewInterfaceIPConfigurationsClientWithBaseURI(
		config.Environment().ResourceManagerEndpoint, config.SubscriptionID())
	auth, _ := iam.GetResourceManagementAuthorizer()
	ipcClient.Authorizer = auth
	ipcClient.AddToUserAgent(config.UserAgent())
	return ipcClient
}

// GetPublicIP returns an existing public IP
func GetIPConfiguration(ctx context.Context, ipconfigID string) (ipconfig network.InterfaceIPConfiguration, err error) {
	ipcClient := getIPConfigurationClient()
	ipc, err := ParseIPConfigurationID(ipconfigID)
	if err != nil {
		return ipconfig, err
	}
	return ipcClient.Get(ctx, ipc.ResourceGroup, ipc.NicName, ipc.Name)
}

// ipConfigurationResource contains details about an Azure IPConfiguration resource.
type IPConfigurationResource struct {
	SubscriptionID string
	ResourceGroup  string
	NicName        string
	Name           string
}

// parseIPConfigurationID parses an IPConfiguration resource ID into a IPConfigurationResource struct.
func ParseIPConfigurationID(ipconfigID string) (resource IPConfigurationResource, err error) {
	const configIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft.Network/networkInterfaces/(.+)/ipConfigurations/(.+)`
	configIDPattern := regexp.MustCompile(configIDPatternText)
	match := configIDPattern.FindStringSubmatch(ipconfigID)

	if len(match) != 5 {
		return resource, fmt.Errorf("parsing failed for %s. Invalid ip configuration Id format", ipconfigID)
	}

	result := IPConfigurationResource{
		SubscriptionID: match[1],
		ResourceGroup:  match[2],
		NicName:        match[3],
		Name:           match[4],
	}

	return result, nil
}
