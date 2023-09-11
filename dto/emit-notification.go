package dto

type EmitNotification struct {
	Id_Company         string `json:"id_company"`
	Destination        string `json:"destination"`
	Expires_In_Seconds int    `json:"expires_in_seconds"`
	Pulse              int    `json:"pulse"`
}
