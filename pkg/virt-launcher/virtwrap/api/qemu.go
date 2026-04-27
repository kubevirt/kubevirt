package api

type QueryBlockJobsResult struct {
	Return []BlockJobStatus `json:"return"`
	ID     string           `json:"id"`
}

type BlockJobStatus struct {
	AutoFinalize   bool   `json:"auto-finalize"`
	IOStatus       string `json:"io-status"`
	Device         string `json:"device"`
	AutoDismiss    bool   `json:"auto-dismiss"`
	Busy           bool   `json:"busy"`
	Len            int64  `json:"len"`
	Offset         int64  `json:"offset"`
	Status         string `json:"status"`
	Paused         bool   `json:"paused"`
	Speed          int64  `json:"speed"`
	Ready          bool   `json:"ready"`
	Type           string `json:"type"`
	ActivelySynced bool   `json:"actively-synced"`
	Error          string `json:"error"`
}

type QueryJobsResult struct {
	Return []JobStatus `json:"return"`
	ID     string      `json:"id"`
}
type JobStatus struct {
	CurrentProgress int64  `json:"current-progress"`
	Status          string `json:"status"`
	TotalProgress   int64  `json:"total-progress"`
	Type            string `json:"type"`
	ID              string `json:"id"`
	Error           string `json:"error"`
}
