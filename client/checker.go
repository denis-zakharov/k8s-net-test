package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/denis-zakharov/k8s-net-test/model"
)

const jsonContent = "application/json"

type Checker struct {
	http.Client
}

func NewChecker() *Checker {
	return &Checker{http.Client{Timeout: 5 * time.Second}}
}

func (c *Checker) Svc(url string, payload *model.SvcReqPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := c.Post(url, jsonContent, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("doGet: response failed with status code %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var respPayload model.SvcRespPayload
	err = json.Unmarshal(data, &respPayload)
	if err != nil {
		return err
	}
	if respPayload.Errors > 0 {
		log.Printf("svc check error: %#+v\n", respPayload)
		return nil
	}
	log.Printf("svc check: %#+v", respPayload)
	log.Printf("svc check: OK")
	return nil
}

func (c *Checker) Direct(url string, payload []model.DirectReqPayloadItem) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := c.Post(url, jsonContent, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("doGet: response failed with status code %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var respPayload []model.DirectRespPayloadItem
	err = json.Unmarshal(data, &respPayload)
	if err != nil {
		return err
	}
	if len(respPayload) > 0 {
		log.Printf("direct check error: %#+v\n", respPayload)
		return nil
	}
	log.Printf("direct check: %#+v", respPayload)
	log.Printf("direct check: OK")
	return nil
}
