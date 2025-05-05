# giobba: a go job distributed executer

**Giobba** is a flexible distributed task scheduling system that allows you to create and manage tasks with different priorities, scheduling options, and execution modes.


**Giobba** relies on a broker for queueing-pub/sub tasks and a database for tasks persistance.

*In the default implementation the broker is **redis** and the database is **mongodb**.*

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

## Best Practices

1. Always check for context cancellation in long-running tasks
2. Use appropriate task priorities based on importance
3. Implement proper error handling and logging
4. Use callbacks for task completion notifications
5. Consider using child tasks and manual triggering for complex workflows
6. Set appropriate ETA values and priorities for scheduled tasks

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

database:
  url: "mongodb://localhost:27017"
  db: giobba
  collection: tasks

broker:
  url: "redis://localhost:6379/0"
  db: 0
```

You can also use environment variables to override configuration values:
- `GIOBBA_ENV`: Environment name (e.g., "dev", "prod" to select giobba-[dev|prod].yml config file)
- `GIOBBA_DATABASE_URL`: MongoDB connection URL
- `GIOBBA_BROKER_URL`: Redis connection URL

## Usage

### Creating Custom Task Handlers

To create a custom task handler, you can embed the `BaseHandler` struct and implement the `Run` method. 

The custom handler will address your business problem. Once created, the handler must be registered on the scheduler.

Here's an example of custom handler, i.e. `MyHandler`

```go

type MyHandler struct {
	BaseHandler
}

func (t *MyHandler) Run(ctx context.Context, task domain.Task) error {
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

Add `MyHandler` to Handlers map before start giobba. The `key` of the added handler must be the value of the name field of the Task to process 
```go
handler.Handlers["myHandler", &MyHandler{}]
```

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
    // Add custom handler
    giobba.handler.Handlers["myHandler", &MyHandler{}]

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

## Other usecases

### Parent Child Tasks

```go
// Create a parent task
parentTask, _ := domain.NewTask("myHandler", payload, "default", time.Now(), 5, domain.AUTO, "")
parentId, _ := scheduler.AddTask(parentTask)

// Create a child task that depends on the parent
childTask, _ := domain.NewTask("myHandler", childPayload, "default", time.Now(), 5, domain.AUTO, parentId)
scheduler.AddTask(childTask)
```

### Manual Task Execution

```go
// Create a task with manual start mode
task, _ := domain.NewTask("myHandler", payload, "default", time.Now(), 5, domain.MANUAL, "")
taskId, _ := scheduler.AddTask(task)

// Manually start the task when ready
scheduler.AutoTask(taskId, "default")
```

### Tasks creation helpers

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


## License
        
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
