package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/service/ansible"
)

func inventoryQuoteSingle(s string) string {
	s = strings.ReplaceAll(s, `'`, `'"'"'`)
	return "'" + s + "'"
}

type AnsibleService struct {
	serverSvc *ServerService
	store     *TaskStore
	runner    *ansible.Runner
}

type RunPlaybookTaskRequest struct {
	ServerIDs []uint64
	Playbook  string
	Vars      map[string]any
	CreatedBy uint64
	Title     string
	Meta      map[string]any
}

func NewAnsibleService(serverSvc *ServerService, store *TaskStore) *AnsibleService {
	return &AnsibleService{
		serverSvc: serverSvc,
		store:     store,
		runner:    ansible.NewRunner(),
	}
}

func (s *AnsibleService) RunPlaybook(ctx context.Context, serverIDs []uint64, playbook string, createdBy uint64) (int64, error) {
	return s.RunPlaybookTask(ctx, RunPlaybookTaskRequest{
		ServerIDs: serverIDs,
		Playbook:  playbook,
		CreatedBy: createdBy,
	})
}

func (s *AnsibleService) RunPlaybookWithVars(ctx context.Context, serverIDs []uint64, playbook string, vars map[string]any, createdBy uint64) (int64, error) {
	return s.RunPlaybookTask(ctx, RunPlaybookTaskRequest{
		ServerIDs: serverIDs,
		Playbook:  playbook,
		Vars:      vars,
		CreatedBy: createdBy,
	})
}

func (s *AnsibleService) RunPlaybookTask(ctx context.Context, req RunPlaybookTaskRequest) (int64, error) {
	if len(req.ServerIDs) == 0 {
		return 0, fmt.Errorf("no target servers")
	}

	// 1. Create Task
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = fmt.Sprintf("Run Playbook on %d servers", len(req.ServerIDs))
	}
	msg := "Preparing..."
	percent := 0
	meta := map[string]any{}
	for key, value := range req.Meta {
		meta[key] = value
	}
	t := &Task{
		Type:      "ansible_playbook",
		Status:    TaskPending,
		Title:     &title,
		CreatedBy: int64(req.CreatedBy),
		Message:   &msg,
		Percent:   &percent,
		Meta:      meta,
		Steps: []TaskStep{
			{Key: "prepare", Title: "Prepare Inventory", Status: StepPending},
			{Key: "run", Title: "Run Playbook", Status: StepPending},
		},
	}
	if err := s.store.Put(t); err != nil {
		return 0, err
	}

	// 2. Async Run
	execCtx, cancel := context.WithCancel(context.Background())
	s.store.RegisterCancel(t.ID, cancel)
	go s.runAsync(execCtx, t, req.ServerIDs, req.Playbook, req.Vars)

	return t.ID, nil
}

func (s *AnsibleService) runAsync(ctx context.Context, t *Task, serverIDs []uint64, playbook string, vars map[string]any) {
	defer s.store.UnregisterCancel(t.ID)

	t.Status = TaskRunning
	t.Update()

	// Step 1: Prepare
	t.Steps[0].Status = StepRunning
	st := time.Now().UTC()
	t.Steps[0].StartedAt = &st
	t.Update()

	var inventoryBuilder strings.Builder
	inventoryBuilder.WriteString("[targets]\n")
	extraFiles := make(map[string][]byte)

	for _, id := range serverIDs {
		srv, err := s.serverSvc.GetServer(ctx, id)
		if err != nil {
			s.failStep(t, 0, fmt.Sprintf("Get server %d failed: %v", id, err))
			return
		}
		pass, key, err := s.serverSvc.GetServerCredentials(ctx, id)
		if err != nil {
			s.failStep(t, 0, fmt.Sprintf("Get creds %d failed: %v", id, err))
			return
		}

		// Use Name as alias, set ansible_host to IP
		line := fmt.Sprintf("%s ansible_host=%s ansible_port=%d ansible_user=%s", srv.Name, srv.IP, srv.Port, srv.Username)

		if srv.AuthType == "key" && key != "" {
			keyFileName := fmt.Sprintf("key_%d", id)
			extraFiles[keyFileName] = []byte(key)
			line += fmt.Sprintf(" ansible_ssh_private_key_file=./%s", keyFileName)
		} else if pass != "" {
			line += fmt.Sprintf(" ansible_ssh_pass=%s", inventoryQuoteSingle(pass))
		}

		// Disable host key checking to avoid interactive prompt
		line += " ansible_ssh_common_args='-o StrictHostKeyChecking=no'"
		inventoryBuilder.WriteString(line + "\n")
	}

	ft := time.Now().UTC()
	t.Steps[0].Status = StepSuccess
	t.Steps[0].FinishedAt = &ft
	t.Update()

	// Step 2: Run
	t.Steps[1].Status = StepRunning
	st2 := time.Now().UTC()
	t.Steps[1].StartedAt = &st2
	t.Update()

	err := s.runner.RunPlaybook(ctx, inventoryBuilder.String(), playbook, vars, extraFiles, func(line string) {
		t.AppendLog(line)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			ft2 := time.Now().UTC()
			t.Steps[1].Status = StepFailed
			t.Steps[1].FinishedAt = &ft2
			msg := "已取消"
			t.Steps[1].Message = &msg
			t.Status = TaskCanceled
			t.Message = &msg
			t.Update()
			return
		}
		s.failStep(t, 1, err.Error())
		return
	}

	ft2 := time.Now().UTC()
	t.Steps[1].Status = StepSuccess
	t.Steps[1].FinishedAt = &ft2

	t.Status = TaskSuccess
	msg := "Success"
	t.Message = &msg
	p := 100
	t.Percent = &p
	t.Update()
}

func (s *AnsibleService) failStep(t *Task, stepIdx int, err string) {
	ft := time.Now().UTC()
	t.Steps[stepIdx].Status = StepFailed
	t.Steps[stepIdx].FinishedAt = &ft
	msg := err
	t.Steps[stepIdx].Message = &msg

	t.Status = TaskFailed
	t.Message = &msg
	t.Update()
	t.AppendLog("[Error] " + err)
}
