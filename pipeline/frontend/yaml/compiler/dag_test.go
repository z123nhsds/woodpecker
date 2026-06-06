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

	backend_types "go.woodpecker-ci.org/woodpecker/v3/pipeline/backend/types"
	"go.woodpecker-ci.org/woodpecker/v3/pipeline/frontend/yaml/constraint"
)

func TestConvertDAGToStages(t *testing.T) {
	steps := map[string]*dagCompilerStep{
		"step1": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "step3"}},
		},
		"step2": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "step1"}},
		},
		"step3": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "step2"}},
		},
	}
	_, err := convertDAGToStages(steps)
	assert.ErrorIs(t, err, &ErrStepDependencyCycle{})

	steps = map[string]*dagCompilerStep{
		"step1": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "step2"}},
		},
		"step2": {
			step: &backend_types.Step{},
		},
	}
	_, err = convertDAGToStages(steps)
	assert.NoError(t, err)

	steps = map[string]*dagCompilerStep{
		"a": {
			step: &backend_types.Step{},
		},
		"b": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "a"}},
		},
		"c": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "a"}},
		},
		"d": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "b"}, {Name: "c"}},
		},
	}
	_, err = convertDAGToStages(steps)
	assert.NoError(t, err)

	steps = map[string]*dagCompilerStep{
		"step1": {
			step:      &backend_types.Step{},
			dependsOn: constraint.DependsOn{{Name: "not-existing-step"}},
		},
	}
	_, err = convertDAGToStages(steps)
	assert.ErrorIs(t, err, &ErrStepMissingDependency{})

	steps = map[string]*dagCompilerStep{
		"echo env": {
			position: 0,
			name:     "echo env",
			step: &backend_types.Step{
				UUID:  "01HJDPEW6R7J0JBE3F1T7Q0TYX",
				Type:  "commands",
				Name:  "echo env",
				Image: "bash",
			},
		},
		"echo 1": {
			position:  1,
			name:      "echo 1",
			dependsOn: constraint.DependsOn{{Name: "echo env"}, {Name: "echo 2"}},
			step: &backend_types.Step{
				UUID:  "01HJDPF770QGRZER8RF79XVS4M",
				Type:  "commands",
				Name:  "echo 1",
				Image: "bash",
			},
		},
		"echo 2": {
			position: 2,
			name:     "echo 2",
			step: &backend_types.Step{
				UUID:  "01HJDPFF5RMEYZW0YTGR1Y1ZR0",
				Type:  "commands",
				Name:  "echo 2",
				Image: "bash",
			},
		},
	}
	stages, err := convertDAGToStages(steps)
	assert.NoError(t, err)
	assert.EqualValues(t, []*backend_types.Stage{{
		Steps: []*backend_types.Step{{
			UUID:  "01HJDPEW6R7J0JBE3F1T7Q0TYX",
			Type:  "commands",
			Name:  "echo env",
			Image: "bash",
		}, {
			UUID:  "01HJDPFF5RMEYZW0YTGR1Y1ZR0",
			Type:  "commands",
			Name:  "echo 2",
			Image: "bash",
		}},
	}, {
		Steps: []*backend_types.Step{{
			UUID:  "01HJDPF770QGRZER8RF79XVS4M",
			Type:  "commands",
			Name:  "echo 1",
			Image: "bash",
		}},
	}}, stages)
}

