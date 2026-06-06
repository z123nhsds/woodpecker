// Copyright 2022 Woodpecker Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package matrix

import (
	"fmt"
	"strings"

	yaml_types "go.woodpecker-ci.org/woodpecker/v3/pipeline/frontend/yaml/types"
)

// Generator generates multiple workflows from a matrix configuration.
type Generator struct{}

// NewGenerator creates a new matrix generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates multiple workflows from a single workflow with matrix configuration.
func (g *Generator) Generate(workflow *yaml_types.Workflow) ([]*yaml_types.Workflow, error) {
	if len(workflow.Matrix) == 0 {
		return []*yaml_types.Workflow{workflow}, nil
	}

	axes, err := calcMatrix(workflow.Matrix)
	if err != nil {
		return nil, err
	}

	var workflows []*yaml_types.Workflow
	for _, axis := range axes {
		clone := g.cloneWorkflow(workflow)
		g.applyAxis(clone, axis)
		workflows = append(workflows, clone)
	}

	return workflows, nil
}

// cloneWorkflow creates a deep copy of a workflow.
func (g *Generator) cloneWorkflow(workflow *yaml_types.Workflow) *yaml_types.Workflow {
	clone := &yaml_types.Workflow{
		When:      workflow.When,
		Workspace: workflow.Workspace,
		Clone: yaml_types.ContainerList{
			ContainerList: make([]*yaml_types.Container, len(workflow.Clone.ContainerList)),
		},
		Steps: yaml_types.ContainerList{
			ContainerList: make([]*yaml_types.Container, len(workflow.Steps.ContainerList)),
		},
		Services: yaml_types.ContainerList{
			ContainerList: make([]*yaml_types.Container, len(workflow.Services.ContainerList)),
		},
		Labels:    make(map[string]string),
		DependsOn: workflow.DependsOn,
		SkipClone: workflow.SkipClone,
		RunsOn:    make([]string, len(workflow.RunsOn)),
	}

	for k, v := range workflow.Labels {
		clone.Labels[k] = v
	}

	copy(clone.RunsOn, workflow.RunsOn)

	for i, container := range workflow.Clone.ContainerList {
		clone.Clone.ContainerList[i] = g.cloneContainer(container)
	}

	for i, container := range workflow.Steps.ContainerList {
		clone.Steps.ContainerList[i] = g.cloneContainer(container)
	}

	for i, container := range workflow.Services.ContainerList {
		clone.Services.ContainerList[i] = g.cloneContainer(container)
	}

	return clone
}

// cloneContainer creates a deep copy of a container.
func (g *Generator) cloneContainer(container *yaml_types.Container) *yaml_types.Container {
	clone := &yaml_types.Container{
		Name:        container.Name,
		Image:       container.Image,
		Pull:        container.Pull,
		Detach:      container.Detach,
		Privileged:  container.Privileged,
		When:        container.When,
		Environment: make(map[string]any),
		Secrets:     make([]string, len(container.Secrets)),
		Commands:    make([]string, len(container.Commands)),
		Entrypoint:  make([]string, len(container.Entrypoint)),
		Settings:    make(map[string]any),
		Volumes:     make([]*yaml_types.Volume, len(container.Volumes)),
		Networks:    make([]*yaml_types.Network, len(container.Networks)),
		DependsOn:   container.DependsOn,
		Failure:     container.Failure,
		Group:       container.Group,
	}

	for k, v := range container.Environment {
		clone.Environment[k] = v
	}

	for k, v := range container.Settings {
		clone.Settings[k] = v
	}

	copy(clone.Secrets, container.Secrets)
	copy(clone.Commands, container.Commands)
	copy(clone.Entrypoint, container.Entrypoint)

	for i, volume := range container.Volumes {
		clone.Volumes[i] = &yaml_types.Volume{
			Name: volume.Name,
			Driver: volume.Driver,
			DriverOpts: make(map[string]string),
		}
		for k, v := range volume.DriverOpts {
			clone.Volumes[i].DriverOpts[k] = v
		}
	}

	for i, network := range container.Networks {
		clone.Networks[i] = &yaml_types.Network{
			Name: network.Name,
			Driver: network.Driver,
			DriverOpts: make(map[string]string),
		}
		for k, v := range network.DriverOpts {
			clone.Networks[i].DriverOpts[k] = v
		}
	}

	return clone
}

// applyAxis applies matrix axis values to a workflow.
func (g *Generator) applyAxis(workflow *yaml_types.Workflow, axis Axis) {
	axisName := axis.String()
	if workflow.Labels == nil {
		workflow.Labels = make(map[string]string)
	}
	workflow.Labels["matrix"] = axisName

	for k, v := range axis {
		workflow.Labels[fmt.Sprintf("matrix.%s", k)] = v
	}

	for _, containers := range [][]*yaml_types.Container{
		workflow.Clone.ContainerList,
		workflow.Steps.ContainerList,
		workflow.Services.ContainerList,
	} {
		for _, container := range containers {
			if container.Environment == nil {
				container.Environment = make(map[string]any)
			}
			for k, v := range axis {
				container.Environment[fmt.Sprintf("MATRIX_%s", strings.ToUpper(k))] = v
			}
		}
	}
}
