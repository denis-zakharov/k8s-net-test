package main

type svcPayload struct {
	Svc   string `json:"svc"`
	Count int    `json:"count"`
}

type directPayloadItem struct {
	Hostname string   `json:"hostname"`
	Addrs    []string `json:"addrs"`
}
