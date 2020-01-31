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

package repository

import (
	"fmt"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/pkg/internal/test"
)

func Test_inspectVariables(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "variable with different spacing around the name",
			args: args{
				data: "yaml with ${A} ${ B} ${ C} ${ D }",
			},
			want: []string{"A", "B", "C", "D"},
		},
		{
			name: "variables used in many places are grouped",
			args: args{
				data: "yaml with ${A} ${A} ${A}",
			},
			want: []string{"A"},
		},
		{
			name: "variables in multiline texts are processed",
			args: args{
				data: "yaml with ${A}\n${B}\n${C}",
			},
			want: []string{"A", "B", "C"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inspectVariables([]byte(tt.args.data)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inspectVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_replaceVariables(t *testing.T) {
	type args struct {
		yaml                  []byte
		variables             []string
		configVariablesClient config.VariablesClient
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "pass and replaces variables",
			args: args{
				yaml:      []byte("foo ${ BAR }"),
				variables: []string{"BAR"},
				configVariablesClient: test.NewFakeVariableClient().
					WithVar("BAR", "bar"),
			},
			want:    []byte("foo bar"),
			wantErr: false,
		},
		{
			name: "fails for missing variables",
			args: args{
				yaml:                  []byte("foo ${ BAR } ${ BAZ }"),
				variables:             []string{"BAR", "BAZ"},
				configVariablesClient: test.NewFakeVariableClient(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceVariables(tt.args.yaml, tt.args.variables, tt.args.configVariablesClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("replaceVariables() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_inspectTargetNamespace(t *testing.T) {
	type args struct {
		objs []unstructured.Unstructured
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "get targetNamespace if exists",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": "Namespace",
							"metadata": map[string]interface{}{
								"name": "foo",
							},
						},
					},
				},
			},
			want: "foo",
		},
		{
			name: "return empty if there is no targetNamespace",
			args: args{
				objs: []unstructured.Unstructured{},
			},
			want: "",
		},
		{
			name: "fails if two Namespace exists",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": "Namespace",
							"metadata": map[string]interface{}{
								"name": "foo",
							},
						},
					},
					{
						Object: map[string]interface{}{
							"kind": "Namespace",
							"metadata": map[string]interface{}{
								"name": "bar",
							},
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inspectTargetNamespace(tt.args.objs)
			if (err != nil) != tt.wantErr {
				t.Errorf("inspectTargetNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("inspectTargetNamespace() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fixTargetNamespace(t *testing.T) {
	type args struct {
		objs            []unstructured.Unstructured
		targetNamespace string
	}
	tests := []struct {
		name string
		args args
		want []unstructured.Unstructured
	}{
		{
			name: "fix Namespace object if exists",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": namespaceKind,
							"metadata": map[string]interface{}{
								"name": "foo",
							},
						},
					},
				},
				targetNamespace: "bar",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind": namespaceKind,
						"metadata": map[string]interface{}{
							"name": "bar",
						},
					},
				},
			},
		},
		{
			name: "fix namespaced objects",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": "Pod",
						},
					},
				},
				targetNamespace: "bar",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind": "Pod",
						"metadata": map[string]interface{}{
							"namespace": "bar",
						},
					},
				},
			},
		},
		{
			name: "ignore global objects",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": "ClusterRole",
						},
					},
				},
				targetNamespace: "bar",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind": "ClusterRole",
						// no namespace
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixTargetNamespace(tt.args.objs, tt.args.targetNamespace); !reflect.DeepEqual(got[0], tt.want[0]) { //skipping from test the automatically added namespace Object
				t.Errorf("fixTargetNamespace() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addNamespaceIfMissing(t *testing.T) {
	type args struct {
		objs            []unstructured.Unstructured
		targetNamespace string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "don't add Namespace object if exists",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": namespaceKind,
							"metadata": map[string]interface{}{
								"name": "foo",
							},
						},
					},
				},
				targetNamespace: "foo",
			},
		},
		{
			name: "add Namespace object if it does not exists",
			args: args{
				objs:            []unstructured.Unstructured{},
				targetNamespace: "bar",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addNamespaceIfMissing(tt.args.objs, tt.args.targetNamespace)

			wgot, err := inspectTargetNamespace(got)
			if err != nil {
				t.Fatalf("inspectTargetNamespace() error = %v", err)
			}

			if wgot != tt.args.targetNamespace {
				t.Errorf("addNamespaceIfMissing().targetNamespace got = %v, want %v", wgot, tt.args.targetNamespace)
			}
		})
	}
}

