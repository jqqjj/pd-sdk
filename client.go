package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type Client struct {
	ctx       context.Context
	baseUrl   string
	userAgent string

	userId       int
	username     string
	password     string
	refreshToken string

	socialId string
	closed   bool

	client *http.Client

	conn    *websocket.Conn
	connMux sync.Mutex

	subMux sync.Mutex
	subs   map[string][]struct {
		ctx context.Context
		ch  chan []byte
	}
}

func NewClient(ctx context.Context, baseUrl string, userAgent string) *Client {
	rand.Seed(time.Now().UnixMicro())
	jar, _ := cookiejar.New(nil)
	c := &Client{
		ctx:       ctx,
		baseUrl:   baseUrl,
		userAgent: userAgent,
		client:    &http.Client{Jar: jar, Timeout: time.Second * 30},

		subs: make(map[string][]struct {
			ctx context.Context
			ch  chan []byte
		}),
	}
	//注册析构函数
	runtime.SetFinalizer(c, func(c *Client) {
		if c.conn != nil {
			c.conn.Close()
		}
	})
	return c
}

func (c *Client) get(uri string) (int, []byte, error) {
	return c.request("GET", uri, nil)
}

func (c *Client) post(uri string, data interface{}) (int, []byte, error) {
	return c.request("POST", uri, data)
}

func (c *Client) request(method, uri string, data interface{}) (int, []byte, error) {
	var (
		err error

		requestBytes  []byte
		responseBytes []byte

		r    *http.Request
		resp *http.Response
	)

	if c.closed {
		panic("do not call a closed client")
	}

	reqUrl := strings.Trim(c.baseUrl, "/") + "/" + strings.Trim(uri, "/")

	if method == "POST" {
		if data != nil {
			if requestBytes, err = json.Marshal(data); err == nil {
				r, err = http.NewRequest(method, reqUrl, bytes.NewBuffer(requestBytes))
			}
		} else {
			r, err = http.NewRequest(method, reqUrl, nil)
		}
	} else {
		r, err = http.NewRequest(method, reqUrl, nil)
	}

	if err != nil {
		return 0, nil, err
	}

	r.Header.Add("content-type", "application/json;charset=utf-8")
	r.Header.Add("accept", "application/json;charset=utf-8")
	if len(c.userAgent) > 0 {
		r.Header.Add("user-agent", c.userAgent)
	}
	if resp, err = c.client.Do(r); err != nil {
		return 0, nil, err
	}
	responseBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, responseBytes, err
}

func (c *Client) startWS() error {
	var (
		err  error
		conn *websocket.Conn
	)

	if conn, err = c.openConn(); err != nil {
		return err
	}
	c.conn = conn

	//循环消息处理
	go c.loopMessage()

	go c.loopPing() //定时发送ping

	//处理退出
	go func() {
		<-c.ctx.Done()
		c.Close()
	}()
	return nil
}

func (c *Client) loopMessage() {
	var (
		err    error
		data   []byte
		strArr []string
		conn   *websocket.Conn
	)
	for {
		var resp struct {
			Result struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			} `json:"result"`
		}
		if _, data, err = c.conn.ReadMessage(); err != nil {
			c.publish(EventWSDisconnected, nil) //断开时触发事件
			if c.closed {
				return
			}
			//非关闭的客户端发生错误时重连
			log.Errorf("[%d-%s]ws read error: %v, try to reconnect\n", c.userId, c.username, err)
			for {
				if conn, err = c.openConn(); err != nil {
					log.Errorf("[%d-%s]reconnect err:%v\n", c.userId, c.username, err)
					select {
					case <-c.ctx.Done():
						return
					case <-time.After(time.Second * 3):
						continue
					}
				}

				if success := func() bool {
					c.connMux.Lock()
					defer c.connMux.Unlock()

					if c.closed {
						conn.Close()
						return false
					}
					c.conn.Close()
					c.conn = conn
					return true
				}(); !success {
					log.Errorf("[%d-%s]reconnect fail cause of closed\n", c.userId, c.username)
					return
				}
				c.publish(EventWSReconnected, nil) //重新连上时触发事件
				log.Infof("[%d-%s]reconnected\n", c.userId, c.username)
				break
			}
			continue
		}

		//广播订阅消息
		if strArr, err = c.parseResp(data); err != nil {
			log.Errorf("[%d-%s]parse ws resp err:%v, data:%s\n", c.userId, c.username, err, string(data))
		}
		for _, v := range strArr {
			if err = json.Unmarshal([]byte(v), &resp); err != nil {
				log.Errorf("[%d-%s]parse ws data error:%v, data:%s\n", c.userId, c.username, err, v)
				continue
			}
			c.publish(resp.Result.Type, resp.Result.Data)
		}
	}
}

