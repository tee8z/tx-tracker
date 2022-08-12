package mempool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"tx-tracker/pkg/models"
	"tx-tracker/pkg/utils"

	"github.com/gorilla/websocket"
	"github.com/slack-go/slack"
)

func ListenForBlocks(newBlock chan bool, network string, mempoolSpaceCtx context.Context) {
	closeByApi, errRegex := regexp.Compile(`(close 1006 \(abnormal closure\))`)
	if errRegex != nil {
		log.Fatal(errRegex)
	}
	conn, err := SetupClient(network, mempoolSpaceCtx)
	if err != nil {
		return
	}
	log.Printf("listening to the blocks event from mempool.space")
	for {
		_, message, errRead := conn.ReadMessage()
		if errRead != nil {

			parseErr := closeByApi.Find([]byte(errRead.Error()))
			if parseErr != nil {
				log.Printf("api killed the connection, re-creating and continuing to listen")
				break
			}
			log.Printf("error while reading message from mempool.space: %s", errRead.Error())
			return
		}
		log.Printf("\nmessage %s", message)
		newBlock <- true
	}
	ListenForBlocks(newBlock, network, mempoolSpaceCtx)
}

func SetupClient(network string, mempoolSpaceCtx context.Context) (*websocket.Conn, error) {
	connnectionString := url.URL{
		Scheme: "wss",
		Host:   "mempool.space",
		Path:   fmt.Sprintf("%s/api/v1/ws", network),
	}
	urlStr := connnectionString.String()
	log.Printf("\nconnecting to mempool.space websocket: %s", urlStr)

	conn, _, err := websocket.DefaultDialer.DialContext(mempoolSpaceCtx, urlStr, nil)
	if err != nil {
		fmt.Printf("error connecting to mempool.space websocket %s", err.Error())
		return nil, err
	}
	errWrite := conn.WriteJSON(models.MempoolListen{Action: "want", Data: []string{"blocks"}})
	if errWrite != nil {
		fmt.Printf("error setting up listen for 'blocks' %s", errWrite.Error())
		return nil, errWrite
	}
	log.Println("setting up listen for 'blocks' from mempool.space websocket")
	KeepAlive(conn, time.Second*120)

	return conn, nil
}

func KeepAlive(c *websocket.Conn, timeout time.Duration) {
	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		for {
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				return
			}
			log.Println("Websocket ping message sent to mempool.space")
			time.Sleep(timeout / 2)
			if time.Since(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}

func ListenForUserTrans(watchTransaction chan models.WatchTx, newBlock chan bool, slackClient *slack.Client, ctx context.Context) {
	set := utils.NewSet[models.WatchTx]()
	go func(set *utils.Set[models.WatchTx], watchTransaction chan models.WatchTx) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting down watch transaction listener")
				return
			case newTransaction := <-watchTransaction:
				log.Printf("New Transaction %v", newTransaction)
				if !set.Contains(newTransaction) {
					curTime := time.Now().UTC()
					set.Add(newTransaction, curTime.Format("20060102150405"))
				}
			}
		}
	}(set, watchTransaction)

	go func(set *utils.Set[models.WatchTx], slackClient *slack.Client, newBlock chan bool) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting down watch transaction listener")
				return
			case newBlc := <-newBlock:
				log.Printf("New Block %v", newBlc)
				if newBlc {
					SendMessageForWatched(set, slackClient)
				}
			default:
				time.Sleep(time.Second * 2)
			}
		}
	}(set, slackClient, newBlock)

}

func SendMessageForWatched(set *utils.Set[models.WatchTx], slackClient *slack.Client) {

	for _, watchTx := range set.Keys() {
		if watchTx.ConfsCount > 0 && watchTx.Confs < watchTx.ConfsCount+1 {
			log.Printf("watching transaction has confsCount >0: %v", watchTx)
			set.Remove(watchTx)
			watchTx.ConfsCount++
			curTime := time.Now().UTC()
			set.Add(watchTx, curTime.Format("20060102150405"))
			go SendUpdatedConfMessage(watchTx, slackClient)
		} else if watchTx.ConfsCount == 0 {
			log.Printf("watching transaction has confsCount = 0: %v", watchTx)
			//check if in recent block
			confirmed, err := CheckTransactionWasConfirmed(watchTx.TxId, watchTx.Network)
			if err != nil {
				log.Println(err)
				continue
			}
			if !confirmed.Confirmed {
				continue
			}
			log.Printf("confirmed results %v", confirmed)
			set.Remove(watchTx)
			watchTx.ConfsCount++
			go SendFirstConfMessage(watchTx, *confirmed, slackClient)
			curTime := time.Now().UTC()
			set.Add(watchTx, curTime.Format("20060102150405"))
		} else {
			log.Printf("removing watchTx %v", watchTx)
			set.Remove(watchTx)
		}
	}

}

func CheckTransactionWasConfirmed(txId string, network *string) (*models.ConfirmedPayload, error) {

	mempoolSpaceUrl := ""
	if network != nil {
		lowerNetwork := strings.ToLower(*network)
		mempoolSpaceUrl = fmt.Sprintf("https://mempool.space/%s/api/tx/%s/status", txId, lowerNetwork)
	} else {
		mempoolSpaceUrl = fmt.Sprintf("https://mempool.space/api/tx/%s/status", txId)
	}
	resp, err := http.Get(mempoolSpaceUrl)
	if err != nil {
		log.Printf("failed to request out to mempool.space")
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("failed to read body from response of mempool.space")
		return nil, readErr
	}
	confirmed := &models.ConfirmedPayload{}
	errMarshal := json.Unmarshal(body, confirmed)
	if errMarshal != nil {
		log.Printf("failed to unmarshal body from response of mempool.space")
		return nil, errMarshal
	}
	return confirmed, nil
}

func SendFirstConfMessage(watchTx models.WatchTx, confirmed models.ConfirmedPayload, slackClient *slack.Client) {
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("Your transaction %s has been picked up from the mempool and confirmed in block %s at %s! ", watchTx.TxId, confirmed.BlockHash, utils.ConvertTimestamp(confirmed.BlockTime))
	attachment.Color = "#4af030"
	_, _, err := slackClient.PostMessage(watchTx.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Printf("failed to post message: %s", err.Error())
	}
}

func SendUpdatedConfMessage(watchTx models.WatchTx, slackClient *slack.Client) {
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("Your transaction %s has moved up a confirmation %d", watchTx.TxId, watchTx.ConfsCount)
	attachment.Color = "#4af030"
	_, _, err := slackClient.PostMessage(watchTx.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Printf("failed to post message: %s", err.Error())
	}
}
