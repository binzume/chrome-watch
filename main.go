package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"gopkg.in/yaml.v3"

	"github.com/chromedp/cdproto/inspector"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type Settings struct {
	Scripts []*struct {
		Prefix     string
		ScriptPath string
	}
}

var settings Settings

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

func Install(ctx context.Context, target target.ID, scriptPath string, currentTarget bool) error {
	script, _ := ioutil.ReadFile(scriptPath)

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if !currentTarget {
		var cc *chromedp.Context
		ctx, cancel = chromedp.NewContext(ctx, chromedp.WithTargetID(target), func(c *chromedp.Context) { cc = c })
		defer cancel()
		defer func() {
			// FIXME: workaround to avoid the browser tab to be closed.
			if cc != nil && cc.Target != nil {
				cc.Target.TargetID = ""
			}
		}()
	}

	// time.Sleep(500 * time.Millisecond)
	var err error
	var res []byte

	err = chromedp.Run(ctx,
		chromedp.Evaluate(string(script), &res),
	)
	log.Println("result: ", string(res), err)

	return err
}

func CheckTarget(ctx context.Context, t *target.Info, currentTarget bool) bool {
	for _, s := range settings.Scripts {
		if strings.HasPrefix(t.URL, s.Prefix) {
			log.Println("install", s.ScriptPath, t.URL)
			go Install(ctx, t.TargetID, s.ScriptPath, currentTarget)
			return true
		}
	}
	return false
}

func Watch(parentCtx context.Context, tid target.ID) error {
	log.Println("attach", tid)
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
	<-done
	log.Println("detached")
	return err
}

func WatchLoop(wsUrl string) error {
	allocatorContext, cancel := chromedp.NewRemoteAllocator(context.Background(), wsUrl)
	defer cancel()

	ctxt, cancel := chromedp.NewContext(allocatorContext)
	defer cancel()

	for {
		targets, err := chromedp.Targets(ctxt)
		if err != nil {
			return err
		}

		for _, t := range targets {
			if !t.Attached {
				err = Watch(ctxt, t.TargetID)
				break
			}
		}
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}

var dummyKey = []byte("QAAAACGAUUQf/OGDf7YmxHuyrgnf/dkDcHHdteq+kfz7r2y4GmUS9A3GuXcF0VGNKbg25QbAtAF4yLOR0o26LMv7VZf/0gJg9xe44ATio/biy8DT48G6A26DUTjObK95kPK3UBmGpRVQBGitVB/FnjYiBgr28C833y+v7ltI6cizgLbS2+5hQ65FgFK+tsHVvGR7xniM9GhvRSztixhlxtAJ3jyjLtC/4Q7XKtYu5OyuomcyW+zGH133JdspLe2RWgToxM0lOrSu12XhOrs3ySqYVaMufayYedrzi8KQF/sBcPU1+dX20Ko/kJTX4vS75QLRREMTi3I0Sv3kSvV2yZAnuDTDiuPJexVWDVaFW1tN5/p97Ot3+Rh8o2+5JjYwxDd4n7/LoguNMAqboDTphD2qWJiP4Mko7MmAlUN0YkmnoraSV/oNDILF6CWA4cQK1axW9Y1hXsT0bs+3UcXSkdZbS2Fpdc9YuDydMgXSluTWzrT5hRr7js9w5WChRnokcLOMJWylYSPYjEBNSQS1ZbNRhk7J71EZn1gwogDPtkSNEXnTtmxcgLchoFnrhIrkeTbH9CtRb9JGmJni5ZwxSa39Zh3LqBvvaFMfSdnww9/HOr45HEziN+nNwnOPgPeN8Mx7we2pZIyDef5rJkkVTkKYIDR099KCGZmDwxWQaMaGJJaT4kZyRwEAAQA=")
var chromeDomainSocket = "localabstract:chrome_devtools_remote"

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
	wsUrl := flag.String("ws", "ws://localhost:9222/devtools/browser", "DevTools Socket URL")
	adb := flag.String("adb", "", "Connect via adb (host:port)")
	flag.Parse()

	b, err := ioutil.ReadFile("settings.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(b, &settings)
	if err != nil {
		log.Fatal(err)
	}

	if *adb != "" {
		conn, err := net.Dial("tcp", *adb)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()
		adb, err := Connect(conn, dummyKey)
		if err != nil {
			log.Fatal(err)
		}
		defer adb.Close()
		ws.DefaultDialer.NetDial = func(ctx context.Context, network, host string) (net.Conn, error) {
			stream, err := adb.Open(chromeDomainSocket)
			return &streamWrapper{stream}, err
		}
	}

	err = WatchLoop(*wsUrl)
	if err != nil {
		log.Println(err)
	}
}