func (c *Client) openConn() (*websocket.Conn, error) {
	var (
		err    error
		data   []byte
		uri    string
		strArr []string

		urlInfo *url.URL
		conn    *websocket.Conn

		serverId = fmt.Sprintf("%03d", rand.Intn(999))

		dial = websocket.Dialer{
			HandshakeTimeout:  time.Second * 15,
			EnableCompression: false,
		}

		authResp struct {
			Result struct {
				Register bool   `json:"register"`
				Key      string `json:"key"`
			} `json:"result"`
		}
	)

	if urlInfo, err = url.Parse(c.baseUrl); err != nil {
		return nil, err
	}
	if strings.HasPrefix(c.baseUrl, "https://") {
		uri = fmt.Sprintf("wss://%s/push/%s/%s/websocket", urlInfo.Host, serverId, c.stringRandom(8))
	} else {
		uri = fmt.Sprintf("ws://%s/push/%s/%s/websocket", urlInfo.Host, serverId, c.stringRandom(8))
	}

	if conn, _, err = dial.DialContext(c.ctx, uri, nil); err != nil {
		return nil, multierror.Append(err, ErrWebsocketNetwork)
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	//发送认证信息 & 确认认证成功
	if err = conn.SetReadDeadline(time.Now().Add(time.Second * 3)); err != nil {
		return nil, multierror.Append(err, ErrWebsocketNetwork)
	}
	_, _, _ = conn.ReadMessage() //读取忽略点第一个数据包
	authStr := fmt.Sprintf(`["{\"method\":\"register\",\"params\":{\"tabId\":\"unknown\",\"key\":\"%s\",\"userId\":null}}"]`, c.socialId)
	if err = conn.WriteMessage(websocket.TextMessage, []byte(authStr)); err != nil {
		return nil, multierror.Append(err, ErrWebsocketNetwork)
	}
	if err = conn.SetReadDeadline(time.Now().Add(time.Second * 15)); err != nil {
		return nil, multierror.Append(err, ErrWebsocketNetwork)
	}
	if _, data, err = conn.ReadMessage(); err != nil {
		return nil, multierror.Append(err, ErrWebsocketNetwork)
	}
	//处理响应
	if strArr, err = c.parseResp(data); err != nil {
		return nil, multierror.Append(err, ErrWebsocketAuth)
	}
	if len(strArr) != 1 {
		err = ErrWebsocketAuth
		return nil, err
	}
	data = []byte(strArr[0])

	if err = json.Unmarshal(data, &authResp); err != nil {
		return nil, multierror.Append(err, ErrWebsocketAuth)
	}
	if !authResp.Result.Register || authResp.Result.Key == "" {
		err = ErrWebsocketAuth
		return nil, err
	}

	//重置超时时间
	if err = conn.SetReadDeadline(time.Time{}); err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Client) loopPing() {
	var (
		err      error
		duration = time.Minute * 1
		ticker   = time.NewTicker(duration)
	)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
		}

		func() {
			c.connMux.Lock()
			defer c.connMux.Unlock()

			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if err = c.conn.WriteMessage(websocket.TextMessage, []byte(`["{\"method\":\"ping\"}"]`)); err != nil {
				log.Errorf("[%d-%s]send ping error: %v\n", c.userId, c.username, err)
			}
			c.conn.SetWriteDeadline(time.Time{})
		}()
	}
}

func (c *Client) parseResp(data []byte) ([]string, error) {
	var (
		err       error
		jsonArray []string
	)
	if strings.HasPrefix(string(data), "a[") {
		data = data[1:]
		if err = json.Unmarshal(data, &jsonArray); err != nil {
			return nil, err
		}
		return jsonArray, nil
	} else {
		return []string{}, nil
	}
}

func (c *Client) stringRandom(n int) string {
	letters := "1234567890abcdefghijklmnopqrstuvwxyz"
	src := rand.NewSource(time.Now().UnixNano())
	letterIdBits := 6
	var letterIdMask int64 = 1<<letterIdBits - 1
	letterIdMax := 63 / letterIdBits

	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(letters) {
			b[i] = letters[idx]
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return *(*string)(unsafe.Pointer(&b))
}

func (c *Client) publish(t string, data []byte) {
	c.subMux.Lock()
	defer c.subMux.Unlock()

	if subs, ok := c.subs[t]; ok {
		for _, v := range subs {
			go func(ctx context.Context, ch chan []byte) {
				select {
				case <-ctx.Done():
					return
				case ch <- data:
				}
			}(v.ctx, v.ch)
		}
	}
}

func (c *Client) Subscribe(ctx context.Context, topic string, ch chan []byte) {
	c.subMux.Lock()
	defer c.subMux.Unlock()

	if _, ok := c.subs[topic]; !ok {
		c.subs[topic] = make([]struct {
			ctx context.Context
			ch  chan []byte
		}, 0)
	}

	c.subs[topic] = append(c.subs[topic], struct {
		ctx context.Context
		ch  chan []byte
	}{ctx: ctx, ch: ch})

	go func() {
		<-ctx.Done()

		c.subMux.Lock()
		defer c.subMux.Unlock()

		index := -1
		for i, v := range c.subs[topic] {
			if v.ch == ch {
				index = i
				break
			}
		}
		if index > -1 {
			copy(c.subs[topic][index:], c.subs[topic][index+1:])
			c.subs[topic] = c.subs[topic][:len(c.subs[topic])-1]
		}
		if len(c.subs[topic]) == 0 {
			delete(c.subs, topic)
		}
	}()
}

func (c *Client) Close() {
	c.connMux.Lock()
	defer c.connMux.Unlock()

	c.closed = true
	if c.conn != nil {
		c.conn.Close()
	}
}
