package notifier

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isClientClosed check whether the error contains any code which indicates client is offline.
// These codes includes: codes.Unavailable
func isClientClosed(err error) bool {
	return status.Convert(err).Code() == codes.Unavailable
}
