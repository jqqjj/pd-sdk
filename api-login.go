package sdk

import (
	"encoding/json"
	"github.com/hashicorp/go-multierror"
	"github.com/jqqjj/pd-sdk/request"
	"github.com/jqqjj/pd-sdk/response"
	"net/http"
)

func (c *Client) ApiLogin(req request.ApiLogin) (*response.ApiLogin, error) {
	var (
		err   error
		bytes []byte
		code  int
		resp  response.ApiLogin
	)

	if code, bytes, err = c.post("auth/login", req); err != nil {
		return nil, multierror.Append(err, ErrNetwork)
	}
	if code != http.StatusOK {
		return nil, ErrNetwork
	}

	if err = json.Unmarshal(bytes, &resp); err != nil {
		return nil, multierror.Append(err, ErrNetwork)
	}

	if resp.Data.Status != nil && !*resp.Data.Status {
		return nil, ErrAuth
	}

	if !resp.Data.Result || resp.Data.IdUser == 0 || len(resp.Data.RefreshToken) == 0 {
		return nil, ErrAuth
	}

	//判断是否需要验证
	if _, err = c.ApiCheckVerification(); err != nil {
		return nil, err
	}

	//获取 social ID
	if c.socialId, err = c.ApiGetSocialId(); err != nil {
		return nil, err
	}

	//开启ws
	if err = c.startWS(); err != nil {
		return nil, err
	}

	//保存一些必要信息
	c.userId = resp.Data.IdUser
	c.refreshToken = resp.Data.RefreshToken
	c.username = req.Email
	c.password = req.Password

	return &resp, nil
}