func TestConvertDAGToStagesTableDriven(t *testing.T) {
	testCases := []struct {
		name           string
		steps          map[string]*dagCompilerStep
		expectedStages int
		expectError    bool
		errorType      error
	}{
		{
			name: "empty steps map",
			steps: map[string]*dagCompilerStep{},
			expectedStages: 0,
			expectError: false,
		},
		{
			name: "single step with no dependencies",
			steps: map[string]*dagCompilerStep{
				"build": {
					position: 0,
					name:     "build",
					step:     &backend_types.Step{Name: "build"},
				},
			},
			expectedStages: 1,
			expectError: false,
		},
		{
			name: "linear chain of steps",
			steps: map[string]*dagCompilerStep{
				"checkout": {
					position: 0,
					name:     "checkout",
					step:     &backend_types.Step{Name: "checkout"},
				},
				"build": {
					position: 1,
					name:     "build",
					step:     &backend_types.Step{Name: "build"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}},
				},
				"test": {
					position: 2,
					name:     "test",
					step:     &backend_types.Step{Name: "test"},
					dependsOn: constraint.DependsOn{{Name: "build"}},
				},
				"deploy": {
					position: 3,
					name:     "deploy",
					step:     &backend_types.Step{Name: "deploy"},
					dependsOn: constraint.DependsOn{{Name: "test"}},
				},
			},
			expectedStages: 4,
			expectError: false,
		},
		{
			name: "diamond dependency graph",
			steps: map[string]*dagCompilerStep{
				"a": {
					position: 0,
					name:     "a",
					step:     &backend_types.Step{Name: "a"},
				},
				"b": {
					position: 1,
					name:     "b",
					step:     &backend_types.Step{Name: "b"},
					dependsOn: constraint.DependsOn{{Name: "a"}},
				},
				"c": {
					position: 2,
					name:     "c",
					step:     &backend_types.Step{Name: "c"},
					dependsOn: constraint.DependsOn{{Name: "a"}},
				},
				"d": {
					position: 3,
					name:     "d",
					step:     &backend_types.Step{Name: "d"},
					dependsOn: constraint.DependsOn{{Name: "b"}, {Name: "c"}},
				},
			},
			expectedStages: 3,
			expectError: false,
		},
		{
			name: "parallel steps with no dependencies",
			steps: map[string]*dagCompilerStep{
				"lint": {
					position: 0,
					name:     "lint",
					step:     &backend_types.Step{Name: "lint"},
					dependsOn: constraint.DependsOn{},
				},
				"test": {
					position: 1,
					name:     "test",
					step:     &backend_types.Step{Name: "test"},
					dependsOn: constraint.DependsOn{},
				},
				"build": {
					position: 2,
					name:     "build",
					step:     &backend_types.Step{Name: "build"},
					dependsOn: constraint.DependsOn{},
				},
			},
			expectedStages: 1,
			expectError: false,
		},
		{
			name: "mix of optional and required dependencies",
			steps: map[string]*dagCompilerStep{
				"checkout": {
					position: 0,
					name:     "checkout",
					step:     &backend_types.Step{Name: "checkout"},
				},
				"lint": {
					position: 1,
					name:     "lint",
					step:     &backend_types.Step{Name: "lint"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}},
				},
				"test": {
					position: 2,
					name:     "test",
					step:     &backend_types.Step{Name: "test"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}, {Name: "lint", Optional: true}},
				},
				"build": {
					position: 3,
					name:     "build",
					step:     &backend_types.Step{Name: "build"},
					dependsOn: constraint.DependsOn{{Name: "test"}},
				},
			},
			expectedStages: 4,
			expectError: false,
		},
		{
			name: "dependency cycle",
			steps: map[string]*dagCompilerStep{
				"a": {
					name:     "a",
					step:     &backend_types.Step{Name: "a"},
					dependsOn: constraint.DependsOn{{Name: "b"}},
				},
				"b": {
					name:     "b",
					step:     &backend_types.Step{Name: "b"},
					dependsOn: constraint.DependsOn{{Name: "c"}},
				},
				"c": {
					name:     "c",
					step:     &backend_types.Step{Name: "c"},
					dependsOn: constraint.DependsOn{{Name: "a"}},
				},
			},
			expectError: true,
			errorType: &ErrStepDependencyCycle{},
		},
		{
			name: "self-dependency cycle",
			steps: map[string]*dagCompilerStep{
				"a": {
					name:     "a",
					step:     &backend_types.Step{Name: "a"},
					dependsOn: constraint.DependsOn{{Name: "a"}},
				},
			},
			expectError: true,
			errorType: &ErrStepDependencyCycle{},
		},
		{
			name: "missing required dependency",
			steps: map[string]*dagCompilerStep{
				"build": {
					name:     "build",
					step:     &backend_types.Step{Name: "build"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}},
				},
			},
			expectError: true,
			errorType: &ErrStepMissingDependency{},
		},
		{
			name: "complex multi-level graph",
			steps: map[string]*dagCompilerStep{
				"setup": {
					position: 0,
					name:     "setup",
					step:     &backend_types.Step{Name: "setup"},
				},
				"checkout": {
					position: 1,
					name:     "checkout",
					step:     &backend_types.Step{Name: "checkout"},
					dependsOn: constraint.DependsOn{{Name: "setup"}},
				},
				"lint": {
					position: 2,
					name:     "lint",
					step:     &backend_types.Step{Name: "lint"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}},
				},
				"test-unit": {
					position: 3,
					name:     "test-unit",
					step:     &backend_types.Step{Name: "test-unit"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}},
				},
				"test-integration": {
					position: 4,
					name:     "test-integration",
					step:     &backend_types.Step{Name: "test-integration"},
					dependsOn: constraint.DependsOn{{Name: "checkout"}, {Name: "test-unit"}},
				},
				"build": {
					position: 5,
					name:     "build",
					step:     &backend_types.Step{Name: "build"},
					dependsOn: constraint.DependsOn{{Name: "lint"}, {Name: "test-unit"}},
				},
				"deploy": {
					position: 6,
					name:     "deploy",
					step:     &backend_types.Step{Name: "deploy"},
					dependsOn: constraint.DependsOn{{Name: "build"}, {Name: "test-integration"}},
				},
			},
			expectedStages: 5,
			expectError: false,
		},
		{
			name: "steps with empty dependsOn slice",
			steps: map[string]*dagCompilerStep{
				"step1": {
					position: 0,
					name:     "step1",
					step:     &backend_types.Step{Name: "step1"},
					dependsOn: constraint.DependsOn{},
				},
				"step2": {
					position: 1,
					name:     "step2",
					step:     &backend_types.Step{Name: "step2"},
					dependsOn: constraint.DependsOn{},
				},
			},
			expectedStages: 1,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the test steps to avoid issues with parallel tests
			testSteps := make(map[string]*dagCompilerStep, len(tc.steps))
			for k, v := range tc.steps {
				testSteps[k] = v
			}

			stages, err := convertDAGToStages(testSteps)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != nil {
					assert.ErrorIs(t, err, tc.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, stages, tc.expectedStages)
			}
		})
	}
}

