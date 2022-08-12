package models

type MempoolListen struct {
	Action string   `json:"action"`
	Data   []string `json:"data"`
}

type NewBlock struct {
	IsNew   bool    `json:"isNew"`
	Network *string `json:"network"`
}

type WatchTx struct {
	TxId       string `json:"txId"`
	Confs      int    `json:"confs"`
	Network    string `json:"network"`
	Channel    string `json:"channel"`
	ConfsCount int    `json:"confsCount"`
}

type ConfirmedPayload struct {
	Confirmed   bool    `json:"confirmed"`
	BlockHeight *int    `json:"block_height"`
	BlockHash   *string `json:"block_hash"`
	BlockTime   *int    `json:"block_time"`
}
