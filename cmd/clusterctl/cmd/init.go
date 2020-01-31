/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/client"
)

type initOptions struct {
	kubeconfig              string
	coreProvider            string
	bootstrapProviders      []string
	controlPlaneProviders   []string
	infrastructureProviders []string
	targetNamespace         string
	watchingNamespace       string
}

var io = &initOptions{}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a management cluster for Cluster API",
	Long: LongDesc(`
		Initialize a management cluster for Cluster API by installing Cluster API core components,
		the kubeadm bootstrap provider, and the selected bootstrap and infrastructure providers.

		The management cluster must be an existing Kubernetes cluster, and the identity used
		for accessing the cluster must have enough privileges for installing Cluster API providers.

		Use 'clusterctl config providers' to get the list of available providers; if necessary, edit
		the $HOME/.cluster-api/clusterctl.yaml file to add new provider configurations or to customize existing ones.

		Some providers require environment variables to be set before running clusterctl init; please
		refer to the provider documentation or use 'clusterctl config provider [name]' to get the
		list of required variables.

		See https://cluster-api.sigs.k8s.io/ for more details.`),

	Example: Examples(`
		# Initialize a management cluster, the cluster currently pointed by kubectl, by installing the
		# AWS infrastructure provider. Please note that this command, when executed on an empty management cluster,
 		# automatically triggers the installation of the cluster-api core provider.
		clusterctl init --infrastructure=aws

		# Initialize a management cluster with a specific version of the AWS infrastructure provider.
		clusterctl init --infrastructure=aws:v0.4.1

		# Initialize a management cluster, the cluster defined in the "foo.yaml" kubeconfig file, by
		# installing the AWS infrastructure provider.
		clusterctl init --kubeconfig=foo.yaml  --infrastructure=aws

		# Initialize a management cluster, by installing both the AWS and the vSphere infrastructure provider
		clusterctl init --infrastructure=aws,vsphere

		# Initialize a management cluster by installing the provider's components' in the "foo" namespace.
		clusterctl init --infrastructure aws --target-namespace foo

		# Initialize a management cluster and configures all the providers for watching Cluster API
		# objects in the "foo" namespace only.
		clusterctl init --infrastructure aws --watching-namespace=foo`),

	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	initCmd.Flags().StringVarP(&io.kubeconfig, "kubeconfig", "", "", "Path to the kubeconfig file to use for accessing the management cluster. If empty, default rules for kubeconfig discovery will be used")
	initCmd.Flags().StringVarP(&io.coreProvider, "core", "", "", "Core provider version (e.g. cluster-api:v0.3.0) to add to the management cluster. By default (empty), the cluster-api core provider's latest release is used")
	initCmd.Flags().StringSliceVarP(&io.infrastructureProviders, "infrastructure", "i", nil, "Infrastructure providers and versions (e.g. aws:v0.5.0) to add to the management cluster")
	initCmd.Flags().StringSliceVarP(&io.bootstrapProviders, "bootstrap", "b", nil, "Bootstrap providers and versions (e.g. kubeadm-bootstrap:v0.3.0) to add to the management cluster. By default (empty), the kubeadm bootstrap provider's latest release is used")
	initCmd.Flags().StringSliceVarP(&io.controlPlaneProviders, "control-plane", "c", nil, "ControlPlane providers and versions (e.g. kubeadm-control-plane:v0.3.0) to add to the management cluster. By default (empty), the kubeadm control plane provider latest release is used")
	initCmd.Flags().StringVarP(&io.targetNamespace, "target-namespace", "", "", "The target namespace where the providers should be deployed. If not specified, each provider will be installed in a provider's default namespace")
	initCmd.Flags().StringVarP(&io.watchingNamespace, "watching-namespace", "", "", "Namespace that the providers should watch to reconcile Cluster API objects. If unspecified, the providers watches for Cluster API objects across all namespaces")

	RootCmd.AddCommand(initCmd)
}

func runInit() error {
	c, err := client.New(cfgFile)
	if err != nil {
		return err
	}

	fmt.Println("performing init...")

	componentList, firstExecution, err := c.Init(client.InitOptions{
		Kubeconfig:              io.kubeconfig,
		CoreProvider:            io.coreProvider,
		BootstrapProviders:      io.bootstrapProviders,
		ControlPlaneProviders:   io.controlPlaneProviders,
		InfrastructureProviders: io.infrastructureProviders,
		TargetNamespace:         io.targetNamespace,
		WatchingNamespace:       io.watchingNamespace,
	})
	if err != nil {
		return err
	}

	for _, components := range componentList {
		fmt.Printf(" - %s %s installed (%s)\n", components.Name(), components.Type(), components.Version())
	}

	if firstExecution {
		fmt.Println("\nYour cluster API management cluster has been initialized successfully!")
		fmt.Println("\nYou can now create your first workload cluster by running the following:")
		fmt.Println("\n  clusterctl config cluster [name] --kubernetes-version [version] | kubectl apply -f -")
		fmt.Println("")
	}

	return nil
}
