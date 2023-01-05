package connection

import (
	"context"
	"time"

	"kubevirt.io/client-go/log"
)

type closer interface {
	Close() error
}

func CloseWithContext(ctx context.Context, closer closer, ifaceName string) {
	for {
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		<-ticker.C
		select {
		case <-ctx.Done():
			if closer == nil {
				log.Log.Warningf("closing %q DHCP server underlying connection not found, is it closed already?: %v", ifaceName, ctx.Err())
			}
			if err := closer.Close(); err != nil {
				log.Log.Warningf("failed to close %q DHCP server connection: %v", ifaceName, err)
			} else {
				log.Log.Infof("closing %q DHCP server underlying connection: %v", ifaceName, ctx.Err())
				return
			}
		default:
		}
	}
}
