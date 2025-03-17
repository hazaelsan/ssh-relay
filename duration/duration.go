// Package duration provides functionality around time.Duration.
package duration

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

// FromProto is a helper to set a *time.Duration from a durationpb.Duration.
// Returns an error if the duration is negative, but NOT if the duration is nil.
func FromProto(dst *time.Duration, src *durationpb.Duration) error {
	if src == nil {
		return nil
	}
	*dst = src.AsDuration()
	if *dst < 0 {
		return fmt.Errorf("negative duration: %v", *dst)
	}
	return nil
}
