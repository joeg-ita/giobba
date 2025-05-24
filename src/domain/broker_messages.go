package domain

import "encoding/json"

type ActionType string

const (
	ACTIVITY   ActionType = "ACTIVITY"
	AUTO_TASK  ActionType = "AUTO_TASK"
	KILL       ActionType = "KILL"
	REVOKE     ActionType = "REVOKE"
	CHECK_TASK ActionType = "CHECK_TASK"
)

type ServiceMessage struct {
	Action  ActionType             `json:"action_type"`
	Payload map[string]interface{} `json:"payload"`
}

func NewServiceMessageFromJsonString(message string) (*ServiceMessage, error) {
	svcMessage := ServiceMessage{}
	err := json.Unmarshal([]byte(message), &svcMessage)
	if err != nil {
		return nil, err
	}
	return &svcMessage, nil
}
