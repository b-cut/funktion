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
	"fmt"
	"os"
	"os/exec"

	"github.com/funktionio/funktion/pkg/k8sutil"
	"github.com/spf13/cobra"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"strings"
	"path/filepath"
	"github.com/funktionio/funktion/pkg/funktion"
	"strconv"
)

type debugCmd struct {
	kubeclient     *kubernetes.Clientset
	cmd            *cobra.Command
	kubeConfigPath string

	namespace      string
	kind           string
	name           string
	localPort      int
	remotePort     int
	chromeDevTools bool
	portText       string

	podAction      k8sutil.PodAction
	debugCmd       *exec.Cmd
}

func init() {
	RootCmd.AddCommand(newDebugCmd())
}

func newDebugCmd() *cobra.Command {
	p := &debugCmd{}
	p.podAction = k8sutil.PodAction{
		OnPodChange: p.viewLog,
	}
	cmd := &cobra.Command{
		Use:   "debug KIND NAME [flags]",
		Short: "debugs the given function or subscription",
		Long:  `This command will debug the latest container implementing the function or subscription`,
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			if len(args) < 1 {
				handleError(fmt.Errorf("No resource kind argument supplied! Possible values ['fn', 'subscription']"))
				return
			}
			if len(args) < 2 {
				handleError(fmt.Errorf("No name specified!"))
				return
			}
			p.kind = args[0]
			p.name = args[1]
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
	f.StringVarP(&p.namespace, "namespace", "n", "", "the namespace to query")
	f.StringVarP(&p.name, "name", "v", "latest", "the version of the connectors to install")
	f.IntVarP(&p.localPort, "local-port", "l", 0, "The localhost port to use for debugging or the container's debugging port is used")
	f.IntVarP(&p.remotePort, "remote-port", "r", 0, "The remote container port to use for debugging or the container's debugging port is used")
	f.BoolVarP(&p.chromeDevTools, "chrome", "c", false, "For node based containers open the Chrome DevTools to debug")
	return cmd
}

func (p *debugCmd) run() error {
	portText, err := p.createPortText(p.kind, p.name)
	if err != nil {
		return err
	}
	p.portText = portText
	kubeclient := p.kubeclient
	ds, err := kubeclient.Deployments(p.namespace).List(api.ListOptions{})
	if err != nil {
		return err
	}
	deployments := map[string]*v1beta1.Deployment{}
	for _, item := range ds.Items {
		name := item.Name
		deployments[name] = &item
	}
	deployment := deployments[p.name]
	if deployment == nil {
		return fmt.Errorf("No Deployment found called `%s`", p.name)
	}
	selector := deployment.Spec.Selector
	if selector == nil {
		return fmt.Errorf("Deployment `%s` does not have a selector!", p.name)
	}
	if selector.MatchLabels == nil {
		return fmt.Errorf("Deployment `%s` selector does not have a matchLabels!", p.name)
	}
	listOpts, err := k8sutil.V1BetaSelectorToListOptions(selector)
	if err != nil {
		return err
	}
	p.podAction.WatchPods(p.kubeclient, p.namespace, listOpts)
	return p.podAction.WatchLoop()
}

func (p *debugCmd) createPortText(kindText, name string) (string, error) {
	kind, listOpts, err := listOptsForKind(kindText)
	if err != nil {
		return "", err
	}
	cms := p.kubeclient.ConfigMaps(p.namespace)
	resources, err := cms.List(*listOpts)
	if err != nil {
		return "", err
	}
	var found *v1.ConfigMap
	for _, resource := range resources.Items {
		if name == resource.Name {
			found = &resource
		}
	}
	if found == nil {
		return "", fmt.Errorf("No %s resource found for name %s", kind, name)
	}

	debugPort := 0
	if kind == functionKind {
		runtime := ""
		data := found.Labels
		if data != nil {
			runtime = data[funktion.RuntimeLabel]
		}
		if len(runtime) > 0 {
			return p.createPortText(runtimeKind, runtime)
		}
	} else if kind == runtimeKind || kind == connectorKind {
		data := found.Data
		if data != nil {
			portValue := data[funktion.DebugPortProperty]
			if len(portValue) > 0 {
				debugPort, err = strconv.Atoi(portValue)
				if err != nil {
					return "", fmt.Errorf("Failed to convert debug port `%s` to a number due to %v", portValue, err)
				}
			}
		}
	} else if kind == subscriptionKind {
		connector := ""
		data := found.Labels
		if data != nil {
			connector = data[funktion.ConnectorLabel]
		}
		if len(connector) > 0 {
			return p.createPortText(runtimeKind, connector)
		}
	}
	if debugPort == 0 {
		if kind == connectorKind || kind == subscriptionKind {
			// default to java debug port for subscriptions and connectors if none specified
			debugPort = 5005
		}
	}
	if debugPort > 0 {
		if p.localPort == 0 {
			p.localPort = debugPort
		}
		if p.remotePort == 0 {
			p.remotePort = debugPort
		}
	}
	if p.remotePort == 0 {
		return "", fmt.Errorf("No remote debug port could be defaulted. Please specify one via the `-r` flag")
	}
	if p.localPort == 0 {
		p.localPort = p.remotePort
	}
	return fmt.Sprintf("%d:%d", p.localPort, p.remotePort), nil
}

func (p *debugCmd) viewLog(pod *v1.Pod) error {
	if pod != nil {
		binaryFile, err := k8sutil.ResolveKubectlBinary(p.kubeclient)
		if err != nil {
			return err
		}
		name := pod.Name
		if p.debugCmd != nil {
			process := p.debugCmd.Process
			if process != nil {
				process.Kill()
			}
		}
		args := []string{"port-forward", name, p.portText}

		fmt.Printf("\n%s %s\n\n", filepath.Base(binaryFile), strings.Join(args, " "))
		cmd := exec.Command(binaryFile, args...)
		p.debugCmd = cmd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return err
		}

		if p.chromeDevTools {
			return p.openChromeDevTools(pod)
		}
	}
	return nil
}

func (p *debugCmd) openChromeDevTools(pod *v1.Pod) error {
	// TODO
	return nil
}

