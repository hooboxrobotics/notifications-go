package dto

type Ack struct {
	IdNotification string `json:"id_notification"`
	IdClient       string `json:"id_client"`
	IdCompany      string `json:"id_company"`
}
