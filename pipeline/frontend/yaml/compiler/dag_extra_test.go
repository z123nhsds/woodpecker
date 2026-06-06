// Copyright 2026 Woodpecker Authors
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

package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	backend_types "go.woodpecker-ci.org/woodpecker/v3/pipeline/backend/types"
	"go.woodpecker-ci.org/woodpecker/v3/pipeline/frontend/yaml/constraint"
)

func TestDagCompilerCompile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		steps       []*dagCompilerStep
		wantStages  int
		wantErr     bool
		wantErrType error
	}{
		{
			name: "empty steps",
			steps: []*dagCompilerStep{},
			wantStages: 0,
		},
		{
			name: "single step sequential",
			steps: []*dagCompilerStep{
				{name: "build", step: &backend_types.Step{Name: "build"}},
			},
			wantStages: 1,
		},
		{
			name: "multiple steps sequential (no depends_on)",
			steps: []*dagCompilerStep{
				{name: "build", position: 0, step: &backend_types.Step{Name: "build"}},
				{name: "test", position: 1, step: &backend_types.Step{Name: "test"}},
				{name: "deploy", position: 2, step: &backend_types.Step{Name: "deploy"}},
			},
			wantStages: 3,
		},
		{
			name: "single step with empty depends_on triggers DAG mode",
			steps: []*dagCompilerStep{
				{name: "build", position: 0, step: &backend_types.Step{Name: "build"}, dependsOn: constraint.DependsOn{}},
			},
			wantStages: 1,
		},
		{
			name: "linear chain a->b->c",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "b"}}},
			},
			wantStages: 3,
		},
		{
			name: "diamond dependency a->(b,c)->d",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
				{name: "d", position: 3, step: &backend_types.Step{Name: "d"}, dependsOn: constraint.DependsOn{{Name: "b"}, {Name: "c"}}},
			},
			wantStages: 3,
		},
		{
			name: "fan-out: a splits to b and c",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
			},
			wantStages: 2,
		},
		{
			name: "fan-in: a and b merge into c",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}},
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "a"}, {Name: "b"}}},
			},
			wantStages: 2,
		},
		{
			name: "self-loop cycle",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
			},
			wantErr:     true,
			wantErrType: &ErrStepDependencyCycle{},
		},
		{
			name: "two-node cycle a<->b",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}, dependsOn: constraint.DependsOn{{Name: "b"}}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
			},
			wantErr:     true,
			wantErrType: &ErrStepDependencyCycle{},
		},
		{
			name: "three-node cycle a->b->c->a",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}, dependsOn: constraint.DependsOn{{Name: "c"}}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "b"}}},
			},
			wantErr:     true,
			wantErrType: &ErrStepDependencyCycle{},
		},
		{
			name: "missing dependency",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}, dependsOn: constraint.DependsOn{{Name: "nonexistent"}}},
			},
			wantErr:     true,
			wantErrType: &ErrStepMissingDependency{},
		},
		{
			name: "optional missing dependency is dropped",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}, {Name: "nonexistent", Optional: true}}},
			},
			wantStages: 2,
		},
		{
			name: "mixed optional and required deps",
			steps: []*dagCompilerStep{
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "b"}, {Name: "d", Optional: true}}},
			},
			wantStages: 3,
		},
		{
			name: "steps sorted by position within stage",
			steps: []*dagCompilerStep{
				{name: "c", position: 2, step: &backend_types.Step{Name: "c"}},
				{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
				{name: "b", position: 1, step: &backend_types.Step{Name: "b"}},
			},
			wantStages: 3,
		},
		{
			name: "complex graph with multiple levels",
			steps: []*dagCompilerStep{
				{name: "clone", position: 0, step: &backend_types.Step{Name: "clone"}},
				{name: "lint", position: 1, step: &backend_types.Step{Name: "lint"}, dependsOn: constraint.DependsOn{{Name: "clone"}}},
				{name: "build", position: 2, step: &backend_types.Step{Name: "build"}, dependsOn: constraint.DependsOn{{Name: "clone"}}},
				{name: "test", position: 3, step: &backend_types.Step{Name: "test"}, dependsOn: constraint.DependsOn{{Name: "build"}}},
				{name: "deploy", position: 4, step: &backend_types.Step{Name: "deploy"}, dependsOn: constraint.DependsOn{{Name: "test"}, {Name: "lint"}}},
			},
			wantStages: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newDAGCompiler(tt.steps)
			stages, err := c.compile()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}
			assert.NoError(t, err)
			assert.Len(t, stages, tt.wantStages)

			switch tt.name {
			case "single step sequential":
				assert.Equal(t, "build", stages[0].Steps[0].Name)
			case "multiple steps sequential (no depends_on)":
				assert.Equal(t, "build", stages[0].Steps[0].Name)
				assert.Equal(t, "test", stages[1].Steps[0].Name)
				assert.Equal(t, "deploy", stages[2].Steps[0].Name)
			case "linear chain a->b->c":
				assert.Equal(t, "a", stages[0].Steps[0].Name)
				assert.Equal(t, "b", stages[1].Steps[0].Name)
				assert.Equal(t, "c", stages[2].Steps[0].Name)
			case "diamond dependency a->(b,c)->d":
				assert.Equal(t, "a", stages[0].Steps[0].Name)
				assert.Len(t, stages[1].Steps, 2)
				assert.Equal(t, "d", stages[2].Steps[0].Name)
			case "fan-out: a splits to b and c":
				assert.Equal(t, "a", stages[0].Steps[0].Name)
				assert.Len(t, stages[1].Steps, 2)
			case "fan-in: a and b merge into c":
				assert.Len(t, stages[0].Steps, 2)
				assert.Equal(t, "c", stages[1].Steps[0].Name)
			case "steps sorted by position within stage":
				assert.Equal(t, "a", stages[0].Steps[0].Name)
				assert.Equal(t, "b", stages[1].Steps[0].Name)
				assert.Equal(t, "c", stages[2].Steps[0].Name)
			}
		})
	}
}

