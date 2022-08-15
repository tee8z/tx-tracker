## Bitcoin Transaction Watcher Slack Bot
Slack bot that will watch the bitcoin transactions based on ID
- can watch on mainnet/testnet/signet, will use mainnet as default
- has a dependency on mempool.space to keep track of the transactions and be notified when a new block comes in

Follow along with this tutorial for how to include this bot into your channel & get the required .env values (the needed permissions will be the same as the 'Slack Events API Call' bot): 
https://www.bacancytechnology.com/blog/develop-slack-bot-using-golang

### Build:
- `git clone git@github.com:tee8z/tx-tracker.git`
- `go mod tidy`
### Install:
 - ``
### Run Bot: 
(make sure to change default.env to .env & update the values first):
- `go run cmd/tx-tracker/main.go`

#### How to use in channel:
- bot command options:
    - asking to watch a bitcoin transaction on mainnet for 3 confirmations:
    
    - asking to watch a bitcoin transaction on testnet for 3 confirmations:

##### NOTE:
- If the bot goes down, the state of all transactions being watched will be saved in a .bin file & it will be reloaded on the next successful startup. This data is deleted as the transaction's # of confirmations have passed or 2 weeks have passed since the request occured.
