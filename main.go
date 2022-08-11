package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/binzume/adbproto"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/gobwas/ws"
)

var userScripts []*UserScript

const recoonectInterval = 10 * time.Second

func Install(ctx context.Context, target *target.Info, script *UserScript, currentTarget bool) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if !currentTarget {
		ctx, cancel = chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID))
		defer cancel()
		defer func() {
			// FIXME: workaround to avoid the browser tab to be closed.
			if cc := chromedp.FromContext(ctx); cc != nil && cc.Target != nil {
				cc.Target.TargetID = ""
			}
		}()
	}

	var actions []chromedp.Action
	scriptParams := map[string]interface{}{}

	scriptStr, err := script.Read()
	if err != nil {
		log.Println("error: ", err)
		return err
	}

	if _, ok := script.Grants["cookie"]; ok {
		chromedp.Run(ctx)
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, _ := network.GetCookies().WithUrls([]string{target.URL}).Do(ctx)
			if len(cookies) > 0 {
				scriptParams["cookie"] = cookies
			}
			return chromedp.Evaluate(WrapScript(string(scriptStr), script.Name, scriptParams), nil).Do(ctx)
		}))

	}
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(WrapScript(string(scriptStr), script.Name, scriptParams), nil).Do(ctx)
	}))

	err = chromedp.Run(ctx, actions...)
	if err != nil {
		log.Println("error: ", err, target.URL)
	}
	return err
}

func CheckTarget(ctx context.Context, t *target.Info, currentTarget bool) {
	for _, s := range userScripts {
		if s.Match(t.URL) {
			go Install(ctx, t, s, currentTarget)
		}
	}
}

func Watch(parentCtx context.Context) error {
	ctx, cancel := chromedp.NewContext(parentCtx)
	defer cancel()
	attachedTargets := map[target.ID]bool{}

	chromedp.ListenBrowser(ctx, func(ev interface{}) {
		// log.Printf("ListenBrowser %#T", ev)
		if ev, ok := ev.(*target.EventTargetInfoChanged); ok {
			if ev.TargetInfo.Attached {
				attachedTargets[ev.TargetInfo.TargetID] = true
			}
			if !attachedTargets[ev.TargetInfo.TargetID] {
				CheckTarget(parentCtx, ev.TargetInfo, false)
			}
			if !ev.TargetInfo.Attached {
				// Do this after CheckTarget() to ignore detach event.
				delete(attachedTargets, ev.TargetInfo.TargetID)
			}
		}
	})
	targets, err := chromedp.Targets(ctx) // Also ensure initialize cc.Browser
	if err != nil {
		return err
	}
	cc := chromedp.FromContext(ctx)
	target.SetDiscoverTargets(true).Do(cdp.WithExecutor(ctx, cc.Browser))

	for _, t := range targets {
		for _, s := range userScripts {
			if !t.Attached && s.Match(t.URL) {
				Install(ctx, t, s, false)
			}
		}
	}

	select {
	case <-cc.Browser.LostConnection:
		log.Println("LostConnection")
		cancel()
	case <-ctx.Done():
	}
	return ctx.Err()
}

func GetTargets(allocatorCtx context.Context) ([]*target.Info, error) {
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()
	return chromedp.Targets(ctx)
}

func WatchLoop(ctx context.Context, wsUrl, scriptsDir string) error {
	allocatorCtx, cancel := chromedp.NewRemoteAllocator(ctx, wsUrl)
	defer cancel()

	for {
		start := time.Now()
		userScripts = ScanUserScript(scriptsDir)

		log.Println("attach")
		err := Watch(allocatorCtx)
		log.Println("detached", err)

		w := recoonectInterval - time.Now().Sub(start)
		if w < 1*time.Second {
			w = 1 * time.Second
		}
		time.Sleep(w)
	}
}

type streamWrapper struct {
	*adbproto.Stream
}

func (*streamWrapper) LocalAddr() net.Addr {
	return nil
}
func (*streamWrapper) RemoteAddr() net.Addr {
	return nil
}
func (*streamWrapper) SetDeadline(time.Time) error {
	return nil
}
func (*streamWrapper) SetReadDeadline(time.Time) error {
	return nil
}
func (*streamWrapper) SetWriteDeadline(time.Time) error {
	return nil
}

func StartAdbConnection(ctx context.Context, adbTarget string, key *rsa.PrivateKey) {
	const chromeDomainSocket = "localabstract:chrome_devtools_remote"

	connect := func() (*adbproto.Conn, error) {
		conn, err := net.Dial("tcp", adbTarget)
		if err != nil {
			return nil, err
		}
		adb, err := adbproto.Connect(conn, key)
		if err != nil {
			conn.Close()
			return nil, err
		}
		ws.DefaultDialer.NetDial = func(_ context.Context, _, _ string) (net.Conn, error) {
			stream, err := adb.Open(chromeDomainSocket)
			return &streamWrapper{stream}, err
		}
		go func() {
			<-adb.Closed()
			conn.Close()
		}()
		return adb, nil
	}

	connected := make(chan struct{})
	go func(connected chan<- struct{}) {
		for {
			adb, err := connect()
			if err != nil {
				log.Println("ADB: failed to connect.", err)
			} else {
				log.Println("ADB: connected.")
				if connected != nil {
					close(connected)
					connected = nil
				}
				select {
				case <-ctx.Done():
				case <-adb.Closed():
				}
				log.Print("ADB: disconnected")
				adb.Close()
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(recoonectInterval):
			}
		}
	}(connected)
	<-connected
}

func main() {
	wsUrl := flag.String("ws", "ws://localhost:9222/devtools/browser", "DevTools Socket URL")
	adb := flag.String("adb", "", "Connect via adb (host:port)")
	adbKey := flag.String("adbkey", "", "RSA Private key file for ADB (e.g. ~/.android/adbkey ) ")
	scriptsDir := flag.String("scripts", "./scripts", "User script dir")
	flag.Parse()

	ctx := context.Background()

	if *adb != "" {
		var key *rsa.PrivateKey
		if *adbKey != "" {
			pemData, err := ioutil.ReadFile(*adbKey)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("ADB: key: ", *adbKey)
			block, _ := pem.Decode(pemData)
			parseResult, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				log.Fatal(err)
			}
			key = parseResult.(*rsa.PrivateKey)
		}

		StartAdbConnection(ctx, *adb, key)
	}

	err := WatchLoop(ctx, *wsUrl, *scriptsDir)
	if err != nil {
		log.Println(err)
	}
}
