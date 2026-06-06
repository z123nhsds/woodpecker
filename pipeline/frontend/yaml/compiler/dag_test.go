// Copyright 2023 Woodpecker Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backend_types "go.woodpecker-ci.org/woodpecker/v3/pipeline/backend/types"
	"go.woodpecker-ci.org/woodpecker/v3/pipeline/frontend/yaml/constraint"
)

func TestDAGCompilerCompile(t *testing.T) {
	tests := []struct {
		name         string
		steps        []*dagCompilerStep
		expected     [][]string
		expectedErr  error
		expectedIsDAG bool
	}{
		{
			name: "sequential steps without dependencies",
			steps: []*dagCompilerStep{
				newTestDAGStep(0, "lint"),
				newTestDAGStep(1, "test"),
				newTestDAGStep(2, "deploy"),
			},
			expected: [][]string{{"lint"}, {"test"}, {"deploy"}},
		},
		{
			name: "groups steps by dependency levels",
			steps: []*dagCompilerStep{
				newTestDAGStep(0, "build"),
				newTestDAGStep(1, "lint"),
				newTestDAGStep(2, "unit", constraint.DependsOn{{Name: "build"}}),
				newTestDAGStep(3, "publish", constraint.DependsOn{{Name: "build"}, {Name: "lint"}}),
				newTestDAGStep(4, "notify", constraint.DependsOn{{Name: "publish"}}),
			},
			expected:      [][]string{{"build", "lint"}, {"unit", "publish"}, {"notify"}},
			expectedIsDAG: true,
		},
		{
			name: "ignores optional missing dependencies",
			steps: []*dagCompilerStep{
				newTestDAGStep(0, "build"),
				newTestDAGStep(1, "deploy", constraint.DependsOn{{Name: "build"}, {Name: "notify", Optional: true}}),
			},
			expected:      [][]string{{"build"}, {"deploy"}},
			expectedIsDAG: true,
		},
		{
			name: "fails on missing required dependency",
			steps: []*dagCompilerStep{
				newTestDAGStep(0, "deploy", constraint.DependsOn{{Name: "build"}}),
			},
			expectedErr:  &ErrStepMissingDependency{},
			expectedIsDAG: true,
		},
		{
			name: "fails on dependency cycle",
			steps: []*dagCompilerStep{
				newTestDAGStep(0, "build", constraint.DependsOn{{Name: "deploy"}}),
				newTestDAGStep(1, "deploy", constraint.DependsOn{{Name: "build"}}),
			},
			expectedErr:  &ErrStepDependencyCycle{},
			expectedIsDAG: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler := newDAGCompiler(test.steps)
			assert.Equal(t, test.expectedIsDAG, compiler.isDAG())

			stages, err := compiler.compile()
			if test.expectedErr != nil {
				assert.ErrorIs(t, err, test.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, dagStageNames(stages))
		})
	}
}

func TestAllDependenciesSatisfied(t *testing.T) {
	tests := []struct {
		name      string
		step      *dagCompilerStep
		added     map[string]struct{}
		expected  bool
	}{
		{
			name:     "step without dependencies is ready",
			step:     newTestDAGStep(0, "build"),
			added:    map[string]struct{}{},
			expected: true,
		},
		{
			name:     "step waits for unmet dependency",
			step:     newTestDAGStep(0, "deploy", constraint.DependsOn{{Name: "build"}}),
			added:    map[string]struct{}{},
			expected: false,
		},
		{
			name:     "step is ready once all dependencies are added",
			step:     newTestDAGStep(0, "deploy", constraint.DependsOn{{Name: "build"}, {Name: "lint"}}),
			added:    map[string]struct{}{"build": {}, "lint": {}},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, allDependenciesSatisfied(test.step, test.added))
		})
	}
}

func newTestDAGStep(position int, name string, dependsOn ...constraint.DependsOn) *dagCompilerStep {
	step := &dagCompilerStep{
		position: position,
		name:     name,
		step: &backend_types.Step{
			Name: name,
		},
	}
	if len(dependsOn) > 0 {
		step.dependsOn = dependsOn[0]
	}
	return step
}

func dagStageNames(stages []*backend_types.Stage) [][]string {
	result := make([][]string, 0, len(stages))
	for _, stage := range stages {
		names := make([]string, 0, len(stage.Steps))
		for _, step := range stage.Steps {
			names = append(names, step.Name)
		}
		result = append(result, names)
	}
	return result
}
