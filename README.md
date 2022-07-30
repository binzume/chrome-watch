
# chrome-watch

Chromiumベースのブラウザを[Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)で監視して，特定のURLにアクセスしたときにスクリプトを実行します．
Android版ChromeやOculus Questのブラウザなど拡張機能をインストールできない環境でユーザースクリプトを実行できます．

- Chrome Extenstionが使えない環境で，色々な操作を自動化できます
- Oculus Quest上の Oculus(Meta Quest) Browser も操作できます
- ADBプロトコルに対応しているのでAndroid端末上のChromeに直接接続できます

## Usage

Go 1.18以降が必要です．

インストール：

```bash
go install github.com/binzume/chrome-watch@latest
```

### User Script

Greasemonkeyとよく似たフォーマットのスクリプトをscriptsフォルダに置くことで実行できます．scripts 以下にサンプルスクリプトがあります．

例：
```js
// ==UserScript==
// @name         RedText
// @match        https://www.binzume.net/*
// ==/UserScript==

let styleEl = document.createElement("style");
styleEl.innerText = "body{color:red !important}";
document.head.appendChild(styleEl);
```

### ADBで接続する場合

対象デバイスのIPアドレスを確認して，ネットワーク経由でADBを使えるようにします．

```bash
adb shell ip -o address
adb tcpip 5555
```

ADBにつなぎます．PC上のadb-serverではなく，Androidデバイス上のadbdのポートを指定してください．

```bash
chrome-watch -adb 192.168.0.123:5555
```

Android上のadbdと直接通信するので `adb connect` や `adb forward` は不要です．またこのツール自体はAndroidのplatform-toolsに依存しないので，ADBコマンドがインストールされていない環境でも起動できます．
`-adbkey` オプションでADB用のRSA鍵ファイルを渡すとADBの接続確認ダイアログが毎回表示されるのを避けられます．


### 指定したソケットに接続する場合

Dev Tools Protocolを有効にしてChromwを起動してください(Androidの場合は adb forward してください)．

```bash
chrome.exe --remote-debugging-port=9222
```

```bash
chrome-watch -ws ws://localhost:9222/devtools/browser
```
