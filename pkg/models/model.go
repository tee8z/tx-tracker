package models

type MempoolListen struct {
	Action string   `json:"action"`
	Data   []string `json:"data"`
}

type NewBlock struct {
	IsNew       bool   `json:"is_new"`
	Network     string `json:"network"`
	BlockHeight int    `json:"block_height"`
}

type WatchTx struct {
	TxID               string `json:"txId"`
	Confs              int    `json:"confs"`
	Network            string `json:"network"`
	Channel            string `json:"channel"`
	ConfsCount         int    `json:"confs_count"`
	ConfirmBlockHeight int    `json:"confirm_block_height"`
	TimeRequested      int64  `json:"time_requested"`
}

type ConfirmedPayload struct {
	Confirmed   bool    `json:"confirmed"`
	BlockHeight *int    `json:"block_height"`
	BlockHash   *string `json:"block_hash"`
	BlockTime   *int    `json:"block_time"`
}

type Block struct {
	Extras            *Extras `json:"extras"`
	Id                string  `json:"id"`
	Height            int     `json:"height"`
	Version           int     `json:"version"`
	Timestamp         int     `json:"timestamp"`
	Bits              int     `json:"bits"`
	Nonce             int     `json:"nonce"`
	Difficulty        float64 `json:"difficulty"`
	MerkleRoot        string  `json:"merkle_root"`
	TxCount           int     `json:"tx_count"`
	Size              int     `json:"size"`
	Weight            int     `json:"weight"`
	Previousblockhash string  `json:"previousblockhash"`
}

type Extras struct {
	Reward      *int        `json:"reward"`
	CoinbaseTx  *CoinbaseTx `json:"coinbaseTx"`
	CoinbaseRaw *string     `json:"coinbaseRaw"`
	Usd         *float64    `json:"usd"`
	MedianFee   *int        `json:"medianFee"`
	FeeRange    []int       `json:"feeRange"`
	TotalFees   int         `json:"totalFees"`
	AvgFee      int         `json:"avgFee"`
	AvgFeeRate  int         `json:"avgFeeRate"`
	Pool        *Pool       `json:"pool"`
	MatchRate   float64     `json:"matchRate"`
}
type CoinbaseTx struct {
	Vins  []Vin  `json:"vin"`
	Vouts []Vout `json:"vout"`
}

type Vin struct {
	Scriptsig *string `json:"scriptsig"`
}

type Vout struct {
	ScriptPubkeyAddress *string `json:"scriptpubkey_address"`
}

type Pool struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
