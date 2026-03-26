package api

// OutEbicsTransaction exposes the technical view of an EBICS transaction.
type OutEbicsTransaction struct {
	ID             int64  `json:"id" yaml:"id"`
	TransactionID  string `json:"transactionID" yaml:"transactionID"`
	OrderType      string `json:"orderType" yaml:"orderType"`
	Status         string `json:"status" yaml:"status"`
	Direction      string `json:"direction" yaml:"direction"`
	SegmentCount   int    `json:"segmentCount" yaml:"segmentCount"`
	CurrentSegment int    `json:"currentSegment" yaml:"currentSegment"`
	TotalSize      int64  `json:"totalSize" yaml:"totalSize"`
	TransferID     *int64 `json:"transferID,omitempty" yaml:"transferID,omitempty"`
}

// OutEbicsTransactionSegment exposes the technical view of a transaction segment.
type OutEbicsTransactionSegment struct {
	ID               int64  `json:"id" yaml:"id"`
	SegmentNumber    int    `json:"segmentNumber" yaml:"segmentNumber"`
	SegmentStatus    string `json:"segmentStatus" yaml:"segmentStatus"`
	PayloadSize      int64  `json:"payloadSize" yaml:"payloadSize"`
	Checksum         string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	StoredPayloadRef string `json:"storedPayloadRef,omitempty" yaml:"storedPayloadRef,omitempty"`
}
