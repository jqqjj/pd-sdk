package sdk

import (
	"encoding/json"
	"github.com/hashicorp/go-multierror"
	"net/http"
)

func (c *Client) ApiGetSocialId() (string, error) {
	var (
		err error

		bytes []byte
		code  int

		resp struct {
			Data struct {
				PushId string `json:"pushId"`
			} `json:"data"`
		}
	)

	if code, bytes, err = c.post("social/get-id", nil); err != nil {
		return "", multierror.Append(err, ErrNetwork)
	}
	if code != http.StatusOK {
		return "", ErrNetwork
	}

	if err = json.Unmarshal(bytes, &resp); err != nil {
		return "", multierror.Append(err, ErrNetwork)
	}

	if resp.Data.PushId == "" {
		return "", ErrNetwork
	}
	return resp.Data.PushId, nil
}
