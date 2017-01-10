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

package funktion

import "time"

const (
	// KindLabel is the label key used on ConfigMaps to indicate the kind of resource
	KindLabel = "funktion.fabric8.io/kind"

	// Flow

	// ConnectorLabel is the label key used on a ConfigMap to refer to a Connector
	ConnectorLabel = "connector"

	// Function

	// RuntimeLabel is the label key used on a ConfigMap to refer to a Runtime
	RuntimeLabel = "runtime"
	// ProjectLabel the name of the folder where the source comes from
	ProjectLabel = "project"

	// ConnectorKind is the value of a Connector fo the KindLabel
	ConnectorKind = "Connector"
	// FlowKind is the value of a Flow fo the KindLabel
	FlowKind = "Flow"
	// RuntimeKind is the value of a Runtime fo the KindLabel
	RuntimeKind = "Runtime"
	// FunctionKind is the value of a Function fo the KindLabel
	FunctionKind = "Function"
	// DeploymentKind is the value of a Deployment fo the KindLabel
	DeploymentKind = "Deployment"
	// ServiceKind is the value of a ConneServicector fo the KindLabel
	ServiceKind = "Service"

	// Runtime

	// ChromeDevToolsAnnotation boolean annotation to indicate chrome dev tools is enabled
	// and that the URL will appear in the pods log
	ChromeDevToolsAnnotation = "funktion.fabric8.io/chromeDevTools"

	// VersionLabel the version of the runtime
	VersionLabel = "version"

	// FileExtensionsProperty a comma separated list of file extensions (without the dot) which are handled by this runtime
	FileExtensionsProperty = "fileExtensions"

	// SourceMountPathProperty the path in the docker image where we should mount the source code
	SourceMountPathProperty = "sourceMountPath"

	resyncPeriod = 30 * time.Second
)
