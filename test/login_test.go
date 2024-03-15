package test

import (
	"context"
	"errors"
	"fmt"
	sdk "github.com/jqqjj/pd-sdk"
	"github.com/jqqjj/pd-sdk/request"
	"testing"
)

func TestLogin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := sdk.NewClient(
		ctx,
		"https://talkytimes.com/platform/",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	)
	resp, err := client.ApiLogin(request.ApiLogin{
		Email:        "",
		Password:     "",
		ReferralCode: 0,
		Captcha:      "",
	})

	if errors.Is(err, sdk.ErrNetwork) {
		fmt.Println(err)
		t.Error("网络错误")
		return
	}
	if errors.Is(err, sdk.ErrAuth) {
		t.Error("认证失败")
		fmt.Println(err)
		return
	}
	if err != nil {
		t.Errorf("其他错误:%v", err)
	}

	fmt.Println(resp.Data.RefreshToken)
}
