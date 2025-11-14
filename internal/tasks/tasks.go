package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/you/lazyadmin/internal/clients"
	"github.com/you/lazyadmin/internal/config"
	"github.com/you/lazyadmin/internal/logging"
)

type StepResult struct {
	Step   config.TaskStep
	OK     bool
	Output string
	Err    error
}

type TaskResult struct {
	Task      config.Task
	Success   bool
	StepOrder []string
	Steps     map[string]StepResult
}

type Runner struct {
	cfg         *config.Config
	logger      *logging.AuditLogger
	httpClients map[string]*clients.HTTPClient
	pgClients   map[string]*clients.PostgresClient
}

func NewRunner(
	cfg *config.Config,
	logger *logging.AuditLogger,
	httpClients map[string]*clients.HTTPClient,
	pgClients map[string]*clients.PostgresClient,
) *Runner {
	return &Runner{
		cfg:         cfg,
		logger:      logger,
		httpClients: httpClients,
		pgClients:   pgClients,
	}
}

func (r *Runner) Run(ctx context.Context, principalUserID, sshUser string, task config.Task) TaskResult {
	res := TaskResult{
		Task:      task,
		Success:   true,
		Steps:     make(map[string]StepResult),
		StepOrder: make([]string, 0, len(task.Steps)),
	}

	taskPolicy := task.OnError
	if taskPolicy == "" {
		taskPolicy = config.OnErrorFailFast
	}

	for _, step := range task.Steps {
		res.StepOrder = append(res.StepOrder, step.ID)

		stepPolicy := step.OnError
		if stepPolicy == "" || stepPolicy == config.StepOnErrorInherit {
			stepPolicy = stepOnErrorFromTask(taskPolicy)
		}

		sr := r.runStep(ctx, step)
		res.Steps[step.ID] = sr

		_ = r.logStep(principalUserID, sshUser, task.ID, sr)

		if sr.Err != nil {
			if stepPolicy == config.StepOnErrorFail {
				res.Success = false
				break
			}
			if stepPolicy == config.StepOnErrorWarn {
				res.Success = false
				continue
			}
			if stepPolicy == config.StepOnErrorContinue {
				continue
			}
		}
	}

	_ = r.logTask(principalUserID, sshUser, task.ID, res.Success)

	return res
}

func stepOnErrorFromTask(taskPolicy config.OnErrorPolicy) config.StepOnError {
	switch taskPolicy {
	case config.OnErrorFailFast:
		return config.StepOnErrorFail
	case config.OnErrorBestEffort:
		return config.StepOnErrorContinue
	default:
		return config.StepOnErrorFail
	}
}

func (r *Runner) runStep(ctx context.Context, step config.TaskStep) StepResult {
	switch step.Type {
	case "http":
		client, ok := r.httpClients[step.Resource]
		if !ok {
			return StepResult{Step: step, OK: false, Err: fmt.Errorf("no http resource %q", step.Resource)}
		}
		out, err := client.Request(ctx, step.Method, step.Path)
		return StepResult{Step: step, OK: err == nil, Output: out, Err: err}

	case "postgres":
		client, ok := r.pgClients[step.Resource]
		if !ok {
			return StepResult{Step: step, OK: false, Err: fmt.Errorf("no postgres resource %q", step.Resource)}
		}
		out, err := client.RunScalarQuery(ctx, step.Query)
		return StepResult{Step: step, OK: err == nil, Output: out, Err: err}

	case "sleep":
		d := time.Duration(step.Seconds) * time.Second
		select {
		case <-time.After(d):
			return StepResult{Step: step, OK: true, Output: fmt.Sprintf("slept %s", d)}
		case <-ctx.Done():
			return StepResult{Step: step, OK: false, Err: ctx.Err()}
		}

	default:
		return StepResult{Step: step, OK: false, Err: fmt.Errorf("unsupported step type %q", step.Type)}
	}
}

func (r *Runner) logStep(userID, sshUser, taskID string, sr StepResult) error {
	if r.logger == nil {
		return nil
	}

	entry := logging.AuditEntry{
		Time:        time.Now(),
		UserID:      userID,
		SSHUser:     sshUser,
		OperationID: fmt.Sprintf("task:%s step:%s", taskID, sr.Step.ID),
		Success:     sr.Err == nil,
	}
	if sr.Err != nil {
		entry.Error = sr.Err.Error()
	}

	return r.logger.Log(context.Background(), entry)
}

func (r *Runner) logTask(userID, sshUser, taskID string, success bool) error {
	if r.logger == nil {
		return nil
	}

	entry := logging.AuditEntry{
		Time:        time.Now(),
		UserID:      userID,
		SSHUser:     sshUser,
		OperationID: fmt.Sprintf("task:%s", taskID),
		Success:     success,
	}

	return r.logger.Log(context.Background(), entry)
}

// RenderSummary executes the task's summary template with the task results.
func RenderSummary(task config.Task, tr TaskResult) (string, error) {
	if task.SummaryTemplate == "" {
		return "", nil
	}

	type stepView struct {
		OK     bool
		Output string
		Error  string
	}

	ctx := struct {
		Task    config.Task
		Success bool
		Steps   map[string]stepView
	}{
		Task:    task,
		Success: tr.Success,
		Steps:   make(map[string]stepView),
	}

	for id, sr := range tr.Steps {
		errText := ""
		if sr.Err != nil {
			errText = sr.Err.Error()
		}
		ctx.Steps[id] = stepView{
			OK:     sr.OK,
			Output: sr.Output,
			Error:  errText,
		}
	}

	return executeTemplate(task.SummaryTemplate, ctx)
}