func TestDagCompilerCompileSequence(t *testing.T) {
	t.Parallel()

	steps := []*dagCompilerStep{
		{name: "step1", position: 0, step: &backend_types.Step{Name: "step1", Image: "alpine"}},
		{name: "step2", position: 1, step: &backend_types.Step{Name: "step2", Image: "bash"}},
	}
	c := newDAGCompiler(steps)
	stages, err := c.compileSequence()

	assert.NoError(t, err)
	assert.Len(t, stages, 2)
	assert.Equal(t, "step1", stages[0].Steps[0].Name)
	assert.Equal(t, "alpine", stages[0].Steps[0].Image)
	assert.Equal(t, "step2", stages[1].Steps[0].Name)
}

func TestDagCompilerCompileByDependsOn(t *testing.T) {
	t.Parallel()

	steps := []*dagCompilerStep{
		{name: "a", position: 0, step: &backend_types.Step{Name: "a"}},
		{name: "b", position: 1, step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
	}
	c := newDAGCompiler(steps)
	stages, err := c.compileByDependsOn()

	assert.NoError(t, err)
	assert.Len(t, stages, 2)
	assert.Equal(t, "a", stages[0].Steps[0].Name)
	assert.Equal(t, "b", stages[1].Steps[0].Name)
}

func TestDfsVisit(t *testing.T) {
	t.Parallel()

	steps := map[string]*dagCompilerStep{
		"a": {name: "a", step: &backend_types.Step{Name: "a"}, dependsOn: constraint.DependsOn{{Name: "b"}}},
		"b": {name: "b", step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "c"}}},
		"c": {name: "c", step: &backend_types.Step{Name: "c"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
	}

	t.Run("detects cycle", func(t *testing.T) {
		t.Parallel()
		visited := make(map[string]struct{})
		err := dfsVisit(steps, "a", visited, []string{})
		assert.Error(t, err)
		assert.ErrorIs(t, err, &ErrStepDependencyCycle{})
	})

	t.Run("no cycle is fine", func(t *testing.T) {
		t.Parallel()
		stepsNoCycle := map[string]*dagCompilerStep{
			"a": {name: "a", step: &backend_types.Step{Name: "a"}},
			"b": {name: "b", step: &backend_types.Step{Name: "b"}, dependsOn: constraint.DependsOn{{Name: "a"}}},
		}
		visited := make(map[string]struct{})
		err := dfsVisit(stepsNoCycle, "b", visited, []string{})
		assert.NoError(t, err)
	})
}

func TestAllDependenciesSatisfied(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		step       *dagCompilerStep
		addedSteps map[string]struct{}
		want       bool
	}{
		{
			name: "no deps",
			step: &dagCompilerStep{name: "a", dependsOn: constraint.DependsOn{}},
			addedSteps: map[string]struct{}{},
			want: true,
		},
		{
			name: "all deps satisfied",
			step: &dagCompilerStep{name: "c", dependsOn: constraint.DependsOn{{Name: "a"}, {Name: "b"}}},
			addedSteps: map[string]struct{}{"a": {}, "b": {}},
			want: true,
		},
		{
			name: "one dep missing",
			step: &dagCompilerStep{name: "c", dependsOn: constraint.DependsOn{{Name: "a"}, {Name: "b"}}},
			addedSteps: map[string]struct{}{"a": {}},
			want: false,
		},
		{
			name: "all deps missing",
			step: &dagCompilerStep{name: "c", dependsOn: constraint.DependsOn{{Name: "a"}, {Name: "b"}}},
			addedSteps: map[string]struct{}{},
			want: false,
		},
		{
			name: "nil step dependsOn",
			step: &dagCompilerStep{name: "a", dependsOn: nil},
			addedSteps: map[string]struct{}{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := allDependenciesSatisfied(tt.step, tt.addedSteps)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsDag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps []*dagCompilerStep
		want  bool
	}{
		{
			name:  "nil depends_on means sequential",
			steps: []*dagCompilerStep{{step: &backend_types.Step{}}},
			want:  false,
		},
		{
			name:  "empty non-nil depends_on means DAG",
			steps: []*dagCompilerStep{{step: &backend_types.Step{}, dependsOn: constraint.DependsOn{}}},
			want:  true,
		},
		{
			name:  "non-empty depends_on means DAG",
			steps: []*dagCompilerStep{{step: &backend_types.Step{}, dependsOn: constraint.DependsOn{{Name: "other"}}}},
			want:  true,
		},
		{
			name: "mixed: any step with non-nil depends_on makes it DAG",
			steps: []*dagCompilerStep{
				{step: &backend_types.Step{}},
				{step: &backend_types.Step{}, dependsOn: constraint.DependsOn{}},
			},
			want: true,
		},
		{
			name: "all nil depends_on means sequential",
			steps: []*dagCompilerStep{
				{step: &backend_types.Step{}},
				{step: &backend_types.Step{}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newDAGCompiler(tt.steps)
			assert.Equal(t, tt.want, c.isDAG())
		})
	}
}

func TestConvertDAGToStagesStageOrder(t *testing.T) {
	t.Parallel()

	steps := map[string]*dagCompilerStep{
		"clone": {
			position: 0,
			name:     "clone",
			step:     &backend_types.Step{Name: "clone"},
		},
		"lint": {
			position: 1,
			name:     "lint",
			step:     &backend_types.Step{Name: "lint"},
			dependsOn: constraint.DependsOn{{Name: "clone"}},
		},
		"build": {
			position: 2,
			name:     "build",
			step:     &backend_types.Step{Name: "build"},
			dependsOn: constraint.DependsOn{{Name: "clone"}},
		},
		"test": {
			position: 3,
			name:     "test",
			step:     &backend_types.Step{Name: "test"},
			dependsOn: constraint.DependsOn{{Name: "build"}},
		},
		"deploy": {
			position: 4,
			name:     "deploy",
			step:     &backend_types.Step{Name: "deploy"},
			dependsOn: constraint.DependsOn{{Name: "test"}, {Name: "lint"}},
		},
	}

	stages, err := convertDAGToStages(steps)
	assert.NoError(t, err)
	assert.Len(t, stages, 4)

	assert.Len(t, stages[0].Steps, 1)
	assert.Equal(t, "clone", stages[0].Steps[0].Name)

	assert.Len(t, stages[1].Steps, 2)
	assert.Equal(t, "lint", stages[1].Steps[0].Name)
	assert.Equal(t, "build", stages[1].Steps[1].Name)

	assert.Len(t, stages[2].Steps, 1)
	assert.Equal(t, "test", stages[2].Steps[0].Name)

	assert.Len(t, stages[3].Steps, 1)
	assert.Equal(t, "deploy", stages[3].Steps[0].Name)
}

func TestConvertDAGToStagesCrossDependency(t *testing.T) {
	t.Parallel()

	steps := map[string]*dagCompilerStep{
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
		"e": {
			position: 4,
			name:     "e",
			step:     &backend_types.Step{Name: "e"},
			dependsOn: constraint.DependsOn{{Name: "b"}},
		},
	}

	stages, err := convertDAGToStages(steps)
	assert.NoError(t, err)
	assert.Len(t, stages, 3)

	assert.Len(t, stages[0].Steps, 1)
	assert.Equal(t, "a", stages[0].Steps[0].Name)

	assert.Len(t, stages[1].Steps, 2)
	assert.Equal(t, "b", stages[1].Steps[0].Name)
	assert.Equal(t, "c", stages[1].Steps[1].Name)

	assert.Len(t, stages[2].Steps, 2)
	assert.Equal(t, "d", stages[2].Steps[0].Name)
	assert.Equal(t, "e", stages[2].Steps[1].Name)
}
