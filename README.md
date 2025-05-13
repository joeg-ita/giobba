# Giobba: A Go Distributed Job Executor

**Giobba** is a flexible distributed task scheduling system that allows you to create and manage tasks with different priorities, scheduling options, and execution modes. It provides a robust solution for handling distributed workloads with features like task dependencies, priority-based execution, and automatic/manual task triggering.

## Architecture

Giobba uses a distributed architecture with the following components:

- **Broker**: Handles task queuing and pub/sub communication (default: Redis)
- **Database**: Stores task and job persistence (default: MongoDB)
- **Scheduler**: Manages task execution and worker coordination
- **Workers**: Execute tasks in parallel across multiple instances

## Features

- Distributed task execution with multiple workers
- Task ETA scheduling with priorities (0-10)
- Support for task dependencies (parent-child relationships)
- Automatic and manual task execution modes
- Task persistence in MongoDB
- Redis-based task queue and pub/sub system
- Configurable number of workers and queues
- Task locking mechanism to prevent duplicate execution
- Graceful shutdown support
- Cron-style scheduling support
- Task callbacks for success/failure notifications
- Task retry mechanism with configurable attempts
- Task expiration support
- Worker health monitoring and heartbeats

## Installation

```bash
go get github.com/joeg-ita/giobba
```

## Configuration

Create a configuration file named `giobba.yml` in one of the following locations:
- `/etc/giobba.d/`
- `~/.giobba/`
- Current working directory

Example configuration:

```yaml
name: giobba
version: 0.1.0
queues: ["default", "background"]
workersNumber: 5
lockDuration: 60
jobsTimeoutRefresh: 300

database:
  url: "mongodb://localhost:27017"
  db: giobba
  tasksCollection: tasks
  jobsCollection: jobs

broker:
  url: "redis://localhost:6379/0"
```

### Environment Variables

You can override configuration values using environment variables:
- `GIOBBA_ENV`: Environment name (e.g., "dev", "prod" to select giobba-[dev|prod].yml)
- `GIOBBA_DATABASE_URL`: MongoDB connection URL
- `GIOBBA_BROKER_URL`: Redis connection URL
- `GIOBBA_DATABASE_PORT`: MongoDB port
- `GIOBBA_DATABASE_ADMIN_USERNAME`: MongoDB admin username
- `GIOBBA_DATABASE_ADMIN_PASSWORD`: MongoDB admin password
- `GIOBBA_BROKER_PORT`: Redis port
- `GIOBBA_BROKER_ADMIN_USERNAME`: Redis admin username
- `GIOBBA_BROKER_ADMIN_PASSWORD`: Redis admin password
- `GIOBBA_BROKER_DB`: Redis database number

## Task Properties

Tasks have several configurable properties:

- `ID`: Unique identifier for the task
- `Name`: Unique identifier for the task type
- `Payload`: Map of data to be processed by the handler
- `Queue`: Queue where the task will be processed
- `State`: Current state of the task (PENDING, RUNNING, COMPLETED, FAILED, REVOKED, KILLED)
- `ETA`: Expected time of arrival/execution
- `Priority`: Task priority (0-10, higher is more important)
- `StartMode`: AUTO or MANUAL
- `ParentID`: ID of the parent task (for task chains)
- `Schedule`: Cron expression for recurring tasks (ie. 0 0 * * *)
- `IsScheduleActive`: Whether the schedule is active
- `JobID`: Associated job ID for scheduled tasks
- `Error`: Last error message if task failed
- `CreatedAt`: Task creation timestamp
- `UpdatedAt`: Last update timestamp
- `StartedAt`: Task start timestamp
- `CompletedAt`: Task completion timestamp
- `ExpiresAt`: Task expiration timestamp
- `Result`: Task execution result
- `Retries`: Number of retry attempts
- `MaxRetries`: Maximum number of retry attempts
- `Tags`: Custom tags for task categorization
- `SchedulerID`: ID of the scheduler that processed the task
- `WorkerID`: ID of the worker that executed the task
- `ChildrenID`: IDs of child tasks
- `Callback`: URL to call when task completes successfully
- `CallbackErr`: URL to call when task fails

## Usage

### Creating Custom Task Handlers

To create a custom task handler, embed the `BaseHandler` struct and implement the `Run` method:

```go
type MyHandler struct {
    BaseHandler
}

func (m *MyHandler) Run(ctx context.Context, task domain.Task) error {
    log.Printf("MyHandler processing task: %s", task.Name)

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

Register your handler before starting Giobba:

```go
handler.Handlers["myHandler"] = &MyHandler{}
```

### Basic Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/joeg-ita/giobba"
    "github.com/joeg-ita/giobba/src/domain"
    "github.com/joeg-ita/giobba/src/handler"
)

func main() {
    // Add custom handler
    handler.Handlers["myHandler"] = &MyHandler{}

    // Start giobba
    go giobba.Giobba()
    
    // Create a task
    payload := map[string]interface{}{
        "user": "john",
        "job":  "process_data",
    }
    
    // Create a task with:
    // - name: "myHandler"
    // - payload: custom data
    // - queue: "default"
    // - execution time: now
    // - priority: 5
    // - start mode: AUTO
    // - parent ID: "" (no parent)
    task, _ := domain.NewTask("myHandler", payload, "default", time.Now(), 5, domain.AUTO, "")
    
    // Add task to the scheduler
    scheduler.Tasks.AddTask(task)
}
```

### Task Creation Helpers

The package provides several helper functions to create tasks with different configurations:

#### Basic Task
```go
task, err := handlers.NewDefaultTask(
    "my-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
)
```

#### Child Task
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

#### Manual Task
```go
task, err := handlers.NewManualTask(
    "manual-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
)
```

#### High Priority Task
```go
task, err := handlers.NewHighPriorityTask(
    "high-priority-task",
    map[string]interface{}{
        "key": "value",
    },
    "my-queue",
)
```

#### Scheduled Task
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

## Best Practices

1. Always check for context cancellation in long-running tasks
2. Use appropriate task priorities based on importance
3. Implement proper error handling and logging
4. Use callbacks for task completion notifications
5. Consider using child tasks and manual triggering for complex workflows
6. Set appropriate ETA values and priorities for scheduled tasks
7. Use task tags for better organization and filtering
8. Implement proper error handling in task callbacks
9. Monitor worker health through heartbeats
10. Use task expiration for cleanup of old tasks
11. Configure appropriate retry policies for failed tasks
12. Use task locking to prevent duplicate execution

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