func TestOptionalStepDependency(t *testing.T) {
	t.Run("missing optional step dep is dropped", func(t *testing.T) {
		steps := map[string]*dagCompilerStep{
			"build": {
				position: 0,
				name:     "build",
				step:     &backend_types.Step{Name: "build"},
			},
			"deploy": {
				position: 1,
				name:     "deploy",
				step:     &backend_types.Step{Name: "deploy"},
				dependsOn: constraint.DependsOn{
					{Name: "build"},
					{Name: "lint", Optional: true},
				},
			},
		}
		stages, err := convertDAGToStages(steps)
		assert.NoError(t, err)
		assert.Len(t, stages, 2, "should produce 2 stages (build then deploy)")
	})

	t.Run("missing required step dep still errors", func(t *testing.T) {
		steps := map[string]*dagCompilerStep{
			"deploy": {
				name: "deploy",
				step: &backend_types.Step{Name: "deploy"},
				dependsOn: constraint.DependsOn{
					{Name: "build"},
				},
			},
		}
		_, err := convertDAGToStages(steps)
		assert.ErrorIs(t, err, &ErrStepMissingDependency{})
	})

	t.Run("present optional step dep is kept", func(t *testing.T) {
		steps := map[string]*dagCompilerStep{
			"build": {
				position: 0,
				name:     "build",
				step:     &backend_types.Step{Name: "build"},
			},
			"lint": {
				position: 1,
				name:     "lint",
				step:     &backend_types.Step{Name: "lint"},
			},
			"deploy": {
				position: 2,
				name:     "deploy",
				step:     &backend_types.Step{Name: "deploy"},
				dependsOn: constraint.DependsOn{
					{Name: "build"},
					{Name: "lint", Optional: true},
				},
			},
		}
		stages, err := convertDAGToStages(steps)
		assert.NoError(t, err)
		assert.Len(t, stages, 2, "build+lint in stage 1, deploy in stage 2")
		assert.Len(t, stages[0].Steps, 2)
		assert.Len(t, stages[1].Steps, 1)
		assert.Equal(t, "deploy", stages[1].Steps[0].Name)
	})
}

func TestIsDag(t *testing.T) {
	testCases := []struct {
		name     string
		steps    []*dagCompilerStep
		expected bool
	}{
		{
			name: "no dependencies - not a DAG",
			steps: []*dagCompilerStep{
				{
					step: &backend_types.Step{},
				},
			},
			expected: false,
		},
		{
			name: "has dependsOn field - is a DAG",
			steps: []*dagCompilerStep{
				{
					step:      &backend_types.Step{},
					dependsOn: constraint.DependsOn{},
				},
			},
			expected: true,
		},
		{
			name: "mix of steps with and without dependencies - is a DAG",
			steps: []*dagCompilerStep{
				{
					step: &backend_types.Step{},
				},
				{
					step:      &backend_types.Step{},
					dependsOn: constraint.DependsOn{{Name: "step1"}},
				},
			},
			expected: true,
		},
		{
			name: "multiple steps with empty dependsOn - is a DAG",
			steps: []*dagCompilerStep{
				{
					step:      &backend_types.Step{},
					dependsOn: constraint.DependsOn{},
				},
				{
					step:      &backend_types.Step{},
					dependsOn: constraint.DependsOn{},
				},
			},
			expected: true,
		},
		{
			name: "empty steps slice - not a DAG",
			steps: []*dagCompilerStep{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newDAGCompiler(tc.steps)
			assert.Equal(t, tc.expected, c.isDAG())
		})
	}
}

func TestCompile(t *testing.T) {
	testCases := []struct {
		name           string
		steps          []*dagCompilerStep
		expectedStages int
		expectError    bool
	}{
		{
			name: "compile sequential steps",
			steps: []*dagCompilerStep{
				{
					position: 0,
					name:     "step1",
					step:     &backend_types.Step{Name: "step1"},
				},
				{
					position: 1,
					name:     "step2",
					step:     &backend_types.Step{Name: "step2"},
				},
			},
			expectedStages: 2,
			expectError: false,
		},
		{
			name: "compile DAG steps",
			steps: []*dagCompilerStep{
				{
					position: 0,
					name:     "a",
					step:     &backend_types.Step{Name: "a"},
				},
				{
					position: 1,
					name:     "b",
					step:     &backend_types.Step{Name: "b"},
					dependsOn: constraint.DependsOn{{Name: "a"}},
				},
			},
			expectedStages: 2,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newDAGCompiler(tc.steps)
			stages, err := c.compile()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, stages, tc.expectedStages)
			}
		})
	}
}

