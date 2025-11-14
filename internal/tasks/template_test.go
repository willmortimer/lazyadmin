package tasks

import (
	"testing"

	"github.com/you/lazyadmin/internal/config"
)

func TestExecuteTemplate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    string
		data    any
		want    string
		wantErr bool
	}{
		{
			name:    "simple template",
			tmpl:    "Hello {{.Name}}",
			data:    map[string]string{"Name": "World"},
			want:    "Hello World",
			wantErr: false,
		},
		{
			name: "template with struct",
			tmpl: "Task {{.Task.ID}} succeeded: {{.Success}}",
			data: struct {
				Task    config.Task
				Success bool
			}{
				Task:    config.Task{ID: "deploy"},
				Success: true,
			},
			want:    "Task deploy succeeded: true",
			wantErr: false,
		},
		{
			name: "template with nested fields",
			tmpl: "{{.Task.ID}} - {{.Task.Label}}",
			data: struct {
				Task config.Task
			}{
				Task: config.Task{ID: "test", Label: "Test Task"},
			},
			want:    "test - Test Task",
			wantErr: false,
		},
		{
			name: "template with range",
			tmpl: "{{range $k, $v := .Items}}{{$k}}:{{$v}} {{end}}",
			data: map[string]any{
				"Items": map[string]string{
					"a": "1",
					"b": "2",
				},
			},
			want:    "a:1 b:2 ",
			wantErr: false,
		},
		{
			name:    "invalid template syntax",
			tmpl:    "{{.Invalid",
			data:    map[string]string{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing field",
			tmpl:    "{{.Missing}}",
			data:    map[string]string{},
			want:    "<no value>",
			wantErr: false,
		},
		{
			name:    "empty template",
			tmpl:    "",
			data:    map[string]string{},
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeTemplate(tt.tmpl, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("executeTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderSummary(t *testing.T) {
	tests := []struct {
		name    string
		task    config.Task
		result  TaskResult
		want    string
		wantErr bool
	}{
		{
			name: "empty template",
			task: config.Task{
				ID:              "test",
				SummaryTemplate: "",
			},
			result: TaskResult{
				Success: true,
				Steps:   map[string]StepResult{},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "simple summary",
			task: config.Task{
				ID:              "deploy",
				SummaryTemplate: "Task {{.Task.ID}} completed: {{.Success}}",
			},
			result: TaskResult{
				Task:    config.Task{ID: "deploy"},
				Success: true,
				Steps:   map[string]StepResult{},
			},
			want:    "Task deploy completed: true",
			wantErr: false,
		},
		{
			name: "summary with steps",
			task: config.Task{
				ID:              "test",
				SummaryTemplate: "{{range $id, $step := .Steps}}Step {{$id}}: {{if $step.OK}}OK{{else}}FAIL{{end}}\n{{end}}",
			},
			result: TaskResult{
				Task:    config.Task{ID: "test"},
				Success: true,
				Steps: map[string]StepResult{
					"step1": {OK: true, Output: "success"},
					"step2": {OK: false, Err: &testError{msg: "failed"}},
				},
			},
			want:    "Step step1: OK\nStep step2: FAIL\n",
			wantErr: false,
		},
		{
			name: "summary with step output",
			task: config.Task{
				ID:              "deploy",
				SummaryTemplate: "{{range $id, $step := .Steps}}{{if $step.OK}}{{$.Task.ID}}.{{$id}}: {{$step.Output}}\n{{end}}{{end}}",
			},
			result: TaskResult{
				Task:    config.Task{ID: "deploy"},
				Success: true,
				Steps: map[string]StepResult{
					"step1": {OK: true, Output: "deployed"},
				},
			},
			want:    "deploy.step1: deployed\n",
			wantErr: false,
		},
		{
			name: "invalid template",
			task: config.Task{
				ID:              "test",
				SummaryTemplate: "{{.Invalid",
			},
			result: TaskResult{
				Success: true,
				Steps:   map[string]StepResult{},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderSummary(tt.task, tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RenderSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
