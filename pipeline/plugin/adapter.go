// Copyright 2025 Woodpecker Authors
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

package plugin

import (
	"context"
	"fmt"
)

// PluginV1 is the old plugin interface (v1).
type PluginV1 interface {
	// Exec executes the plugin with given settings.
	Exec(settings map[string]any) error
}

// PluginV2 is the new plugin interface (v2).
type PluginV2 interface {
	// Name returns the plugin name.
	Name() string
	// Version returns the plugin version.
	Version() string
	// Validate validates the plugin configuration.
	Validate(ctx context.Context, settings map[string]any) error
	// Execute executes the plugin.
	Execute(ctx context.Context, settings map[string]any) (*Result, error)
}

// Result represents the result of a plugin v2 execution.
type Result struct {
	Success bool
	Message string
	Output  map[string]any
	Logs    []string
}

// Adapter adapts PluginV1 to PluginV2.
type Adapter struct {
	v1Plugin PluginV1
	name     string
	version  string
}

// NewAdapter creates a new PluginV1 to PluginV2 adapter.
func NewAdapter(v1 PluginV1, name, version string) *Adapter {
	return &Adapter{
		v1Plugin: v1,
		name:     name,
		version:  version,
	}
}

// Name returns the plugin name.
func (a *Adapter) Name() string {
	return a.name
}

// Version returns the plugin version.
func (a *Adapter) Version() string {
	return a.version
}

// Validate validates the plugin configuration (no-op for v1 plugins).
func (a *Adapter) Validate(_ context.Context, _ map[string]any) error {
	return nil
}

// Execute executes the plugin by delegating to the v1 Exec method.
func (a *Adapter) Execute(_ context.Context, settings map[string]any) (*Result, error) {
	err := a.v1Plugin.Exec(settings)
	if err != nil {
		return &Result{
			Success: false,
			Message: err.Error(),
		}, err
	}
	return &Result{
		Success: true,
		Message: "Plugin executed successfully",
	}, nil
}

// PluginRegistry is a registry for both v1 and v2 plugins.
type PluginRegistry struct {
	v2Plugins map[string]PluginV2
}

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		v2Plugins: make(map[string]PluginV2),
	}
}

// RegisterV1 registers a v1 plugin with the registry by adapting it to v2.
func (r *PluginRegistry) RegisterV1(name, version string, plugin PluginV1) {
	adapter := NewAdapter(plugin, name, version)
	r.v2Plugins[name] = adapter
}

// RegisterV2 registers a v2 plugin with the registry.
func (r *PluginRegistry) RegisterV2(plugin PluginV2) {
	r.v2Plugins[plugin.Name()] = plugin
}

// Get retrieves a plugin by name.
func (r *PluginRegistry) Get(name string) (PluginV2, bool) {
	plugin, ok := r.v2Plugins[name]
	return plugin, ok
}

// List returns all registered plugins.
func (r *PluginRegistry) List() []PluginV2 {
	plugins := make([]PluginV2, 0, len(r.v2Plugins))
	for _, plugin := range r.v2Plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// ExamplePluginV1 is an example of a v1 plugin.
type ExamplePluginV1 struct{}

// Exec executes the example v1 plugin.
func (p *ExamplePluginV1) Exec(settings map[string]any) error {
	fmt.Println("ExamplePluginV1 executing with settings:", settings)
	return nil
}

// ExamplePluginV2 is an example of a v2 plugin.
type ExamplePluginV2 struct{}

// Name returns the plugin name.
func (p *ExamplePluginV2) Name() string {
	return "example-v2"
}

// Version returns the plugin version.
func (p *ExamplePluginV2) Version() string {
	return "2.0.0"
}

// Validate validates the plugin configuration.
func (p *ExamplePluginV2) Validate(_ context.Context, settings map[string]any) error {
	if _, ok := settings["required"]; !ok {
		return fmt.Errorf("required setting missing")
	}
	return nil
}

// Execute executes the plugin.
func (p *ExamplePluginV2) Execute(_ context.Context, settings map[string]any) (*Result, error) {
	fmt.Println("ExamplePluginV2 executing with settings:", settings)
	return &Result{
		Success: true,
		Message: "Success",
		Output: map[string]any{
			"result": "done",
		},
	}, nil
}
