package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.woodpecker-ci.org/woodpecker/v3/pipeline/frontend/metadata"
	"go.woodpecker-ci.org/woodpecker/v3/pipeline/frontend/yaml/constraint"
)

func TestBuildSteps_DependencyGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		pipelineEvent    metadata.Event
		branch           string
		yamls            []*YamlFile
		envs             map[string]string
		wantItemCount    int
		wantStageCounts  map[string]int
		wantDepNames     map[string][]string
		wantErrContains  string
		wantItemNames    []string
	}{
		{
			name:          "single_step_no_deps",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
    commands:
      - echo build
`)},
			},
			wantItemCount: 1,
			wantStageCounts: map[string]int{
				"build": 1,
			},
			wantItemNames: []string{"build"},
		},
		{
			name:          "linear_dep_chain_A_B_C",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "test", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: test
    image: scratch
depends_on: [build]
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on: [test]
`)},
			},
			wantItemCount: 3,
			wantDepNames: map[string][]string{
				"test":   {"build"},
				"deploy": {"test"},
			},
			wantItemNames: []string{"build", "test", "deploy"},
		},
		{
			name:          "diamond_dep_A_to_B_C_to_D",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "lint", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: lint
    image: scratch
depends_on: [build]
`)},
				{Name: "test", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: test
    image: scratch
depends_on: [build]
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on: [lint, test]
`)},
			},
			wantItemCount: 4,
			wantDepNames: map[string][]string{
				"lint":   {"build"},
				"test":   {"build"},
				"deploy": {"lint", "test"},
			},
			wantItemNames: []string{"build", "lint", "test", "deploy"},
		},
		{
			name:          "missing_required_workflow_dep_filtered",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on: [nonexistent]
`)},
			},
			wantItemCount: 0,
		},
		{
			name:          "missing_optional_workflow_dep_dropped",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on:
  - build
  - name: lint
    optional: true
`)},
			},
			wantItemCount: 2,
			wantDepNames: map[string][]string{
				"deploy": {"build"},
			},
			wantItemNames: []string{"build", "deploy"},
		},
		{
			name:          "present_optional_workflow_dep_kept",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "lint", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: lint
    image: scratch
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on:
  - build
  - name: lint
    optional: true
`)},
			},
			wantItemCount: 3,
			wantDepNames: map[string][]string{
				"deploy": {"build", "lint"},
			},
			wantItemNames: []string{"build", "lint", "deploy"},
		},
		{
			name:          "transitive_missing_dep_cascade",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "broken", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: broken
    image: scratch
depends_on: [nonexistent]
`)},
				{Name: "downstream", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: downstream
    image: scratch
depends_on: [broken]
`)},
				{Name: "standalone", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: standalone
    image: scratch
`)},
			},
			wantItemCount: 1,
			wantItemNames: []string{"standalone"},
		},
		{
			name:          "transitive_optional_dep_on_removed_workflow",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "broken", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: broken
    image: scratch
depends_on: [nonexistent]
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on:
  - name: broken
    optional: true
`)},
			},
			wantItemCount: 1,
			wantItemNames: []string{"deploy"},
		},
		{
			name:          "step_level_depends_on_linear",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "pipeline", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: test
    image: scratch
    depends_on: [build]
  - name: deploy
    image: scratch
    depends_on: [test]
`)},
			},
			wantItemCount: 1,
			wantStageCounts: map[string]int{
				"pipeline": 3,
			},
			wantItemNames: []string{"pipeline"},
		},
		{
			name:          "step_level_depends_on_diamond",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "pipeline", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: lint
    image: scratch
    depends_on: [build]
  - name: test
    image: scratch
    depends_on: [build]
  - name: deploy
    image: scratch
    depends_on: [lint, test]
`)},
			},
			wantItemCount: 1,
			wantStageCounts: map[string]int{
				"pipeline": 3,
			},
			wantItemNames: []string{"pipeline"},
		},
		{
			name:          "step_level_missing_dep_error",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "pipeline", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
    depends_on: [ghost-step]
`)},
			},
			wantItemCount:    0,
			wantErrContains:  "depends on unknown step",
		},
		{
			name:          "step_level_cycle_error",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "pipeline", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: step-a
    image: scratch
    depends_on: [step-b]
  - name: step-b
    image: scratch
    depends_on: [step-a]
`)},
			},
			wantItemCount:    0,
			wantErrContains:  "cycle detected",
		},
		{
			name:          "when_filter_removes_step_with_dep",
			pipelineEvent: metadata.EventPush,
			branch:        "develop",
			yamls: []*YamlFile{
				{Name: "pipeline", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: deploy-prod
    image: scratch
    depends_on: [build]
    when:
      branch: main
`)},
			},
			wantItemCount: 1,
			wantStageCounts: map[string]int{
				"pipeline": 1,
			},
			wantItemNames: []string{"pipeline"},
		},
		{
			name:          "when_filter_removes_dep_causes_filtered_dep_error",
			pipelineEvent: metadata.EventPush,
			branch:        "develop",
			yamls: []*YamlFile{
				{Name: "pipeline", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: lint
    image: scratch
    when:
      branch: main
  - name: deploy
    image: scratch
    depends_on: [lint]
`)},
			},
			wantItemCount:    0,
			wantErrContains:  "depends on step",
		},
		{
			name:          "matrix_expands_to_multiple_items",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true

matrix:
  GO_VERSION:
    - "1.22"
    - "1.23"

steps:
  - name: test
    image: golang:${GO_VERSION}
    commands:
      - go test
`)},
			},
			wantItemCount: 2,
			wantItemNames: []string{"build", "build"},
		},
		{
			name:          "empty_pipeline_no_steps",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "empty", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: filtered
    image: scratch
    when:
      branch: nonexistent
`)},
			},
			wantItemCount: 0,
		},
		{
			name:          "multi_pipeline_parallel_no_deps",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "lint", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: lint
    image: scratch
`)},
				{Name: "test", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: test
    image: scratch
`)},
			},
			wantItemCount: 2,
			wantItemNames: []string{"lint", "test"},
		},
		{
			name:          "global_env_substitution_in_dep_name",
			pipelineEvent: metadata.EventPush,
			branch:        "main",
			envs: map[string]string{
				"UPSTREAM": "build",
			},
			yamls: []*YamlFile{
				{Name: "build", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on:
  - ${UPSTREAM}
`)},
			},
			wantItemCount: 2,
			wantDepNames: map[string][]string{
				"deploy": {"build"},
			},
			wantItemNames: []string{"build", "deploy"},
		},
		{
			name:          "workflow_dep_on_filtered_out_workflow",
			pipelineEvent: metadata.EventTag,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "push-only", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: tag
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on: [push-only]
`)},
			},
			wantItemCount: 0,
		},
		{
			name:          "workflow_optional_dep_on_filtered_out_workflow",
			pipelineEvent: metadata.EventTag,
			branch:        "main",
			yamls: []*YamlFile{
				{Name: "push-only", Data: []byte(`
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
`)},
				{Name: "deploy", Data: []byte(`
when:
  event: tag
skip_clone: true
steps:
  - name: deploy
    image: scratch
depends_on:
  - name: push-only
    optional: true
`)},
			},
			wantItemCount: 1,
			wantItemNames: []string{"deploy"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &testMetadata{
				pipelineEvent: tc.pipelineEvent,
				branch:        tc.branch,
				repo:          "test-repo",
			}

			envs := tc.envs
			if envs == nil {
				envs = map[string]string{}
			}

			b := PipelineBuilder{
				GetWorkflowMetadata: m.GetWorkflowMetadata,
				Envs:                envs,
				RepoTrusted:         &metadata.TrustedConfiguration{},
				Yamls:               tc.yamls,
			}

			items, err := b.Build()

			if tc.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContains)
				return
			}

			require.NoError(t, err)
			assert.Len(t, items, tc.wantItemCount, "item count mismatch")

			if tc.wantItemNames != nil && tc.wantItemCount > 0 {
				gotNames := make([]string, 0, len(items))
				for _, item := range items {
					gotNames = append(gotNames, item.Workflow.Name)
				}
				assert.Equal(t, tc.wantItemNames, gotNames, "item names mismatch")
			}

			if tc.wantStageCounts != nil {
				for _, item := range items {
					if expected, ok := tc.wantStageCounts[item.Workflow.Name]; ok {
						assert.Len(t, item.Config.Stages, expected,
							"workflow %q stage count mismatch", item.Workflow.Name)
					}
				}
			}

			if tc.wantDepNames != nil {
				for _, item := range items {
					if expected, ok := tc.wantDepNames[item.Workflow.Name]; ok {
						gotNames := make([]string, 0, len(item.DependsOn))
						for _, dep := range item.DependsOn {
							gotNames = append(gotNames, dep.Name)
						}
						assert.Equal(t, expected, gotNames,
							"workflow %q dependency names mismatch", item.Workflow.Name)
					}
				}
			}
		})
	}
}

func TestBuildSteps_StepLevelDAGStageStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yamlData       string
		wantStageSteps [][]string
	}{
		{
			name: "sequential_steps_each_in_own_stage",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: test
    image: scratch
  - name: deploy
    image: scratch
`,
			wantStageSteps: [][]string{
				{"build"},
				{"test"},
				{"deploy"},
			},
		},
		{
			name: "diamond_dependency_three_stages",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: lint
    image: scratch
    depends_on: [build]
  - name: test
    image: scratch
    depends_on: [build]
  - name: deploy
    image: scratch
    depends_on: [lint, test]
`,
			wantStageSteps: [][]string{
				{"build"},
				{"lint", "test"},
				{"deploy"},
			},
		},
		{
			name: "fan_out_parallel",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: unit-test
    image: scratch
    depends_on: [build]
  - name: integration-test
    image: scratch
    depends_on: [build]
  - name: lint
    image: scratch
    depends_on: [build]
`,
			wantStageSteps: [][]string{
				{"build"},
				{"unit-test", "integration-test", "lint"},
			},
		},
		{
			name: "fan_in_multiple_deps",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: frontend
    image: scratch
  - name: backend
    image: scratch
  - name: integration
    image: scratch
    depends_on: [frontend, backend]
`,
			wantStageSteps: [][]string{
				{"frontend", "backend"},
				{"integration"},
			},
		},
		{
			name: "deep_chain_five_stages",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: checkout
    image: scratch
  - name: install
    image: scratch
    depends_on: [checkout]
  - name: build
    image: scratch
    depends_on: [install]
  - name: test
    image: scratch
    depends_on: [build]
  - name: deploy
    image: scratch
    depends_on: [test]
`,
			wantStageSteps: [][]string{
				{"checkout"},
				{"install"},
				{"build"},
				{"test"},
				{"deploy"},
			},
		},
		{
			name: "mixed_parallel_and_serial",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: build-a
    image: scratch
  - name: build-b
    image: scratch
  - name: test-a
    image: scratch
    depends_on: [build-a]
  - name: test-b
    image: scratch
    depends_on: [build-b]
  - name: deploy
    image: scratch
    depends_on: [test-a, test-b]
`,
			wantStageSteps: [][]string{
				{"build-a", "build-b"},
				{"test-a", "test-b"},
				{"deploy"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &testMetadata{
				pipelineEvent: metadata.EventPush,
				branch:        "main",
				repo:          "test-repo",
			}

			b := PipelineBuilder{
				GetWorkflowMetadata: m.GetWorkflowMetadata,
				RepoTrusted:         &metadata.TrustedConfiguration{},
				Yamls: []*YamlFile{
					{Name: "pipeline", Data: []byte(tc.yamlData)},
				},
			}

			items, err := b.Build()
			require.NoError(t, err)
			require.Len(t, items, 1, "expected exactly 1 item")

			stages := items[0].Config.Stages
			require.Len(t, stages, len(tc.wantStageSteps),
				"stage count mismatch: got %d, want %d", len(stages), len(tc.wantStageSteps))

			for stageIdx, wantStepNames := range tc.wantStageSteps {
				gotStepNames := make([]string, 0, len(stages[stageIdx].Steps))
				for _, step := range stages[stageIdx].Steps {
					gotStepNames = append(gotStepNames, step.Name)
				}
				assert.Equal(t, wantStepNames, gotStepNames,
					"stage %d step names mismatch", stageIdx)
			}
		})
	}
}

func TestBuildSteps_StepLevelOptionalDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		yamlData       string
		wantErr        bool
		wantStageSteps [][]string
	}{
		{
			name: "missing_optional_step_dep_succeeds",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: deploy
    image: scratch
    depends_on:
      - build
      - name: lint
        optional: true
`,
			wantErr: false,
			wantStageSteps: [][]string{
				{"build"},
				{"deploy"},
			},
		},
		{
			name: "present_optional_step_dep_kept",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: build
    image: scratch
  - name: lint
    image: scratch
  - name: deploy
    image: scratch
    depends_on:
      - build
      - name: lint
        optional: true
`,
			wantErr: false,
			wantStageSteps: [][]string{
				{"build", "lint"},
				{"deploy"},
			},
		},
		{
			name: "missing_required_step_dep_errors",
			yamlData: `
when:
  event: push
skip_clone: true
steps:
  - name: deploy
    image: scratch
    depends_on: [ghost]
`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &testMetadata{
				pipelineEvent: metadata.EventPush,
				branch:        "main",
				repo:          "test-repo",
			}

			b := PipelineBuilder{
				GetWorkflowMetadata: m.GetWorkflowMetadata,
				RepoTrusted:         &metadata.TrustedConfiguration{},
				Yamls: []*YamlFile{
					{Name: "pipeline", Data: []byte(tc.yamlData)},
				},
			}

			items, err := b.Build()

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, items, 1, "expected exactly 1 item")

			stages := items[0].Config.Stages
			require.Len(t, stages, len(tc.wantStageSteps),
				"stage count mismatch: got %d, want %d", len(stages), len(tc.wantStageSteps))

			for stageIdx, wantStepNames := range tc.wantStageSteps {
				gotStepNames := make([]string, 0, len(stages[stageIdx].Steps))
				for _, step := range stages[stageIdx].Steps {
					gotStepNames = append(gotStepNames, step.Name)
				}
				assert.Equal(t, wantStepNames, gotStepNames,
					"stage %d step names mismatch", stageIdx)
			}
		})
	}
}

func TestFilterMissingDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		items         []*Item
		wantNames     []string
		wantDepNames  map[string][]string
	}{
		{
			name: "all_deps_present",
			items: []*Item{
				{Workflow: &Workflow{Name: "build"}, DependsOn: constraint.DependsOn{}},
				{Workflow: &Workflow{Name: "test"}, DependsOn: constraint.DependsOn{{Name: "build"}}},
			},
			wantNames:    []string{"build", "test"},
			wantDepNames: map[string][]string{"test": {"build"}},
		},
		{
			name: "missing_required_removes_consumer",
			items: []*Item{
				{Workflow: &Workflow{Name: "deploy"}, DependsOn: constraint.DependsOn{{Name: "missing"}}},
			},
			wantNames: []string{},
		},
		{
			name: "missing_optional_dropped",
			items: []*Item{
				{Workflow: &Workflow{Name: "build"}, DependsOn: constraint.DependsOn{}},
				{Workflow: &Workflow{Name: "deploy"}, DependsOn: constraint.DependsOn{
					{Name: "build"},
					{Name: "lint", Optional: true},
				}},
			},
			wantNames:    []string{"build", "deploy"},
			wantDepNames: map[string][]string{"deploy": {"build"}},
		},
		{
			name: "transitive_removal_cascade",
			items: []*Item{
				{Workflow: &Workflow{Name: "broken"}, DependsOn: constraint.DependsOn{{Name: "nonexistent"}}},
				{Workflow: &Workflow{Name: "downstream"}, DependsOn: constraint.DependsOn{{Name: "broken"}}},
				{Workflow: &Workflow{Name: "standalone"}, DependsOn: constraint.DependsOn{}},
			},
			wantNames: []string{"standalone"},
		},
		{
			name: "transitive_optional_survives",
			items: []*Item{
				{Workflow: &Workflow{Name: "broken"}, DependsOn: constraint.DependsOn{{Name: "nonexistent"}}},
				{Workflow: &Workflow{Name: "deploy"}, DependsOn: constraint.DependsOn{
					{Name: "broken", Optional: true},
				}},
			},
			wantNames:    []string{"deploy"},
			wantDepNames: map[string][]string{"deploy": {}},
		},
		{
			name: "optional_flag_cleared_after_resolution",
			items: []*Item{
				{Workflow: &Workflow{Name: "build"}, DependsOn: constraint.DependsOn{}},
				{Workflow: &Workflow{Name: "deploy"}, DependsOn: constraint.DependsOn{
					{Name: "build", Optional: true},
				}},
			},
			wantNames: []string{"build", "deploy"},
			wantDepNames: map[string][]string{"deploy": {"build"}},
		},
		{
			name: "empty_items",
			items: []*Item{},
			wantNames: []string{},
		},
		{
			name: "no_deps_at_all",
			items: []*Item{
				{Workflow: &Workflow{Name: "a"}, DependsOn: constraint.DependsOn{}},
				{Workflow: &Workflow{Name: "b"}, DependsOn: constraint.DependsOn{}},
			},
			wantNames: []string{"a", "b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := filterMissingDependencies(tc.items)

			gotNames := make([]string, 0, len(got))
			for _, item := range got {
				gotNames = append(gotNames, item.Workflow.Name)
			}
			assert.Equal(t, tc.wantNames, gotNames, "item names mismatch")

			if tc.wantDepNames != nil {
				for _, item := range got {
					if expected, ok := tc.wantDepNames[item.Workflow.Name]; ok {
						gotDepNames := make([]string, 0, len(item.DependsOn))
						for _, dep := range item.DependsOn {
							gotDepNames = append(gotDepNames, dep.Name)
						}
						assert.Equal(t, expected, gotDepNames,
							"dependency names for %q mismatch", item.Workflow.Name)
					}
				}
			}
		})
	}
}
