# Agent Poller + Executor 解耦设计

## 概述

本设计文档描述了如何将 Agent 中的任务轮询（Polling）和任务执行（Execution）从当前的强耦合架构解耦为独立的组件。

## 重构前架构

### 当前依赖关系

```
┌─────────────────────────────────────────────────────────┐
│                         Runner                          │
│  ┌─────────────────────────────────────────────────┐  │
│  │  直接依赖 rpc.Peer (gRPC 客户端)                 │  │
│  │  - Next()      轮询下一个任务                    │  │
│  │  - Init()      初始化任务状态                    │  │
│  │  - Extend()    续约任务                          │  │
│  │  - Wait()      等待取消信号                      │  │
│  │  - Done()      报告任务完成                      │  │
│  └─────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │   rpc.Peer      │
              │  (gRPC 客户端)  │
              └─────────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │   Server        │
              └─────────────────┘
```

### 问题

1. **强耦合**：Runner 直接依赖 gRPC 客户端，难以测试和替换
2. **职责混杂**：轮询逻辑和执行逻辑混合在一起
3. **难以扩展**：添加新的任务获取方式或执行方式需要修改核心代码

## 重构后架构

### 新的依赖关系

```
┌─────────────────────────────────────────────────────────────┐
│                        Runner                               │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  依赖抽象接口                                        │  │
│  │  - Poller: 负责轮询任务                              │  │
│  │  - Executor: 负责执行任务                            │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
┌──────────────────────┐    ┌─────────────────────────┐
│   Poller 接口        │    │   Executor 接口        │
│  ┌────────────────┐ │    │  ┌──────────────────┐  │
│  │ - Poll()      │ │    │  │ - Execute()      │  │
│  │ - WaitCancel()│ │    │  │ - Cancel()       │  │
│  │ - Extend()    │ │    │  └──────────────────┘  │
│  │ - ReportDone()│ │    └─────────────────────────┘
│  └────────────────┘ │              │
└──────────────────────┘              │
           │                          │
           ▼                          ▼
┌──────────────────────┐    ┌─────────────────────────┐
│ GRPCPoller 实现      │    │  DefaultExecutor 实现   │
│ (封装 gRPC 客户端)   │    │ (封装执行逻辑)          │
└──────────────────────┘    └─────────────────────────┘
```

## 接口定义

### Poller 接口

```go
// Poller defines the interface for fetching work from the server
type Poller interface {
	// Poll fetches the next available work item
	Poll(ctx context.Context, filter rpc.Filter) (*rpc.Work, error)

	// WaitCancel waits for a cancellation signal from the server
	WaitCancel(ctx context.Context, workID string) (canceled bool, err error)

	// Extend extends the lease for a work item
	Extend(ctx context.Context, workID string) error

	// ReportInit reports the work has started
	ReportInit(ctx context.Context, workID string, state rpc.WorkflowState) error

	// ReportDone reports the work has completed
	ReportDone(ctx context.Context, workID string, state rpc.WorkflowState) error

	// Close releases resources
	Close() error
}
```

### Executor 接口

```go
// Executor defines the interface for executing work items
type Executor interface {
	// Execute executes the work item
	Execute(ctx context.Context, work *rpc.Work, opts *ExecuteOptions) (*ExecutionResult, error)

	// Cancel cancels the ongoing execution
	Cancel(ctx context.Context) error
}

// ExecuteOptions contains options for work execution
type ExecuteOptions struct {
	Hostname  string
	Backend   backend_types.Backend
	Logger    zerolog.Logger
}

// ExecutionResult contains the result of work execution
type ExecutionResult struct {
	State rpc.WorkflowState
	Error error
}
```

### 重构后的 Runner

```go
// Runner orchestrates the polling and execution of work
type Runner struct {
	poller     Poller
	executor   Executor
	hostname   string
	state      *State
	filter     rpc.Filter
}

func NewRunner(poller Poller, executor Executor, hostname string, state *State, filter rpc.Filter) *Runner {
	return &Runner{
		poller:   poller,
		executor: executor,
		hostname: hostname,
		state:    state,
		filter:   filter,
	}
}

// Run runs the main work loop
func (r *Runner) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			work, err := r.poller.Poll(ctx, r.filter)
			if err != nil {
				return err
			}
			if work == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			go r.executeWork(ctx, work)
		}
	}
}

func (r *Runner) executeWork(ctx context.Context, work *rpc.Work) {
	// Track work state
	timeout := time.Hour
	if work.Timeout > 0 {
		timeout = time.Duration(work.Timeout) * time.Minute
	}
	r.state.Add(work.ID, timeout, extractRepoName(work.Config), extractPipelineNumber(work.Config))
	defer r.state.Done(work.ID)

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Listen for cancellation
	go func() {
		canceled, err := r.poller.WaitCancel(execCtx, work.ID)
		if err != nil || canceled {
			cancel()
		}
	}()

	// Extend lease periodically
	go func() {
		ticker := time.NewTicker(constant.TaskTimeout / 3)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				_ = r.poller.Extend(execCtx, work.ID)
			}
		}
	}()

	// Report initialization
	initState := rpc.WorkflowState{Started: time.Now().Unix()}
	if err := r.poller.ReportInit(execCtx, work.ID, initState); err != nil {
		cancel()
		return
	}

	// Execute the work
	result, err := r.executor.Execute(execCtx, work, &ExecuteOptions{
		Hostname: r.hostname,
		Backend:  r.backend, // 注意：这里需要重新设计传递
	})

	// Report completion
	if err != nil {
		result.State.Error = err.Error()
		if errors.Is(err, pipeline_errors.ErrCancel) {
			result.State.Canceled = true
			result.State.Error = pipeline_errors.ErrCancel.Error()
		}
	}

	// Use a new context for reporting completion in case execCtx is canceled
	reportCtx := ctx
	if reportCtx.Err() != nil {
		var cancelReport context.CancelFunc
		reportCtx, cancelReport = GetShutdownContext()
		defer cancelReport()
	}
	_ = r.poller.ReportDone(reportCtx, work.ID, result.State)
}
```

## 优势

1. **解耦**：轮询和执行逻辑完全分离
2. **可测试性**：可以轻松 mock Poller 和 Executor 进行单元测试
3. **可扩展性**：
   - 可以实现不同的 Poller（如本地队列、HTTP 等）
   - 可以实现不同的 Executor（如容器、本地进程等）
4. **单一职责**：每个组件只有一个明确的职责
5. **依赖倒置**：高层模块（Runner）依赖抽象接口，而不是具体实现

## 实现建议

1. 保持向后兼容性，逐步迁移
2. 先创建接口定义，再重构现有代码
3. 充分测试每个独立组件
