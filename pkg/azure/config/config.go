package config

import (
	"github.com/yingeli/pod-external-ip-operator/pkg/azure/internal/config"
)

func ParseEnvironment() error {
	return config.ParseEnvironment()
}

func SetGroup(cloud string, subscription string, group string) {
	config.SetGroup(cloud, subscription, group)
}