func Test_fixRBAC(t *testing.T) {
	type args struct {
		objs            []unstructured.Unstructured
		targetNamespace string
	}
	tests := []struct {
		name    string
		args    args
		want    []unstructured.Unstructured
		wantErr bool
	}{
		{
			name: "ClusterRole get fixed",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind":       "ClusterRole",
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"metadata": map[string]interface{}{
								"name": "foo",
							},
						},
					},
				},
				targetNamespace: "target",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind":       "ClusterRole",
						"apiVersion": "rbac.authorization.k8s.io/v1",
						"metadata": map[string]interface{}{
							"name": "target-foo", // ClusterRole name fixed!
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ClusterRoleBinding with roleRef NOT IN components YAML get fixed",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind":       "ClusterRoleBinding",
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"metadata": map[string]interface{}{
								"name": "foo",
							},
							"roleRef": map[string]interface{}{
								"apiGroup": "",
								"kind":     "",
								"name":     "bar",
							},
							"subjects": []interface{}{
								map[string]interface{}{
									"kind":      "ServiceAccount",
									"name":      "baz",
									"namespace": "baz",
								},
								map[string]interface{}{
									"kind": "User",
									"name": "qux",
								},
							},
						},
					},
				},
				targetNamespace: "target",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind":       "ClusterRoleBinding",
						"apiVersion": "rbac.authorization.k8s.io/v1",
						"metadata": map[string]interface{}{
							"name":              "target-foo", // ClusterRoleBinding name fixed!
							"creationTimestamp": nil,
						},
						"roleRef": map[string]interface{}{
							"apiGroup": "",
							"kind":     "",
							"name":     "bar", // ClusterRole name NOT fixed (not in components YAML)!
						},
						"subjects": []interface{}{
							map[string]interface{}{
								"kind":      "ServiceAccount",
								"name":      "baz",
								"namespace": "target", // Subjects namespace fixed!
							},
							map[string]interface{}{
								"kind": "User",
								"name": "qux",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ClusterRoleBinding with roleRef IN components YAML get fixed",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind":       "ClusterRoleBinding",
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"metadata": map[string]interface{}{
								"name": "foo",
							},
							"roleRef": map[string]interface{}{
								"apiGroup": "",
								"kind":     "",
								"name":     "bar",
							},
							"subjects": []interface{}{
								map[string]interface{}{
									"kind":      "ServiceAccount",
									"name":      "baz",
									"namespace": "baz",
								},
								map[string]interface{}{
									"kind": "User",
									"name": "qux",
								},
							},
						},
					},
					{
						Object: map[string]interface{}{
							"kind":       "ClusterRole",
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"metadata": map[string]interface{}{
								"name": "bar",
							},
						},
					},
				},
				targetNamespace: "target",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind":       "ClusterRoleBinding",
						"apiVersion": "rbac.authorization.k8s.io/v1",
						"metadata": map[string]interface{}{
							"name":              "target-foo", // ClusterRoleBinding name fixed!
							"creationTimestamp": nil,
						},
						"roleRef": map[string]interface{}{
							"apiGroup": "",
							"kind":     "",
							"name":     "target-bar", // ClusterRole name fixed!
						},
						"subjects": []interface{}{
							map[string]interface{}{
								"kind":      "ServiceAccount",
								"name":      "baz",
								"namespace": "target", // Subjects namespace fixed!
							},
							map[string]interface{}{
								"kind": "User",
								"name": "qux",
							},
						},
					},
				},
				{
					Object: map[string]interface{}{
						"kind":       "ClusterRole",
						"apiVersion": "rbac.authorization.k8s.io/v1",
						"metadata": map[string]interface{}{
							"name": "target-bar", // ClusterRole fixed!
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "RoleBinding get fixed",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind":       "RoleBinding",
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"metadata": map[string]interface{}{
								"name":      "foo",
								"namespace": "target",
							},
							"roleRef": map[string]interface{}{
								"apiGroup": "",
								"kind":     "",
								"name":     "bar",
							},
							"subjects": []interface{}{
								map[string]interface{}{
									"kind":      "ServiceAccount",
									"name":      "baz",
									"namespace": "baz",
								},
								map[string]interface{}{
									"kind": "User",
									"name": "qux",
								},
							},
						},
					},
				},
				targetNamespace: "target",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind":       "RoleBinding",
						"apiVersion": "rbac.authorization.k8s.io/v1",
						"metadata": map[string]interface{}{
							"name":              "foo",
							"namespace":         "target",
							"creationTimestamp": nil,
						},
						"roleRef": map[string]interface{}{
							"apiGroup": "",
							"kind":     "",
							"name":     "bar",
						},
						"subjects": []interface{}{
							map[string]interface{}{
								"kind":      "ServiceAccount",
								"name":      "baz",
								"namespace": "target", // Subjects namespace fixed!
							},
							map[string]interface{}{
								"kind": "User",
								"name": "qux",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixRBAC(tt.args.objs, tt.args.targetNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("fixRBAC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fixRBAC() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func fakeDeployment(watchNamespace string) unstructured.Unstructured {
	var args []string //nolint
	if watchNamespace != "" {
		args = append(args, fmt.Sprintf("%s%s", namespaceArgPrefix, watchNamespace))
	}
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       deploymentKind,
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name": controllerContainerName,
								"args": args,
							},
						},
					},
				},
			},
		},
	}
}

