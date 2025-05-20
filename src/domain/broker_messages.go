package domain

import "encoding/json"

type ServiceMessage struct {
	Action  string                 `json:"action"`
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
