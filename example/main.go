package main

import (
	"context"
	"errors"
	"fmt"
	sdk "github.com/jqqjj/pd-sdk"
	"github.com/jqqjj/pd-sdk/request"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	go func() {
		for {
			log.Errorln("协程数：", runtime.NumGoroutine())
			time.Sleep(time.Second * 5)
		}
	}()

	go func() {
		for {
			t2()
			time.Sleep(time.Second * 5)
		}
	}()

	t()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGHUP)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	<-sig

	log.Info("EXIT")
}

func aa() {
	ctx, cancel := context.WithCancel(context.Background())
	client := sdk.NewClient(
		ctx,
		"https://talkytimes.com/platform/",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	)
	if _, err := client.ApiLogin(request.ApiLogin{
		Email:        "123456@qq.com",
		Password:     "123456",
		ReferralCode: 123456,
		Captcha:      "",
	}); err != nil {
		log.Errorln("aa", err)
		return
	}

	go func() {
		time.Sleep(time.Second * 2)
		cancel()
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			runtime.GC()
		}
	}()
}

func t() {
	var (
		ctx, cancel = context.WithCancel(context.Background())
	)
	defer cancel()

	client := sdk.NewClient(
		ctx,
		"https://talkytimes.com/platform/",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	)
	_, err := client.ApiLogin(request.ApiLogin{
		Email:        "123456@qq.com",
		Password:     "123456",
		ReferralCode: 123456,
		Captcha:      "",
	})

	if errors.Is(err, sdk.ErrNetwork) {
		fmt.Println("1", err)
		return
	}
	if errors.Is(err, sdk.ErrAuth) {
		fmt.Println("2", err)
		return
	}
	if err != nil {
		fmt.Println("3", err)
		return
	}

	ch := make(chan []byte)
	client.Subscribe(ctx, "streaming_StreamStarted", ch)
	client.Subscribe(ctx, "video-call_SessionStarted", ch)
	client.Subscribe(ctx, sdk.EventDialogCreated, ch)
	client.Subscribe(ctx, sdk.EventMessageRead, ch)
	client.Subscribe(ctx, sdk.EventMessageSent, ch)
	client.Subscribe(ctx, sdk.EventDialogLimitChanged, ch)
	//client.Subscribe(ctx, "pong", ch)

	for {
		select {
		case <-ctx.Done():
			return
		case v := <-ch:
			fmt.Printf("收到消息:%s\n", string(v))
		}
	}
}

func t2() {
	ctx, _ := context.WithCancel(context.Background())
	client := sdk.NewClient(
		ctx,
		"https://talkytimes.com/platform/",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	)
	if _, err := client.ApiLogin(request.ApiLogin{
		Email:        "123456@qq.com",
		Password:     "123456",
		ReferralCode: 123456,
		Captcha:      "",
	}); err != nil {
		log.Errorln("aa", err)
		return
	}

	go func() {
		time.Sleep(time.Second * 5)
		client.Close()
	}()
}
