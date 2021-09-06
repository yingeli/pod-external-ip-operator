package azurecni

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/coreos/go-iptables/iptables"
)

const (
	localChainName  = "EXTERNAL-IP-LOCAL"
	egressChainName = "EXTERNAL-IP-EGRESS"
)

func SetupIptables(localNetworks []string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	exists, err := ipt.ChainExists("nat", egressChainName)
	if err != nil {
		return err
	}
	if !exists {
		if err := ipt.NewChain("nat", egressChainName); err != nil {
			return err
		}
	}
	ruleSpec := []string{"-j", "RETURN"}
	if err := ipt.AppendUnique("nat", egressChainName, ruleSpec...); err != nil {
		return err
	}

	if err := ipt.ClearChain("nat", localChainName); err != nil {
		return err
	}
	for _, ln := range localNetworks {
		if ln == "" {
			continue
		}
		ruleSpec := []string{"-d", ln, "-j", "RETURN"}
		if err := ipt.AppendUnique("nat", localChainName, ruleSpec...); err != nil {
			return fmt.Errorf("error adding local network %s, error: %v", ln, err)
		}
	}
	ruleSpec = []string{"-j", egressChainName}
	if err := ipt.Append("nat", localChainName, ruleSpec...); err != nil {
		return err
	}
	ruleSpec = []string{"-j", "RETURN"}
	if err := ipt.Append("nat", localChainName, ruleSpec...); err != nil {
		return err
	}

	ruleSpec = []string{"-j", localChainName}
	if err := insertUnique(ipt, "nat", "POSTROUTING", ruleSpec); err != nil {
		return err
	}

	return nil
}

func AddPodIPRules(pod *corev1.Pod) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	ruleSpec := []string{"-s", pod.Status.PodIP, "-j", "ACCEPT", "-m", "comment", "--comment", getComment(pod)}
	if err := insertUnique(ipt, "nat", egressChainName, ruleSpec); err != nil {
		return err
	}
	return nil
}

func RemovePodIPRules(pod *corev1.Pod) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	rules, err := ipt.List("nat", egressChainName)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		ruleSpec := strings.Split(rule, " \"")
		if ruleComment(ruleSpec) == getComment(pod) {
			if err := ipt.Delete("nat", egressChainName, ruleSpec[2:]...); err != nil {
				return err
			}
		}
	}

	return nil
}

func getComment(pod *corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func ruleComment(ruleSpec []string) string {
	for i, t := range ruleSpec {
		if t == "--comment" && i < len(ruleSpec)-1 {
			return ruleSpec[i+1]
		}
	}
	return ""
}

func insertUnique(ipt *iptables.IPTables, table string, chain string, ruleSpec []string) error {
	hasRule, err := ipt.Exists(table, chain, ruleSpec...)
	if err != nil {
		return err
	}
	if !hasRule {
		if err := ipt.Insert(table, chain, 1, ruleSpec...); err != nil {
			return err
		}
	}
	return nil
}
