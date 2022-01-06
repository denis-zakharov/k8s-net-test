package main

type svcReqPayload struct {
	SvcURL string `json:"svcURL"`
	Count  int    `json:"count"`
}

type svcRespPayload struct {
	SrcHost string `json:"srcHost"`
	Errors  int    `json:"errors"`
}

type directReqPayloadItem struct {
	Hostname string   `json:"hostname"`
	Addrs    []string `json:"addrs"`
}
