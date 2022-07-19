
# chrome-watch

Chromeiumベースのブラウザを監視して，特定のURLにアクセスしたときにスクリプトを実行します．
Android版ChromeやOculus Questのブラウザなど拡張機能をインストールできない環境でユーザースクリプトを実行する代替手段として実装されています．

- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/) で通信します
- Chrome Extenstionが使えない環境で，色々な操作を自動化できます
- Oculus Quest上の Oculus(Meta Quest) Browserも操作できます
- ADBプロトコルもしゃべるのでAndroid端末に直接つながります

## Usage

インストール：

```bash
go install github.com/binzume/chrome-watch@latest
```

スクリプトの書き方は scripts 以下にあるものを参照して下さい．

### ADBで繋ぐ場合

対象デバイスのIPアドレスを確認して，ネットワーク経由でADBを使えるようにします．

```bash
adb shell ip -o address
adb tcpip 5555
```

ADBにつなぎます．PC上のadb-serverではなく，Androidデバイス上のadbdのポートを指定してください．

```bash
chrome-watch -adb 192.168.0.123:5555
```

Android上のadbdと直接通信するので `adb connect` や `adb forward` は不要です．またこのツール自体はADBコマンドがインストールされていない環境でも起動できます(chrome-watchはAndroidのplatform-toolsに依存しません)
`-adbkey` オプションでADB用のRSA鍵ファイルを渡すとADBの接続確認ダイアログが毎回表示されるのを避けられます．


### 指定したソケットに繋ぐ場合

Dev Tools Protocolを有効にしてChromwを起動してください(Androidの場合は adb forwardしてください)．

```bash
chrome.exe --remote-debugging-port=9222
```

```bash
chrome-watch -ws ws://localhost:9222/devtools/browser
```

