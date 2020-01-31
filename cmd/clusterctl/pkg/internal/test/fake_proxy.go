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

package test

import (
	apiextensionslv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	fakebootstrap "sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/internal/test/providers/bootstrap"
	fakecontrolplane "sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/internal/test/providers/controlplane"
	fakeinfrastructure "sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/internal/test/providers/infrastructure"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type FakeProxy struct {
	cs   client.Client
	objs []runtime.Object
}

var (
	FakeScheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(FakeScheme)
	_ = clusterctlv1.AddToScheme(FakeScheme)
	_ = clusterv1.AddToScheme(FakeScheme)
	_ = apiextensionslv1.AddToScheme(FakeScheme)

	_ = fakebootstrap.AddToScheme(FakeScheme)
	_ = fakecontrolplane.AddToScheme(FakeScheme)
	_ = fakeinfrastructure.AddToScheme(FakeScheme)
}

func (f *FakeProxy) CurrentNamespace() (string, error) {
	return "default", nil
}

func (f *FakeProxy) NewClient() (client.Client, error) {
	if f.cs != nil {
		return f.cs, nil
	}
	f.cs = fake.NewFakeClientWithScheme(FakeScheme, f.objs...)

	return f.cs, nil
}

// ListResources returns all the resources known by the FakeProxy
func (f *FakeProxy) ListResources(namespace string, labels map[string]string) ([]unstructured.Unstructured, error) {
	var ret []unstructured.Unstructured //nolint
	for _, o := range f.objs {
		u := unstructured.Unstructured{}
		err := FakeScheme.Convert(o, &u, nil)
		if err != nil {
			return nil, err
		}

		// filter by namespace, if any
		if namespace != "" && u.GetNamespace() != "" && u.GetNamespace() != namespace {
			continue
		}

		// filter by label, if any
		haslabel := false
		for l, v := range labels {
			for ul, uv := range u.GetLabels() {
				if l == ul && v == uv {
					haslabel = true
				}
			}
		}
		if !haslabel {
			continue
		}

		ret = append(ret, u)
	}

	return ret, nil
}

func NewFakeProxy() *FakeProxy {
	return &FakeProxy{}
}

func (f *FakeProxy) WithObjs(objs ...runtime.Object) *FakeProxy {
	f.objs = append(f.objs, objs...)
	return f
}

// WithProviderInventory can be used as a fast track for setting up test scenarios requiring an already initialized management cluster.
// NB. this method adds an items to the Provider inventory, but it doesn't install the corresponding provider; if the
// test case requires the actual provider to be installed, use the the fake client to install both the provider
// components and the corresponding inventory item.
func (f *FakeProxy) WithProviderInventory(name string, providerType clusterctlv1.ProviderType, version, targetNamespace, watchingNamespace string) *FakeProxy {
	f.objs = append(f.objs, &clusterctlv1.Provider{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterctlv1.GroupVersion.String(),
			Kind:       "Provider",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: targetNamespace,
			Name:      name,
			Labels: map[string]string{
				clusterctlv1.ClusterctlLabelName:     "",
				clusterv1.ProviderLabelName:          name,
				clusterctlv1.ClusterctlCoreLabelName: "inventory",
			},
		},
		Type:             string(providerType),
		Version:          version,
		WatchedNamespace: watchingNamespace,
	})

	return f
}
