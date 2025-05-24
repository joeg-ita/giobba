package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/joeg-ita/giobba/src/domain"

	"github.com/redis/go-redis/v9"
)

const (
	HEARTBEATS_CHANNEL = "giobba-heartbeats"
)

type RedisBroker struct {
	client *redis.Client
}

func NewRedisBrokerByOptions(options *redis.Options) *RedisBroker {
	client := redis.NewClient(options)
	return &RedisBroker{
		client: client,
	}
}

func NewRedisBroker(address string, password string, db int, protocol int) *RedisBroker {
	opt := &redis.Options{
		Addr:     address,  // "localhost:6379"
		Password: password, // No password set
		DB:       db,       // Use default DB
		Protocol: protocol, // Connection protocol ie: 2
	}
	return NewRedisBrokerByOptions(opt)
}

func NewRedisBrokerByUrl(url string) *RedisBroker {
	// url = "redis://<user>:<pass>@localhost:6379/<db>"
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}
	return NewRedisBrokerByOptions(opt)
}

func (r *RedisBroker) AddTask(ctx context.Context, task domain.Task, queue string) (string, error) {
	now := time.Now()
	task.UpdatedAt = now

	return r.SaveTask(ctx, task, queue)
}

func (r *RedisBroker) SaveTask(ctx context.Context, task domain.Task, queue string) (string, error) {
	log.Printf("Saving task %v in queue %v", task.ID, queue)

	now := time.Now()
	task.UpdatedAt = now

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return "", err
	}
	err = r.client.HSet(ctx, task.Queue, task.ID, taskJSON).Err()
	if err != nil {
		return "", fmt.Errorf("error while adding task to queue: %w", err)
	}
	return task.ID, nil
}

func (r *RedisBroker) DeleteTask(ctx context.Context, taskId string, queue string) error {
	err := r.client.HDel(ctx, queue, taskId).Err()
	if err != nil {
		return fmt.Errorf("error while deleting task to queue: %w", err)
	}
	return nil
}

func (r *RedisBroker) GetTask(ctx context.Context, taskId string, queue string) (domain.Task, error) {
	taskJSON, err := r.client.HGet(ctx, queue, taskId).Result()
	if err == redis.Nil {
		return domain.Task{}, fmt.Errorf("task not found: %w", err)
	}
	var task domain.Task
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return domain.Task{}, fmt.Errorf("deserialization error: %w", err)
	}
	return task, nil
}

func (r *RedisBroker) UnSchedule(ctx context.Context, taskId string, queue string, withWildcards bool) error {
	log.Printf("unschedule taskId %v with queue %v", taskId, queue)

	key := taskId

	if withWildcards {
		scanResult, _, err := r.client.ZScan(ctx, queue, 0, taskId, 10).Result()

		if err != nil {
			return err
		}
		if len(scanResult) == 0 {
			return fmt.Errorf("unschedule of taskId %v with queue %v using wildcards fail", taskId, queue)
		}
		if len(scanResult) > 2 {
			return fmt.Errorf("unschedule of taskId %v with queue %v using wildcards found to many results", taskId, queue)
		}
		key = strings.Split(scanResult[0], " ")[0]
	}

	_, err := r.client.ZRem(ctx, queue, key).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisBroker) Schedule(ctx context.Context, task domain.Task, queue string) error {
	score := float64(task.ETA.Unix())
	if score < float64(time.Now().Unix()) {
		score = float64(time.Now().Unix())
	}
	priority := float64(10-task.Priority) / (100000.0 / math.Pow(float64(10-task.Priority), 2))
	finalScore := score + priority

	r.client.ZAdd(ctx, queue, redis.Z{
		Score:  finalScore,
		Member: fmt.Sprintf("%s::%s::%f", task.ID, queue, finalScore),
	})

	return nil
}

func (r *RedisBroker) GetScheduled(ctx context.Context, queue string) ([]string, error) {
	now := time.Now()
	maxScore := float64(now.Add(10 * time.Millisecond).Unix())

	taskIDs, err := r.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     queue,
		Start:   "-inf",
		Stop:    fmt.Sprintf("%f", maxScore),
		Offset:  0,
		Count:   100,
		Rev:     false,
		ByScore: true,
	}).Result()

	if err != nil {
		return nil, err
	}

	return taskIDs, nil
}

func (r *RedisBroker) Lock(ctx context.Context, taskId string, queue string, lockDuration time.Duration) bool {
	lockKey := queue + "-LOCK-" + taskId

	success, err := r.client.SetNX(ctx, lockKey, "1", lockDuration).Result()
	if err != nil {
		log.Printf("Errore nell'acquisizione del lock per il task %s: %v", taskId, err)
		return false
	}

	return success
}

func (r *RedisBroker) RenewLock(ctx context.Context, taskId string, queue string, lockDuration time.Duration) bool {
	lockKey := queue + "-LOCK-" + taskId
	log.Printf("Renewing lock for task %s", taskId)

	success, err := r.client.SetXX(ctx, lockKey, "1", lockDuration).Result()
	if err != nil {
		log.Printf("Errore nel rinnovo del lock per il task %s: %v", taskId, err)
		return false
	}

	return success
}

func (r *RedisBroker) UnLock(ctx context.Context, taskId string, queue string) error {
	lockKey := queue + "-LOCK-" + taskId
	_, err := r.client.Del(ctx, lockKey).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisBroker) Close() {
	r.client.Close()
}

func (r *RedisBroker) Subscribe(ctx context.Context, channels ...string) interface{} {
	return r.client.Subscribe(ctx, channels...)
}

func (r *RedisBroker) Publish(ctx context.Context, channel string, message domain.ServiceMessage) error {
	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = r.client.Publish(ctx, channel, msg).Result()
	return err
}
