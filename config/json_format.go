package config

type JsonFormat struct {
	ComponentName string `json:"componentName"`
	InterfaceName string `json:"interfaceName"`
	CostTime int `json:"costTime"`
	ReturnCode int `json:"returnCode"`
	Timestamp int `json:"timestamp"`
	Event string `json:"event"`
}