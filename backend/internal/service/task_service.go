package service

// 任务中心对外暴露的业务接口（对 TaskStore 的一层封装）。
//
// 设计目的：
// - Controller 不直接操作 store/内部结构，保持"controller 薄、service 厚"的分层
// - 统一错误类型已迁移至 errors.go（ErrTaskNotFound / ErrTaskCannotCancel）

type TaskService struct {
	// store 为任务的存储实现（当前为内存）。
	store *TaskStore
}

// NewTaskService 创建任务服务。
func NewTaskService(store *TaskStore) *TaskService {
	return &TaskService{store: store}
}

// List 返回分页任务列表。
// 过滤/排序逻辑由 listAndFilterTasks 完成。
func (ts *TaskService) List(req ListTasksRequest) PageResult[*Task] {
	all := ts.store.List()
	filtered := listAndFilterTasks(all, req)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	slice := []*Task{}
	if start < len(filtered) {
		slice = filtered[start:end]
	}
	return PageResult[*Task]{List: slice, Total: len(filtered), Page: page, PageSize: pageSize}
}

// Get 获取任务详情。
func (ts *TaskService) Get(id int64) (*Task, bool) {
	return ts.store.Get(id)
}

// Logs 获取任务日志（分页）。
func (ts *TaskService) Logs(id int64, offset, limit int) ([]string, bool) {
	t, ok := ts.store.Get(id)
	if !ok {
		return nil, false
	}
	return t.Logs(offset, limit), true
}

// Cancel 取消任务。
//
// 注意：
// - 此处只修改任务状态
// - 实际执行器需要配合检查状态并主动停止，才能做到“真正中断”
func (ts *TaskService) Cancel(id int64) error {
	t, ok := ts.store.Get(id)
	if !ok {
		return ErrTaskNotFound
	}
	if !t.CanCancel() {
		return ErrTaskCannotCancel
	}
	if err := t.Cancel(); err != nil {
		return err
	}
	ts.store.CancelExecution(id)
	return nil
}
