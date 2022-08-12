package slack

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"tx-tracker/pkg/models"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func ListenForSlackMessages(ctx context.Context, client *slack.Client, socketClient *socketmode.Client, watchTransaction chan models.WatchTx) {
	defer close(watchTransaction)
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down socketmode listener")
			return
		case event := <-socketClient.Events:
			log.Printf("event %#v", event)
			switch event.Type {

			case socketmode.EventTypeEventsAPI:

				eventsAPI, ok := event.Data.(slackevents.EventsAPIEvent)
				if !ok {
					log.Printf("Could not type cast the event to the EventsAPI: %v\n", event)
					continue
				}

				socketClient.Ack(*event.Request)
				log.Println(eventsAPI)
				err := HandleEventMessage(eventsAPI, client, watchTransaction)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func HandleEventMessage(event slackevents.EventsAPIEvent, client *slack.Client, watchTransaction chan models.WatchTx) error {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch evnt := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			err := HandleAppMentionEventToBot(evnt, client, watchTransaction)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported event type")
	}
	return nil
}

func HandleAppMentionEventToBot(event *slackevents.AppMentionEvent, client *slack.Client, watchTransaction chan models.WatchTx) error {

	watchTx, errConv := ParseMessage(event.Text)

	if watchTx != nil {
		watchTx.Channel = event.Channel
		watchTransaction <- *watchTx
	}

	attachment := slack.Attachment{}
	if errConv != nil {
		attachment.Text = fmt.Sprintf("Failed to setup watcher, check your format? %s", errConv)
		attachment.Color = "#ef3232"
	} else {
		attachment.Text = fmt.Sprintf("Your transaction %s is being watched and you will be notified of each block until %d confirmations have occured", watchTx.TxId, watchTx.Confs)
		attachment.Color = "#4af030"
	}
	_, _, err := client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}

func ParseMessage(rawMessage string) (*models.WatchTx, error) {

	transactionMatch, errTransRegex := regexp.Compile("(txId: [0-9|a-z|A-Z]+)")
	if errTransRegex != nil {
		return nil, errTransRegex
	}
	confirmsMatch, errConfRegex := regexp.Compile("(confirms: [0-9|a-z|A-Z]+)")
	if errConfRegex != nil {
		return nil, errConfRegex
	}
	networkMatch, errNetworkReges := regexp.Compile("(network: [a-z|A-Z]+)")
	if errNetworkReges != nil {
		return nil, errNetworkReges
	}
	transactionText := transactionMatch.Find([]byte(rawMessage))

	var txId *string
	if transactionText != nil {
		rawId := strings.Split(string(transactionText), ": ")[1]
		txId = &rawId
	}
	networkText := networkMatch.Find([]byte(rawMessage))

	var network *string = nil
	if networkText != nil {
		rawNetwork := strings.Split(string(networkText), ": ")[1]
		network = &rawNetwork
	}
	confirmText := confirmsMatch.Find([]byte(rawMessage))

	var confirms *int
	if confirmText != nil {
		rawConfirms := strings.Split(string(confirmText), ": ")[1]
		convConfirms, err := strconv.Atoi(rawConfirms)
		if err != nil {
			return nil, err
		}
		confirms = &convConfirms
	}

	if txId != nil {
		if confirms == nil {
			defaultCount := 6
			confirms = &defaultCount
		}
		return &models.WatchTx{
			TxId:       *txId,
			Confs:      *confirms,
			ConfsCount: 0,
			Network:    network,
		}, nil
	} else {
		return nil, fmt.Errorf("a txId is required, in the format: 'txId: <transacitonId to watch>'")
	}
}
