# Installation
```apt install golang```
```go get github.com/skip2/go-qrcode```
```git clone https://git.sr.ht/~anon_/shadowchat```
```cd shadowchat```
```go run shadowchat```
A webserver at 127.0.0.1:8900 is running. Pressing the pay button will result in a 500 Error if the `monero-wallet-rpc` is not running.

# Monero Setup
1. Generate a view only wallet using the `monero-wallet-gui` from getmonero.org. Preferably with no password
2. Copy the newly generated `walletname_viewonly` and `walletname_viewonly.keys` files to your VPS
3. Download the `monero-wallet-rpc` binary that is bundled with the getmonero.org wallets.
4. Run `monero-wallet-rpc`: `monero-wallet-rpc --rpc-bind-port 28088 --daemon-address https://xmr-node.cakewallet.com:18081 --wallet-file /opt/wallet/walletname_viewonly --disable-rpc-login --password ""`
