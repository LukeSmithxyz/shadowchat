# Installation
1. ```apt install golang```
2. ```go get github.com/skip2/go-qrcode```
3. ```git clone https://git.sr.ht/~anon_/shadowchat```
4. ```cd shadowchat```
5. ```go run shadowchat```

A webserver at 127.0.0.1:8900 is running. Pressing the pay button will result in a 500 Error if the `monero-wallet-rpc` is not running.
This is designed to be run on a cloud server with nginx proxypass for TLS.

# Monero Setup
1. Generate a view only wallet using the `monero-wallet-gui` from getmonero.org. Preferably with no password
2. Copy the newly generated `walletname_viewonly` and `walletname_viewonly.keys` files to your VPS
3. Download the `monero-wallet-rpc` binary that is bundled with the getmonero.org wallets.
4. Start the RPC wallet: `monero-wallet-rpc --rpc-bind-port 28088 --daemon-address https://xmr-node.cakewallet.com:18081 --wallet-file /opt/wallet/walletname_viewonly --disable-rpc-login --password ""`

# Usage
- Visit 127.0.0.1:8900/view to view your superchat history
- Visit 127.0.0.1:8900/alert?auth=adminadmin to see notifications
- The default username is `admin` and password `adminadmin`. Change these in `main.go`

# OBS
- Add a Browser source in obs and point it to `https://example.com/alert?auth=adminadmin`
# Future plans
- Use settings file instead of editing source
- Settings page for on-the-fly changes (minimum dono amount, hide all amounts, etc.)
- Blocklist for naughty words
- Widget for OBS displaying top donators
- Remove discord and streamlabs integration features

# License
GPLv3

### Donate
sir,,thank you
`84U6xHT7KVaWqdKwc7LiwkAXKCS2f2g6b6SFyt1G7u6xWqLBYTVXH2aEsEPho64uPFJQS6KHqSg7XLEfEkqvjdgd9H1vQSm`
