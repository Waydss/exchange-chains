package rejection

const rejectionRecordedStatus = "rejection_recorded"

// DeclineChainRequest contains the item that caused the chain rejection.
type DeclineChainRequest struct {
	ReceivedItemID int64 `json:"received_item_id" example:"205"`
}

// DeclineChainResponse confirms that an item rejection was recorded.
type DeclineChainResponse struct {
	ChainID        int64  `json:"chain_id" example:"77"`
	ReceivedItemID int64  `json:"received_item_id" example:"205"`
	Status         string `json:"status" example:"rejection_recorded"`
}
