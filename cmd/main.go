package main

import (
	"context"

	"log"
	"os"
	"tx-tracker/pkg/mempool"
	"tx-tracker/pkg/models"
	slackUtils "tx-tracker/pkg/slack"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

func main() {

	godotenv.Load(".env")
	token := os.Getenv("SLACK_AUTH_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	newBlock := make(chan bool)
	watchTransaction := make(chan models.WatchTx)

	mempoolSpaceCtx, cancelMempoolSpace := context.WithCancel(context.Background())
	defer cancelMempoolSpace()
	defer close(newBlock)
	go mempool.ListenForBlocks(newBlock, "", mempoolSpaceCtx)        //mainnet
	go mempool.ListenForBlocks(newBlock, "testnet", mempoolSpaceCtx) //testnet
	go mempool.ListenForBlocks(newBlock, "signet", mempoolSpaceCtx)  //signet

	slackClient := slack.New(token, slack.OptionDebug(true), slack.OptionAppLevelToken(appToken))
	listenUserTransCtx, cancelUserListen := context.WithCancel(context.Background())
	defer cancelUserListen()
	go mempool.ListenForUserTrans(watchTransaction, newBlock, slackClient, listenUserTransCtx)

	socketClient := socketmode.New(
		slackClient,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	slackContext, slackCancel := context.WithCancel(context.Background())
	defer slackCancel()
	go slackUtils.ListenForSlackMessages(slackContext, slackClient, socketClient, watchTransaction)

	errRun := socketClient.Run()
	if errRun != nil {
		log.Fatal(errRun)
	}
}
