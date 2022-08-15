package mempool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
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

func ListenForBlocks(newBlock chan models.NewBlock, network string, mempoolSpaceCtx context.Context) {
	if network == "mainnet" {
		network = ""
	}
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
		var objmap map[string]json.RawMessage
		err := json.Unmarshal(message, &objmap)
		if err != nil {
			log.Printf("\nerror while unmarshalling top level block object %s", err.Error())
			continue
		}
		var block models.Block
		errBlock := json.Unmarshal(objmap["block"], &block)
		if errBlock != nil {
			log.Printf("\nerror while unmarshalling block object %s", errBlock.Error())
			continue
		}
		log.Printf("\nnew block message %s", message)
		newBlock <- models.NewBlock{IsNew: true, Network: network, BlockHeight: block.Height}
	}
	ListenForBlocks(newBlock, network, mempoolSpaceCtx)
}

func SetupClient(network string, mempoolSpaceCtx context.Context) (*websocket.Conn, error) {
	if network == "mainnet" {
		network = ""
	}
	connnectionString := url.URL{
		Scheme: "wss",
		Host:   "mempool.space",
		Path:   fmt.Sprintf("%s/api/v1/ws", network),
	}
	urlStr := connnectionString.String()
	log.Printf("\nconnecting to mempool.space websocket: %s", urlStr)

	conn, _, err := websocket.DefaultDialer.DialContext(mempoolSpaceCtx, urlStr, nil)
	if err != nil {
		log.Printf("error connecting to mempool.space websocket %s", err.Error())
		return nil, err
	}
	errWrite := conn.WriteJSON(models.MempoolListen{Action: "want", Data: []string{"blocks"}})
	if errWrite != nil {
		log.Printf("error setting up listen for 'blocks' %s", errWrite.Error())
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

func ListenForUserTrans(set *utils.Set[models.WatchTx], watchTransaction chan models.WatchTx, newBlock chan models.NewBlock, slackClient *slack.Client, ctx context.Context) {
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

	go func(set *utils.Set[models.WatchTx], slackClient *slack.Client, newBlock chan models.NewBlock) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting down send watch message")
				return
			case newBlc := <-newBlock:
				log.Printf("New Block %v", newBlc)
				if newBlc.IsNew {
					SendMessageForWatched(set, newBlc.Network, newBlc.BlockHeight, slackClient)
				}
			default:
				time.Sleep(time.Second * 2)
			}
		}
	}(set, slackClient, newBlock)

}

func SendMessageForWatched(set *utils.Set[models.WatchTx], network string, curBlockHeight int, slackClient *slack.Client) {

	for _, watchTx := range set.Keys() {
		log.Printf("\nnetwork: %s watchTx: %v curBlockHeight: %d", network, watchTx, curBlockHeight)

		if watchTx.Network != network {
			log.Println("\n skipping")
			continue
		}

		if watchTx.ConfsCount > 0 && (watchTx.ConfsCount+1) != watchTx.Confs && watchTx.ConfirmBlockHeight < curBlockHeight {
			log.Printf("\nwatching transaction has confsCount >0: %v", watchTx)
			set.Remove(watchTx)
			log.Printf("\nwatchTx %v", watchTx)
			watchTx.ConfsCount = watchTx.ConfsCount + 1
			watchTx.ConfirmBlockHeight = curBlockHeight
			curTime := time.Now().UTC()
			log.Printf("\nwatchTx %v", watchTx)
			set.Add(watchTx, curTime.Format("20060102150405"))
			go SendUpdatedConfMessage(watchTx, slackClient)
		} else if watchTx.ConfsCount == 0 {
			log.Printf("watching transaction has confsCount = 0: %v", watchTx)
			//check if in recent block
			confirmed, err := CheckTransactionWasConfirmed(watchTx.TxID, watchTx.Network)
			if err != nil {
				log.Println(err)
				continue
			}
			if !confirmed.Confirmed {
				continue
			}
			log.Printf("confirmed results %v", confirmed)
			set.Remove(watchTx)
			watchTx.ConfsCount = watchTx.ConfsCount + 1
			watchTx.ConfirmBlockHeight = curBlockHeight
			curTime := time.Now().UTC()
			log.Printf("watchTx %v", watchTx)
			set.Add(watchTx, curTime.Format("20060102150405"))
			go SendFirstConfMessage(watchTx, *confirmed, slackClient)
		} else if (watchTx.ConfsCount + 1) == watchTx.Confs {
			log.Printf("removing watchTx %v", watchTx)
			watchTx.ConfsCount = watchTx.Confs
			go SendFinalMessage(watchTx, slackClient)
			set.Remove(watchTx)
		} else {
			log.Printf("\n didnt hit any matching")
		}
	}

	utils.RemoveOldItems(set, time.Now().UTC().Unix())
}

func CheckTransactionWasConfirmed(txId string, network string) (*models.ConfirmedPayload, error) {
	if network == "mainnet" {
		network = ""
	}
	mempoolSpaceUrl := ""
	if len(network) > 0 {
		lowerNetwork := strings.ToLower(network)
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
	log.Printf("\npayload %s", body)
	confirmed := &models.ConfirmedPayload{}
	errMarshal := json.Unmarshal(body, confirmed)
	if errMarshal != nil {
		log.Printf("failed to unmarshal body from response of mempool.space")
		return nil, errMarshal
	}
	return confirmed, nil
}

func GetLastBlockHeight(network string) (*int, error) {
	if network == "mainnet" {
		network = ""
	}
	mempoolSpaceUrl := ""
	if len(network) > 0 {
		lowerNetwork := strings.ToLower(network)
		mempoolSpaceUrl = fmt.Sprintf("https://mempool.space/%s/api/blocks/tip/height", lowerNetwork)
	} else {
		mempoolSpaceUrl = "https://mempool.space/api/blocks/tip/height"
	}
	resp, err := http.Get(mempoolSpaceUrl)
	if err != nil {
		log.Printf("failed to request out to mempool.space")
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	heightRaw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("failed to read body from response of mempool.space")
		return nil, readErr
	}
	log.Printf("\nLast Block Height %s", heightRaw)
	height := int(big.NewInt(0).SetBytes(heightRaw).Uint64())

	return &height, nil
}

func SendFirstConfMessage(watchTx models.WatchTx, confirmed models.ConfirmedPayload, slackClient *slack.Client) {
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("Your transaction %s has been picked up from the mempool and confirmed in block %s at %s! ", watchTx.TxID, *confirmed.BlockHash, utils.ConvertTimestamp(*confirmed.BlockTime))
	attachment.Color = "#4af030"
	_, _, err := slackClient.PostMessage(watchTx.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Printf("failed to post message: %s", err.Error())
	}
}

func SendUpdatedConfMessage(watchTx models.WatchTx, slackClient *slack.Client) {
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("Your transaction %s has moved up a confirmation %d", watchTx.TxID, watchTx.ConfsCount)
	attachment.Color = "#4af030"
	_, _, err := slackClient.PostMessage(watchTx.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Printf("failed to post message: %s", err.Error())
	}
}

func SendFinalMessage(watchTx models.WatchTx, slackClient *slack.Client) {
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("The transaction %s has moved up to your limit of confirmations %d and you will no longer be notified", watchTx.TxID, watchTx.ConfsCount)
	attachment.Color = "#4af030"
	_, _, err := slackClient.PostMessage(watchTx.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Printf("failed to post message: %s", err.Error())
	}
}
