package request

type ApiLogin struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	ReferralCode int    `json:"referral_code,omitempty"`
	Captcha      string `json:"captcha"`
}
