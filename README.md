# Installation
1. ```apt install golang```
2. ```go get github.com/skip2/go-qrcode```
3. ```git clone https://git.sr.ht/~anon_/shadowchat```
4. ```cd shadowchat```
5. ```go run shadowchat```

A webserver at 127.0.0.1:8900 is running. Pressing the pay button will result in a 500 Error if the `monero-wallet-rpc` is not running.

# Monero Setup
1. Generate a view only wallet using the `monero-wallet-gui` from getmonero.org. Preferably with no password
2. Copy the newly generated `walletname_viewonly` and `walletname_viewonly.keys` files to your VPS
3. Download the `monero-wallet-rpc` binary that is bundled with the getmonero.org wallets.
4. Start the RPC wallet: `monero-wallet-rpc --rpc-bind-port 28088 --daemon-address https://xmr-node.cakewallet.com:18081 --wallet-file /opt/wallet/walletname_viewonly --disable-rpc-login --password ""`

# Usage
- Visit 127.0.0.1:8900/view to view your superchat history
- Paste 127.0.0.1:8900/alert?auth=adminadmin into OBS for an alert box
- The default username is `admin` and password `adminadmin`. Change these in `main.go`

# Future plans
- Use settings file instead of editing source
- Settings page for on-the-fly changes (minimum dono amount, hide all amounts, etc.)
- Blocklist for naughty words
- Widget for OBS displaying top donators

# License
GPLv3

### Origin
This comes from [https://git.sr.ht/~anon_/shadowchat](https://git.sr.ht/~anon_/shadowchat) and is not Luke's original work.

### Donate
sir,,thank you
`84U6xHT7KVaWqdKwc7LiwkAXKCS2f2g6b6SFyt1G7u6xWqLBYTVXH2aEsEPho64uPFJQS6KHqSg7XLEfEkqvjdgd9H1vQSm`

### Example
To see a working instance of shadowchat, see [xmr.lukesmith.xyz](https://xmr.lukesmith.xyz).
