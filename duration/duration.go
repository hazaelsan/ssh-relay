// Package duration provides functionality around time.Duration.
package duration

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"

	dpb "github.com/golang/protobuf/ptypes/duration"
)

// FromProto is a helper to set a *time.Duration from a dpb.Duration.
// Returns an error if the duration is negative, but NOT if the duration is nil.
func FromProto(dst *time.Duration, src *dpb.Duration) error {
	if src == nil {
		return nil
	}
	var err error
	*dst, err = ptypes.Duration(src)
	if *dst < 0 {
		return fmt.Errorf("negative duration: %v", *dst)
	}
	return err
}
