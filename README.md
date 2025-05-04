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



