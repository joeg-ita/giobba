# Giobba: A Go Distributed Job Executor

**Giobba** is a flexible distributed task scheduling system that allows you to create and manage tasks with different priorities, scheduling options, and execution modes. It provides a robust solution for handling distributed workloads with features like task dependencies, priority-based execution, and automatic/manual task triggering.

## Architecture

Giobba uses a distributed architecture with the following components:

- **Broker**: Handles task queuing and pub/sub communication (Redis)
  - Task queuing with priority support
  - Pub/sub messaging for service communication
  - Task locking mechanism
  - Scheduled task management
- **Database**: Stores task and job persistence (MongoDB)
  - Task history and state tracking
  - Job scheduling information
  - Task recovery and monitoring
- **Scheduler**: Manages task execution and worker coordination
  - Worker pool management
  - Task distribution
  - Health monitoring
  - Graceful shutdown
- **Workers**: Execute tasks in parallel across multiple instances
  - Concurrent task execution
  - Task retry mechanism
  - Context-aware execution
  - Error handling

## Features

- **Task Management**
  - Distributed task execution with multiple workers
  - Task ETA scheduling with priorities (0-10)
  - Support for task dependencies (parent-child relationships)
  - Automatic and manual task execution modes
  - Task persistence in MongoDB
  - Task locking mechanism to prevent duplicate execution
  - Task retry mechanism with configurable attempts
  - Task expiration support
  - Task callbacks for success/failure notifications (HTTP/HTTPS endpoints)

- **Scheduling**
  - Cron-style scheduling support
  - Job management system
  - Configurable number of workers and queues
  - Priority-based task execution
  - ETA-based task scheduling

- **Monitoring & Recovery**
  - Worker health monitoring and heartbeats
  - Task state tracking
  - Stuck task detection and recovery
  - Comprehensive task validation
  - Service communication via pub/sub

- **Infrastructure**
  - Redis-based task queue and pub/sub system
  - MongoDB for task and job persistence
  - Configurable logging levels
  - Environment-based configuration
  - Database and broker authentication support
  - Graceful shutdown support

## Requirements

- Go 1.23.3 or later
- Redis 6.0 or later
- MongoDB 4.4 or later

## Installation

1. Clone the repository:
```bash
git clone https://github.com/joeg-ita/giobba.git
cd giobba
```

2. Install dependencies:
```bash
go mod download
```

3. Build the project:
```bash
go build
```

## Project Structure

```
giobba/
├── .github/           # GitHub workflows and templates
├── src/              # Source code
├── test/             # Test files and examples
├── giobba.go         # Main entry point
├── giobba.yaml       # Default configuration
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
└── README.md         # This file
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
jobsTimeoutRefresh: 30
pollingTimeout: 1
executionTimeCutOff: 300  # Maximum execution time in seconds

database:
  url: "mongodb://localhost:27017"
  db: giobba
  tasksCollection: tasks
  jobsCollection: jobs

broker:
  url: "redis://localhost:6379/0"

log:
  level: info  # debug, info, warn, error
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

- `ID`: Unique identifier for the task (UUID)
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
- `SchedulerID`: ID of the scheduler that processed the task
- `WorkerID`: ID of the worker that executed the task
- `Callback`: URL to call when task completes successfully (must be valid HTTP/HTTPS URL)
- `CallbackErr`: URL to call when task fails (must be valid HTTP/HTTPS URL)

### Job Properties

Jobs have the following properties for managing scheduled tasks:

- `ID`: Unique identifier for the job (UUID)
- `LastExecution`: Timestamp of the last job execution
- `NextExecution`: Timestamp of the next scheduled execution
- `Schedule`: Cron expression for the job schedule
- `TaskID`: ID of the associated task
- `TaskQueue`: Queue where the task will be processed
- `CreatedAt`: Job creation timestamp
- `UpdatedAt`: Last update timestamp
- `IsActive`: Whether the job is currently active

## Usage

### Creating Custom Task Handlers

To create a custom task handler, implement the `TaskHandlerInt` interface:

```go
type MyHandler struct {
    // Your custom fields here
}

func (m *MyHandler) Run(ctx context.Context, task domain.Task) services.HandlerResult {
    log.Printf("MyHandler processing task: %s", task.Name)

    // Access task payload
    payload := task.Payload

    // Your custom logic here
    // ...

    // Check for cancellation
    select {
    case <-ctx.Done():
        return services.HandlerResult{
            Err: fmt.Errorf("task cancelled"),
        }
    default:
        // Continue with work
    }

    return services.HandlerResult{
        Payload: map[string]interface{}{
            "result": "success",
        },
    }
}
```

Register your handler before starting Giobba:

```go
scheduler.RegisterHandler("myHandler", &MyHandler{})
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
    scheduler.RegisterHandler("myHandler", &MyHandler{})

    // Start giobba
    go giobba.Giobba()
    
    // Create a task
    task := domain.Task{
        Name: "myHandler",
        Payload: map[string]interface{}{
            "user": "john",
            "job":  "process_data",
        },
        Queue: "default",
        Priority: 5,
        StartMode: domain.AUTO,
        ETA: time.Now().Add(5 * time.Minute),
        MaxRetries: 3,
        Callback: "https://api.example.com/callback",
        CallbackErr: "https://api.example.com/error",
    }
    
    // Add the task
    taskID, err := tasker.AddTask(task)
    if err != nil {
        log.Fatalf("Failed to add task: %v", err)
    }
    
    log.Printf("Task added with ID: %s", taskID)
}
```

## Development

### Running Tests

```bash
go test ./...
```

### Running Linters

```bash
go vet ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
