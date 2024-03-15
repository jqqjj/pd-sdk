package sdk

import (
	"encoding/json"
	"github.com/hashicorp/go-multierror"
	"net/http"
)

func (c *Client) ApiCheckVerification() (bool, error) {
	var (
		err  error
		code int

		bytes []byte

		resp struct {
			Data struct {
				Status string `json:"status"` // 非not_need表示需要验证才能登录
			} `json:"data"`
		}
	)

	if code, bytes, err = c.post("verification-request/status", nil); err != nil {
		return false, multierror.Append(err, ErrNetwork)
	}
	if code != http.StatusOK {
		return false, ErrNetwork
	}

	if err = json.Unmarshal(bytes, &resp); err != nil {
		return false, multierror.Append(err, ErrNetwork)
	}

	if resp.Data.Status != "not_need" {
		return false, ErrNeedVerification
	}
	return true, nil
}