func Test_inspectWatchNamespace(t *testing.T) {
	type args struct {
		objs []unstructured.Unstructured
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "get watchingNamespace if exists",
			args: args{
				objs: []unstructured.Unstructured{
					fakeDeployment("foo"),
				},
			},
			want: "foo",
		},
		{
			name: "get watchingNamespace if exists more than once, but it is consistent",
			args: args{
				objs: []unstructured.Unstructured{
					fakeDeployment("foo"),
					fakeDeployment("foo"),
				},
			},
			want: "foo",
		},
		{
			name: "return empty if there is no watchingNamespace",
			args: args{
				objs: []unstructured.Unstructured{},
			},
			want: "",
		},
		{
			name: "fails if inconsistent watchingNamespace",
			args: args{
				objs: []unstructured.Unstructured{
					fakeDeployment("foo"),
					fakeDeployment("bar"),
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inspectWatchNamespace(tt.args.objs)
			if (err != nil) != tt.wantErr {
				t.Errorf("inspectWatchNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("inspectWatchNamespace() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fixWatchNamespace(t *testing.T) {
	type args struct {
		objs              []unstructured.Unstructured
		watchingNamespace string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "fix if existing",
			args: args{
				objs: []unstructured.Unstructured{
					fakeDeployment("foo"),
				},
				watchingNamespace: "bar",
			},
			wantErr: false,
		},
		{
			name: "set if not existing",
			args: args{
				objs: []unstructured.Unstructured{
					fakeDeployment(""),
				},
				watchingNamespace: "bar",
			},
			wantErr: false,
		},
		{
			name: "unset if existing",
			args: args{
				objs: []unstructured.Unstructured{
					fakeDeployment("foo"),
				},
				watchingNamespace: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixWatchNamespace(tt.args.objs, tt.args.watchingNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("fixWatchNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			wgot, err := inspectWatchNamespace(got)
			if err != nil {
				t.Fatalf("inspectWatchNamespace() error = %v", err)
			}

			if wgot != tt.args.watchingNamespace {
				t.Errorf("fixWatchNamespace().watchingNamespace got = %v, want %v", wgot, tt.args.watchingNamespace)
			}
		})
	}
}

func Test_inspectImages(t *testing.T) {
	type args struct {
		objs []unstructured.Unstructured
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "controller without the RBAC proxy",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       deploymentKind,
							"spec": map[string]interface{}{
								"template": map[string]interface{}{
									"spec": map[string]interface{}{
										"containers": []map[string]interface{}{
											{
												"name":  controllerContainerName,
												"image": "gcr.io/k8s-staging-cluster-api/cluster-api-controller:master",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    []string{"gcr.io/k8s-staging-cluster-api/cluster-api-controller:master"},
			wantErr: false,
		},
		{
			name: "controller with the RBAC proxy",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       deploymentKind,
							"spec": map[string]interface{}{
								"template": map[string]interface{}{
									"spec": map[string]interface{}{
										"containers": []map[string]interface{}{
											{
												"name":  controllerContainerName,
												"image": "gcr.io/k8s-staging-cluster-api/cluster-api-controller:master",
											},
											{
												"name":  "kube-rbac-proxy",
												"image": "gcr.io/kubebuilder/kube-rbac-proxy:v0.4.1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    []string{"gcr.io/k8s-staging-cluster-api/cluster-api-controller:master", "gcr.io/kubebuilder/kube-rbac-proxy:v0.4.1"},
			wantErr: false,
		},
		{
			name: "controller with init container",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       deploymentKind,
							"spec": map[string]interface{}{
								"template": map[string]interface{}{
									"spec": map[string]interface{}{
										"containers": []map[string]interface{}{
											{
												"name":  controllerContainerName,
												"image": "gcr.io/k8s-staging-cluster-api/cluster-api-controller:master",
											},
										},
										"initContainers": []map[string]interface{}{
											{
												"name":  controllerContainerName,
												"image": "gcr.io/k8s-staging-cluster-api/cluster-api-controller:init",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    []string{"gcr.io/k8s-staging-cluster-api/cluster-api-controller:master", "gcr.io/k8s-staging-cluster-api/cluster-api-controller:init"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inspectImages(tt.args.objs)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addLabels(t *testing.T) {
	type args struct {
		objs []unstructured.Unstructured
		name string
	}
	tests := []struct {
		name string
		args args
		want []unstructured.Unstructured
	}{
		{
			name: "add labels",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind": "ClusterRole",
						},
					},
				},
				name: "provider",
			},
			want: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"kind": "ClusterRole",
						"metadata": map[string]interface{}{
							"labels": map[string]interface{}{
								clusterctlv1.ClusterctlLabelName: "",
								clusterv1.ProviderLabelName:      "provider",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addLabels(tt.args.objs, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
