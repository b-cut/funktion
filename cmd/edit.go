//  Copyright 2016 Red Hat, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package cmd

import (
	"bytes"
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/magiconair/properties"
	"github.com/fabric8io/funktion-operator/pkg/funktion"
	"github.com/fabric8io/funktion-operator/pkg/spec"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

type editConnectorCmd struct {
	kubeclient            *kubernetes.Clientset
	cmd                   *cobra.Command
	kubeConfigPath        string

	namespace             string
	name                  string

	configMaps            map[string]*v1.ConfigMap
	schema                *spec.ConnectorSchema
	applicationProperties *properties.Properties
}

func init() {
	RootCmd.AddCommand(newEditCmd())
}

func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit KIND [NAME] [flags]",
		Short: "edits a resources",
		Long:  `This command will create edit a resource`,
	}

	cmd.AddCommand(newEditConnectorCmd())
	return cmd
}

func newEditConnectorCmd() *cobra.Command {
	p := &editConnectorCmd{
	}
	cmd := &cobra.Command{
		Use:   "connector NAME [flags]",
		Short: "edits a connectors configuration",
		Long:  `This command will edit a connector resource`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			if len(args) == 0 {
				fmt.Printf("You must specify the name of the connector as an argument!")
				return
			}
			p.name = args[0]
			err := createKubernetesClient(cmd, p.kubeConfigPath, &p.kubeclient, &p.namespace)
			if err != nil {
				handleError(err)
				return
			}
			handleError(p.run())
		},
	}
	f := cmd.Flags()
	f.StringVar(&p.kubeConfigPath, "kubeconfig", "", "the directory to look for the kubernetes configuration")
	f.StringVar(&p.namespace, "namespace", "", "the namespace to query")
	return cmd
}

func (p *editConnectorCmd) run() error {
	listOpts, err := funktion.CreateConnectorListOptions()
	if err != nil {
		return err
	}
	kubeclient := p.kubeclient
	cms := kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return err
	}
	p.configMaps = map[string]*v1.ConfigMap{}
	for _, resource := range resources.Items {
		p.configMaps[resource.Name] = &resource
	}

	name := p.name
	connector := p.configMaps[name]
	if connector == nil {
		return fmt.Errorf("No Connector called `%s` exists in namespace %s", name, p.namespace)
	}
	err = p.loadConnectorSchema(name, connector)
	if err != nil {
		return err
	}
	err = p.editConnector(name, connector)
	if err != nil {
		return err
	}
	_, err = cms.Update(connector);
	if err == nil {
		fmt.Printf("Connector %s updated\n", name)
	}
	return err
}

func (p *editConnectorCmd) loadConnectorSchema(name string, connector *v1.ConfigMap) error {
	// lets load the connector model
	if connector.Data == nil {
		return fmt.Errorf("No data in the Connector %s", name);
	}
	schemaYaml := connector.Data[funktion.SchemaYmlProperty]
	if len(schemaYaml) == 0 {
		return fmt.Errorf("No YAML for data key %s in Connector %s", funktion.SchemaYmlProperty, name);
	}
	schema, err := funktion.LoadConnectorSchema([]byte(schemaYaml))
	if err != nil {
		return err
	}
	p.schema = schema

	propertiesText := connector.Data[funktion.ApplicationPropertiesProperty]
	if len(propertiesText) > 0 {
		props, err := properties.LoadString(propertiesText)
		if err != nil {
			return err
		}
		p.applicationProperties = props
	} else {
		p.applicationProperties = properties.NewProperties()
	}
	return nil
}

func (p *editConnectorCmd) editConnector(name string, connector *v1.ConfigMap) error {
	compProps := p.schema.ComponentProperties
	if len(compProps) == 0 {
		fmt.Printf("The connector `%s` has no properties to configure!", name)
		return nil
	}

	updated := false
	for k, cp := range compProps {
		label := funktion.HumanizeString(k)
		propertyKey := "camel.component." + name + "." + funktion.ToSpringBootPropertyName(k)

		value := p.applicationProperties.GetString(propertyKey, "")
		valueText := ""
		boolType := cp.Type == "boolean"
		if len(value) > 0 {
			if boolType {
				if value == "true" {
					valueText = "[Y/n]"
				} else {
					valueText = "[y/N]"
				}

			} else {
				valueText = "[" + value + "]"
			}
		}
		prompt := "?"
		if cp.Required {
			prompt = "*"
		}
		fmt.Printf("%s %s%s: ", prompt, label, valueText)

		var input string
		fmt.Scanln(&input)
		if len(input) > 0 {
			// convert boolean to true/false values
			if boolType {
				lower := strings.TrimSpace(strings.ToLower(input))
				if lower[0] == 't' {
					input = "true"
				} else {
					input = "false"
				}
			}

			p.applicationProperties.Set(propertyKey, input)
			updated = true
		}
	}

	if updated {
		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		p.applicationProperties.Write(w, properties.UTF8)
		w.Flush()
		propText := b.String()
		cms := p.kubeclient.ConfigMaps(p.namespace)
		latestCon, err := cms.Get(name)
		if err != nil {
			return err
		}
		latestCon.Data[funktion.ApplicationPropertiesProperty] = propText
		_, err = cms.Update(latestCon)
		if err != nil {
			fmt.Printf("Connector %s updated", name)
		}
		return err
	}
	return nil
}
