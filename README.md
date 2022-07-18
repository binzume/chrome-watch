
# WIP: chrome-watch

ブラウザのタブを監視して，特定のURLを開いたときにスクリプトを実行するやつ．

- Chrome Extenstionが使えない環境で，色々な操作を自動化できます
- Oculus Quest上の Oculus(Meta Quest) Browserも操作できます
- ADBプロトコルもしゃべるのでAndroid端末に直接つながります

## Usage

設定ファイルの内容は examples 以下にあるものを参照して下さい．

```
go install github.com/binzume/chrome-watch@latest
```

### ADBで繋ぐ場合

対象デバイスのIPアドレスを確認して，ネットワーク経由でADBを使えるようにします．

```bash
adb shell ip -o address
adb tcpip 5555
```

ADBにつなぎます．PC上のadb-serverではなく，Androidデバイス上のadbdのポートを指定してください．

```
chrome-watch -adb 192.168.0.123:5555
```

Android上のadbdと直接通信するので `adb connect` や `adb forward` は不要です．(chrome-watchはAndroidのplatform-toolsに依存しません)


### 指定したソケットに繋ぐ場合


```bash
chrome-watch -ws ws://localhost:9222/devtools/browser
```

