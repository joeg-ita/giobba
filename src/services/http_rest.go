package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type HttpRest struct {
}

func NewHttpRest() *HttpRest {
	return &HttpRest{}
}

func (h *HttpRest) Post(url string, payload map[string]interface{}) error {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	body := bytes.NewReader(jsonData)

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)
	return nil
}
