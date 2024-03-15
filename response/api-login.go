package response

type ApiLoginData struct {
	Status       *bool  `json:"status"`
	Result       bool   `json:"result"`
	IdUser       int    `json:"idUser"`
	RefreshToken string `json:"refreshToken"`
}

type ApiLogin struct {
	Data ApiLoginData `json:"data"`
}
