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

package loadbalancer

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

// ConfigData is supplied to the loadbalancer config template
type ConfigData struct {
	ControlPlanePort int
	BackendServers   map[string]string
}

// DefaultConfigTemplate is the loadbalancer config template
const DefaultConfigTemplate = `# generated by kind

# load balance over the control planes
stream {
    upstream tcp_backend {
    {{- range $server, $address := .BackendServers}}
        server {{ $address }};
    {{- end}}
    }

    server {
        listen {{ .ControlPlanePort }};
        proxy_pass tcp_backend;
    }
}

# minimal events entry to make nginx happy
events {}
`

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data *ConfigData) (config string, err error) {
	t, err := template.New("loadbalancer-config").Parse(DefaultConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}
	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}
	return buff.String(), nil
}
