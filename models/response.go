package models

type Response struct {
	Status      string      `json:"status"`
	ErrorCode   string      `json:"error_code"`
	Description string      `json:"description"`
	Payload     interface{} `json:"payload"`
}
