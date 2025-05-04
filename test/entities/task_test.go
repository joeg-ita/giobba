package entities

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/entities"
)

func TestValidateTaskRequiredFields(t *testing.T) {

	fmt.Println("TestValidateTaskRequiredFields...")

	now := time.Now()
	queue := "default"

	payload := map[string]interface{}{
		"user": "tizio",
		"job":  "process",
	}

	_, err := entities.NewTask("process", payload, queue, now, 5, entities.AUTO, "")

	log.Printf("Task validate %v", err)
	if err != nil {
		t.Error(err.Error())
	}

}

func TestValidateTaskId(t *testing.T) {

	fmt.Println("TestValidateTaskId...")

	now := time.Now()
	queue := "default"

	payload := map[string]interface{}{
		"user": "tizio",
		"job":  "process",
	}

	task, _ := entities.NewTask("process", payload, queue, now, 5, entities.AUTO, "")

	task.ID = task.ID[:8]

	log.Printf("Task validate %v", task.ID)

	if task.Validate() == nil {
		t.Error(task.Validate().Error())
	}

}
