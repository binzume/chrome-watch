package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gobwas/ws"

	"github.com/chromedp/cdproto/inspector"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

var userScripts []*UserScript

type Tab struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

func GetTabs(jsonUrl string) ([]*Tab, error) {
	var tabs []*Tab

	res, err := http.Get(jsonUrl)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", res.StatusCode)
	}
	jsonBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonBody, &tabs); err != nil {
		return nil, err
	}

	return tabs, nil
}

const SCRIPT_PREFIX = `if (!globalThis._chromeWatchScripts?.["%[1]s"]) {(globalThis._chromeWatchScripts??={})["%[1]s"] = true; let f=async (CW=%[2]s)=>{`
const SCRIPT_SUFFIX = `};document.readyState=='loading'?document.addEventListener('DOMContentLoaded',f,{once:true}):f();}`

func WrapScript(script, name string, params map[string]interface{}) string {
	json, err := json.Marshal(params)
	if err != nil {
		json = []byte("{}")
	}
	return fmt.Sprintf(SCRIPT_PREFIX, name, string(json)) + script + SCRIPT_SUFFIX
}

func Install(ctx context.Context, target *target.Info, script *UserScript, currentTarget bool) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if !currentTarget {
		var cc *chromedp.Context
		ctx, cancel = chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID), func(c *chromedp.Context) { cc = c })
		defer cancel()
		defer func() {
			// FIXME: workaround to avoid the browser tab to be closed.
			if cc != nil && cc.Target != nil {
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
		log.Println("error: ", err)
	} else {
		log.Println("ok.")
	}
	return err
}

func CheckTarget(ctx context.Context, t *target.Info, currentTarget bool) bool {
	for _, s := range userScripts {
		if s.Match(t.URL) {
			log.Println("install", s.Name, t.URL)
			go Install(ctx, t, s, currentTarget)
			return true
		}
	}
	return false
}

func Watch(parentCtx context.Context, tid target.ID) error {
	var cc *chromedp.Context
	ctx, cancel := chromedp.NewContext(parentCtx, chromedp.WithTargetID(tid), func(c *chromedp.Context) { cc = c })
	defer cancel()
	defer func() {
		// FIXME: workaround to avoid the browser tab to be closed.
		if cc != nil && cc.Target != nil {
			cc.Target.TargetID = ""
		}
	}()
	done := make(chan interface{})
	var detached sync.Once
	attachedTargets := map[target.ID]bool{}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		// log.Printf("tab event %#T", ev)
		if _, ok := ev.(*inspector.EventDetached); ok {
			detached.Do(func() { close(done) })
		}
		if _, ok := ev.(*inspector.EventTargetCrashed); ok {
			detached.Do(func() { close(done) })
		}
		if ev, ok := ev.(*target.EventTargetInfoChanged); ok {
			// log.Printf("%#v", ev.TargetInfo)
			if ev.TargetInfo.Attached {
				attachedTargets[ev.TargetInfo.TargetID] = true
			}
			if ev.TargetInfo.TargetID == tid {
				CheckTarget(ctx, ev.TargetInfo, true)
			} else if !attachedTargets[ev.TargetInfo.TargetID] {
				CheckTarget(ctx, ev.TargetInfo, false)
			}
			if !ev.TargetInfo.Attached {
				delete(attachedTargets, ev.TargetInfo.TargetID)
			}
		}
	})
	err := chromedp.Run(ctx)
	if err != nil {
		detached.Do(func() { close(done) })
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
	return err
}

func GetTargets(allocatorCtx context.Context) ([]*target.Info, error) {
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()
	return chromedp.Targets(ctx)
}

func WatchLoop(wsUrl, scriptsDir string) error {
	const recoonectInterval = 10 * time.Second
	allocatorCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), wsUrl)
	defer cancel()

	for {
		start := time.Now()
		userScripts = ScanUserScript(scriptsDir)

		targets, err := GetTargets(allocatorCtx)
		if err != nil {
			log.Println("failed to get targets ", err)
			time.Sleep(recoonectInterval)
			continue
		}

		for _, t := range targets {
			for _, s := range userScripts {
				if !t.Attached && s.Match(t.URL) {
					log.Println("install", s.Name, t.URL)
					Install(allocatorCtx, t, s, false)
					break
				}
			}
		}

		for _, t := range targets {
			if !t.Attached && t.Type == "page" {
				log.Println("attach", t.TargetID, t.URL)
				err = Watch(allocatorCtx, t.TargetID)
				log.Println("detached", err)
				break
			}
		}

		w := recoonectInterval - time.Now().Sub(start)
		if w < 1*time.Second {
			w = 1 * time.Second
		}
		time.Sleep(w)
	}
}

type streamWrapper struct {
	*Stream
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

func main() {
	const chromeDomainSocket = "localabstract:chrome_devtools_remote"

	wsUrl := flag.String("ws", "ws://localhost:9222/devtools/browser", "DevTools Socket URL")
	adb := flag.String("adb", "", "Connect via adb (host:port)")
	adbKey := flag.String("adbkey", "", "RSA Private key file for ADB (e.g. ~/.android/adbkey ) ")
	scriptsDir := flag.String("scripts", "./scripts", "User script dir")
	flag.Parse()

	if *adb != "" {
		var key *rsa.PrivateKey
		if *adbKey != "" {
			pemData, err := ioutil.ReadFile(*adbKey)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("adb key: ", *adbKey)
			block, _ := pem.Decode(pemData)
			parseResult, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				log.Fatal(err)
			}
			key = parseResult.(*rsa.PrivateKey)
		}

		conn, err := net.Dial("tcp", *adb)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()
		adb, err := Connect(conn, key)
		if err != nil {
			log.Fatal(err)
		}
		defer adb.Close()
		ws.DefaultDialer.NetDial = func(_ context.Context, _, _ string) (net.Conn, error) {
			stream, err := adb.Open(chromeDomainSocket)
			return &streamWrapper{stream}, err
		}
	}

	err := WatchLoop(*wsUrl, *scriptsDir)
	if err != nil {
		log.Println(err)
	}
}
