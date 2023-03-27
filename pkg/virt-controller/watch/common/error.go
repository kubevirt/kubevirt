package common

type SyncError interface {
	error
	Reason() string
	// RequiresRequeue indicates if the sync error should trigger a requeue, or
	// if information should just be added to the sync condition and a regular controller
	// wakeup will resolve the situation.
	RequiresRequeue() bool
}

type syncErrorImpl struct {
	err    error
	reason string
}

func NewSyncError(err error, reason string) *syncErrorImpl {
	return &syncErrorImpl{err, reason}
}

func (e *syncErrorImpl) Error() string {
	return e.err.Error()
}

func (e *syncErrorImpl) Reason() string {
	return e.reason
}

func (e *syncErrorImpl) RequiresRequeue() bool {
	return true
}
