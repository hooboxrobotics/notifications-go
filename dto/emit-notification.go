package dto

type EmitNotification struct {
	IdCompany        string                 `json:"id_company"`
	ExpiresInSeconds int                    `json:"expires_in_seconds"`
	Pulse            int                    `json:"pulse"`
	Payload          map[string]interface{} `json:"payload"`
	Subject          string                 `json:"subject"`
}
