### Bitcoin Transaction Watcher Slack Bot
Slack bot that will watch the bitcoin transactions based on ID
- can watch on mainnet/testnet/signet, will use mainnet as default
- has a dependency on mempool.space to keep track of the transactions and be notified when a new block comes in

Follow along with the tutoiral for how to include this bot into your channel: 
https://www.bacancytechnology.com/blog/develop-slack-bot-using-golang

Install:
- `git clone git@github.com:tee8z/tx-tracker.git`
- `go mod tidy`

Run (make sure to change default.env to .env & update the values first):
- `go run cmd/main.go`
