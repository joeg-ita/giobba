# giobba: a go job distributed executer

giobba is a job executer relying on a broker for queueing and pub/sub tasks and a database for tasks persistance,
In the default implementation the broker is **redis** and the database is **mongodb**.

## Features

- Distributed task execution with multiple workers
- task ETA scheduling with priorities (0-10)
- Support for task dependencies (parent-child relationships)
- Automatic and manual task execution modes
- Task persistence in MongoDB
- Redis-based task queue and pub/sub system
- Configurable number of workers and queues
- Task locking mechanism to prevent duplicate execution
- Graceful shutdown support

## Installation

```bash
go get github.com/joeg-ita/giobba
```

## Configuration

Create a configuration file named `config.yml` in one of the following locations:
- `/etc/giobba.d/`
- `~/.giobba/`
- Current working directory

Example configuration:

```yaml
name: giobba
version: 0.1.0
queues: ["default", "background"]
workersNumber: 5
lockDuration: 10

database:
  url: localhost
  port: 27017
  username: user
  password: pass
  db: giobba
  collection: tasks

broker:
  url: localhost
  port: 6379
  username: user
  password: pass
  db: 0
```

You can also use environment variables to override configuration values:
- `GIOBBA_ENV`: Environment name (e.g., "dev", "prod")
- `GIOBBA_DATABASE_URL`: MongoDB connection URL
- `GIOBBA_BROKER_URL`: Redis connection URL
- `APP_VERSION`: Application version

## Usage

### Basic Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/joeg-ita/giobba"
    "github.com/joeg-ita/giobba/src/domain"
)

func main() {
    // Start giobba
    go giobba.Giobba()
    
    // Create a task
    payload := map[string]interface{}{
        "user": "john",
        "job":  "process_data",
    }
    
    // Create a task with:
    // - name: "process"
    // - payload: custom data
    // - queue: "default"
    // - execution time: now
    // - priority: 5
    // - start mode: AUTO
    // - parent ID: "" (no parent)
    task, _ := domain.NewTask("process", payload, "default", time.Now(), 5, domain.AUTO, "")
    
    // Add task to the scheduler
    scheduler.AddTask(task)
}
```

### Task Dependencies Example

```go
// Create a parent task
parentTask, _ := domain.NewTask("process", payload, "default", time.Now(), 5, domain.AUTO, "")
parentId, _ := scheduler.AddTask(parentTask)

// Create a child task that depends on the parent
childTask, _ := domain.NewTask("process", childPayload, "default", time.Now(), 5, domain.AUTO, parentId)
scheduler.AddTask(childTask)
```

### Manual Task Execution

```go
// Create a task with manual start mode
task, _ := domain.NewTask("process", payload, "default", time.Now(), 5, domain.MANUAL, "")
taskId, _ := scheduler.AddTask(task)

// Manually start the task when ready
scheduler.AutoTask(taskId, "default")
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

# Giobba Task Scheduler

Giobba is a flexible task scheduling system that allows you to create and manage tasks with different priorities, scheduling options, and execution modes.

## Creating Custom Task Handlers

To create a custom task handler, you can embed the `BaseHandler` struct and implement the `Run` method. Here's an example:

```go
package myhandlers

import (
    "context"
    "github.com/joeg-ita/giobba/src/domain"
    "github.com/joeg-ita/giobba/src/handlers"
)

// MyCustomHandler implements a custom task handler
type MyCustomHandler struct {
    handlers.BaseHandler
    // Add any custom fields here
}

// Run implements the task execution logic
func (h *MyCustomHandler) Run(ctx context.Context, task domain.Task) error {
    // Access task payload
    payload := task.Payload
    
    // Your custom logic here
    // ...
    
    // Check for cancellation
    select {
    case <-ctx.Done():
        return fmt.Errorf("task cancelled")
    default:
        // Continue with work
    }
    
    return nil
}
```

## Creating Tasks

The package provides several helper functions to create tasks with different configurations:

### Basic Task
```go
task, err := handlers.NewDefaultTask(
    "my-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
)
```

### Child Task
```go
task, err := handlers.NewChildTask(
    "child-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
    parentTaskId,
)
```

### Manual Task
```go
task, err := handlers.NewManualTask(
    "manual-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
)
```

### High Priority Task
```go
task, err := handlers.NewHighPriorityTask(
    "high-priority-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
)
```

### Scheduled Task
```go
task, err := handlers.NewScheduledTask(
    "scheduled-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
    time.Now().Add(1 * time.Hour),
)
```

## Task Properties

Tasks have several configurable properties:

- `Name`: Unique identifier for the task type
- `Payload`: Map of data to be processed by the handler
- `Queue`: Queue where the task will be processed
- `State`: Current state of the task (PENDING, RUNNING, COMPLETED, FAILED, REVOKED, KILLED)
- `ETA`: Expected time of arrival/execution
- `Priority`: Task priority (0-10, higher is more important)
- `StartMode`: AUTO or MANUAL
- `ParentID`: ID of the parent task (for task chains)
- `Callback`: URL to call when task completes successfully
- `CallbackErr`: URL to call when task fails

## Best Practices

1. Always check for context cancellation in long-running tasks
2. Use appropriate task priorities based on importance
3. Implement proper error handling and logging
4. Use callbacks for task completion notifications
5. Consider using child tasks for complex workflows
6. Set appropriate ETA values for scheduled tasks

## Example Workflow

```go
// Create a parent task
parentTask, err := handlers.NewDefaultTask(
    "parent-task",
    map[string]interface{}{
        "data": "parent-data",
    },
    "my-queue",
)

// Create child tasks
childTask1, err := handlers.NewChildTask(
    "child-task-1",
    map[string]interface{}{
        "data": "child-1-data",
    },
    "my-queue",
    parentTask.ID,
)

childTask2, err := handlers.NewChildTask(
    "child-task-2",
    map[string]interface{}{
        "data": "child-2-data",
    },
    "my-queue",
    parentTask.ID,
)
```

This creates a workflow where child tasks will only execute after the parent task completes successfully.



