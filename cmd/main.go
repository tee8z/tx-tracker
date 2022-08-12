package main

import (
	"context"
	"os/signal"

	"log"
	"os"
	"tx-tracker/pkg/mempool"
	"tx-tracker/pkg/models"
	slackUtils "tx-tracker/pkg/slack"
	"tx-tracker/pkg/utils"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

func HandleSignals[T comparable](cancel func(), fileName string, toSave *utils.Set[T]) {
	// register signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// handle signals
	go func() {

		// if the signal is caught more than once we will force quit.
		var forced bool

		for sig := range c {
			if !forced {
				utils.Save(fileName, toSave)
				log.Printf("Shutting down bot (%v)", sig)
				cancel()
			} else {
				log.Printf("Forcing bot to shutdown")
				os.Exit(1)
			}

			forced = true
		}
	}()
}

func main() {

	//load .env variables
	godotenv.Load(".env")
	token := os.Getenv("SLACK_AUTH_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	filename := os.Getenv("SAVE_FILE")
	set := utils.NewSet[models.WatchTx]()

	//load state of saved transactions
	errLoad := utils.Load(filename, set)
	if errLoad != nil {
		log.Fatalf(errLoad.Error())
	}

	newBlock := make(chan models.NewBlock)
	watchTransaction := make(chan models.WatchTx)

	mempoolSpaceCtx, cancelMempoolSpace := context.WithCancel(context.Background())
	defer cancelMempoolSpace()
	defer close(newBlock)

	//setup to gracefully handle shutdown from interupt signal
	HandleSignals(cancelMempoolSpace, filename, set)

	slackClient := slack.New(token, slack.OptionDebug(true), slack.OptionAppLevelToken(appToken))
	listenUserTransCtx, cancelUserListen := context.WithCancel(mempoolSpaceCtx)
	defer cancelUserListen()

	//request initial state of saved transactions after a restart
	if len(set.Keys()) > 0 {
		mempool.SendMessageForWatched(set, nil, slackClient)
		testnet := "testnet"
		mempool.SendMessageForWatched(set, &testnet, slackClient)
		signet := "signet"
		mempool.SendMessageForWatched(set, &signet, slackClient)
	}
	//listen for new blocks on each chain
	go mempool.ListenForBlocks(newBlock, "", mempoolSpaceCtx)        //mainnet
	go mempool.ListenForBlocks(newBlock, "testnet", mempoolSpaceCtx) //testnet
	go mempool.ListenForBlocks(newBlock, "signet", mempoolSpaceCtx)  //signet

	//update watched transactions as new block come in
	go mempool.ListenForUserTrans(set, watchTransaction, newBlock, slackClient, listenUserTransCtx)

	socketClient := socketmode.New(
		slackClient,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	slackContext, slackCancel := context.WithCancel(mempoolSpaceCtx)
	defer slackCancel()

	//listen for new slack messages and add transactions to ones that are watched
	go slackUtils.ListenForSlackMessages(slackContext, slackClient, socketClient, watchTransaction)

	errRun := socketClient.RunContext(mempoolSpaceCtx)
	if errRun != nil {
		log.Fatal(errRun)
	}

}
