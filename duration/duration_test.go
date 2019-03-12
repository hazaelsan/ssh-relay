package duration

import (
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	dpb "github.com/golang/protobuf/ptypes/duration"
)

func TestFromProto(t *testing.T) {
	testdata := []struct {
		d    *dpb.Duration
		want time.Duration
		ok   bool
	}{
		{
			d:    &dpb.Duration{Seconds: 10},
			want: 10 * time.Second,
			ok:   true,
		},
		// nil duration.
		{
			ok: true,
		},
		// Negative durations are an error.
		{
			d: &dpb.Duration{Seconds: -10},
		},
	}
	for _, tt := range testdata {
		var got time.Duration
		if err := FromProto(&got, tt.d); err != nil {
			if tt.ok {
				t.Errorf("FromProto(%v) error = %v", pretty.Sprint(tt.d), err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("FromProto(%v) error = nil", pretty.Sprint(tt.d))
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("FromProto(%v) diff (-got +want):\n%v", pretty.Sprint(tt.d), diff)
		}
	}
}
