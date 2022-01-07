package model

type SvcReqPayload struct {
	SvcURL string `json:"svcURL"`
	Count  int    `json:"count"`
}

type SvcRespPayload struct {
	SrcHost string `json:"srcHost"`
	Errors  int    `json:"errors"`
}

type DirectReqPayloadItem struct {
	Hostname string   `json:"hostname"`
	Addrs    []string `json:"addrs"`
}

type DirectRespPayloadItem struct {
	SrcHost string `json:"srcHost"`
	DstHost string `json:"dstHost"`
	Addr    string `json:"addr"`
	Error   string `json:"error"`
}
