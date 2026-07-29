package application

import "strings"

type AutomationTaskListRequest struct {
	Page, PageSize                       int
	Type, Status, Keyword, SortBy, Order string
}

type AutomationTaskRuntime interface {
	ListAutomationTasks(AutomationTaskListRequest) (any, error)
	GetAutomationTask(int64) (any, error)
	AutomationTaskLogs(int64, int, int) ([]string, error)
	CancelAutomationTask(int64) error
}

type AutomationTaskService struct{ runtime AutomationTaskRuntime }

func NewAutomationTaskService(runtime AutomationTaskRuntime) *AutomationTaskService {
	return &AutomationTaskService{runtime: runtime}
}

func (s *AutomationTaskService) List(request AutomationTaskListRequest) (any, error) {
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	request.Page, request.PageSize = normalizePage(request.Page, request.PageSize)
	request.Type, request.Status, request.Keyword = strings.TrimSpace(request.Type), strings.TrimSpace(request.Status), strings.TrimSpace(request.Keyword)
	request.SortBy, request.Order = strings.TrimSpace(request.SortBy), strings.TrimSpace(request.Order)
	return s.runtime.ListAutomationTasks(request)
}

func (s *AutomationTaskService) Get(taskID int64) (any, error) {
	if taskID <= 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.GetAutomationTask(taskID)
}

func (s *AutomationTaskService) Logs(taskID int64, offset, limit int) ([]string, error) {
	if taskID <= 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	return s.runtime.AutomationTaskLogs(taskID, offset, limit)
}

func (s *AutomationTaskService) Cancel(taskID int64) error {
	if taskID <= 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CancelAutomationTask(taskID)
}
