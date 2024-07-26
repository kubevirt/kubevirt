package common

type SyncError interface {
	error
	Reason() string
	// RequiresRequeue indicates if the sync error should trigger a requeue, or
	// if information should just be added to the sync condition and a regular controller
	// wakeup will resolve the situation.
	RequiresRequeue() bool
}
