package dto

type Notification struct {
	Token              string `json:"Token"`
	Destination        string `json:"Destination"`
	Created_At         string `json:"Created_At"`
	Expires_In_Seconds int    `json:"Expires_In_Seconds"`
	Expired            bool   `json:"Expired"`
	Pulse              int    `json:"Pulse"`
}
