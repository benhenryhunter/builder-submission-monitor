package main

type blockSubmission struct {
	Slot                 string `json:"slot"`
	ParentHash           string `json:"parent_hash"`
	BlockHash            string `json:"block_hash"`
	BuilderPubkey        string `json:"builder_pubkey"`
	ProposerPubkey       string `json:"proposer_pubkey"`
	ProposerFeeRecipient string `json:"proposer_fee_recipient"`
	GasLimit             string `json:"gas_limit"`
	GasUsed              string `json:"gas_used"`
	Value                string `json:"value"`
	NumTx                string `json:"num_tx"`
	BlockNumber          string `json:"block_number"`
	TimestampMs          string `json:"timestamp_ms"`
	Timestamp            string `json:"timestamp"`
}

type payloadAttributeEvent struct {
	Data payloadAttributeData `json:"data"`
}

type payloadAttributeData struct {
	ProposalSlot    uint64 `json:"proposal_slot,string"`
	ParentBlockHash string `json:"parent_block_hash"`
	ProposerIndex   string `json:"proposer_index"`
}

type validatorData struct {
	Index     string    `json:"index"`
	Balance   string    `json:"balance"`
	Status    string    `json:"status"`
	Validator validator `json:"validator"`
}

type validator struct {
	Pubkey string `json:"pubkey"`
}

type validatorResponse struct {
	Data []validatorData `json:"data"`
}
