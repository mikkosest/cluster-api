/*
Copyright 2020 The Kubernetes Authors.

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

package repository

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/internal/test"
)

func Test_templates_Get(t *testing.T) {
	p1 := config.NewProvider("p1", "", clusterctlv1.BootstrapProviderType)

	type fields struct {
		version               string
		provider              config.Provider
		repository            Repository
		configVariablesClient config.VariablesClient
	}
	type args struct {
		flavor          string
		bootstrap       string
		targetNamespace string
	}
	type want struct {
		provider        config.Provider
		version         string
		flavor          string
		bootstrap       string
		variables       []string
		targetNamespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "pass if default template exists",
			fields: fields{
				version:  "v1.0",
				provider: p1,
				repository: test.NewFakeRepository().
					WithPaths("root", "").
					WithDefaultVersion("v1.0").
					WithFile("v1.0", "config-kubeadm.yaml", templateMapYaml),
				configVariablesClient: test.NewFakeVariableClient().WithVar(variableName, variableValue),
			},
			args: args{
				flavor:          "",
				bootstrap:       "kubeadm",
				targetNamespace: "ns1",
			},
			want: want{
				provider:        p1,
				version:         "v1.0",
				flavor:          "",
				bootstrap:       "kubeadm",
				variables:       []string{variableName},
				targetNamespace: "ns1",
			},
			wantErr: false,
		},
		{
			name: "pass if template for a flavor exists",
			fields: fields{
				version:  "v1.0",
				provider: p1,
				repository: test.NewFakeRepository().
					WithPaths("root", "").
					WithDefaultVersion("v1.0").
					WithFile("v1.0", "config-prod-kubeadm.yaml", templateMapYaml),
				configVariablesClient: test.NewFakeVariableClient().WithVar(variableName, variableValue),
			},
			args: args{
				flavor:          "prod",
				bootstrap:       "kubeadm",
				targetNamespace: "ns1",
			},
			want: want{
				provider:        p1,
				version:         "v1.0",
				flavor:          "prod",
				bootstrap:       "kubeadm",
				variables:       []string{variableName},
				targetNamespace: "ns1",
			},
			wantErr: false,
		},
		{
			name: "fails if template does not exists",
			fields: fields{
				version:  "v1.0",
				provider: p1,
				repository: test.NewFakeRepository().
					WithPaths("root", "").
					WithDefaultVersion("v1.0"),
				configVariablesClient: test.NewFakeVariableClient().WithVar(variableName, variableValue),
			},
			args: args{
				flavor:          "",
				bootstrap:       "kubeadm",
				targetNamespace: "ns1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTemplateClient(tt.fields.provider, tt.fields.version, tt.fields.repository, tt.fields.configVariablesClient)
			got, err := f.Get(tt.args.flavor, tt.args.bootstrap, tt.args.targetNamespace)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if got.Name() != tt.want.provider.Name() {
				t.Errorf("got.Name() = %v, want = %v ", got.Name(), tt.want.provider.Name())
			}

			if got.Type() != tt.want.provider.Type() {
				t.Errorf("got.Type() = %v, want = %v ", got.Type(), tt.want.provider.Type())
			}

			if got.Version() != tt.want.version {
				t.Errorf("got.Version() = %v, want = %v ", got.Version(), tt.want.version)
			}

			if got.Bootstrap() != tt.want.bootstrap {
				t.Errorf("got.Bootstrap() = %v, want = %v ", got.Bootstrap(), tt.want.bootstrap)
			}

			if !reflect.DeepEqual(got.Variables(), tt.want.variables) {
				t.Errorf("got.Variables() = %v, want = %v ", got.Variables(), tt.want.variables)
			}

			if !reflect.DeepEqual(got.TargetNamespace(), tt.want.targetNamespace) {
				t.Errorf("got.TargetNamespace() = %v, want = %v ", got.TargetNamespace(), tt.want.targetNamespace)
			}

			// check variable replaced in yaml
			yaml, err := got.Yaml()
			if tt.wantErr {
				t.Fatalf("got.Yaml error = %v", err)
			}

			if !bytes.Contains(yaml, []byte(fmt.Sprintf("variable: %s", variableValue))) {
				t.Error("got.Yaml without variable substitution")
			}

			// check if target namespace is set
			for _, o := range got.Objs() {
				if o.GetNamespace() != tt.want.targetNamespace {
					t.Errorf("got.Object[%s].Namespace = %v, want = %v ", o.GetName(), o.GetNamespace(), tt.want.targetNamespace)
				}
			}
		})
	}
}
