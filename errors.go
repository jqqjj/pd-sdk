package sdk

import "errors"

var (
	ErrAuth             = errors.New("auth fail error")
	ErrNetwork          = errors.New("network error")
	ErrWebsocketNetwork = errors.New("websocket network error")
	ErrWebsocketAuth    = errors.New("websocket auth error")
	ErrNeedVerification = errors.New("need verification")
)
