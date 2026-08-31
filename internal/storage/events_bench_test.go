package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"logy/internal/events"
)

func BenchmarkInsertAndSearchEvents(b *testing.B) {
	db, err := Open(filepath.Join(b.TempDir(), "bench.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	evts := make([]events.Event, 100)
	for i := range evts {
		evts[i] = sampleEvent("evt-"+itoa(i), `C:\dev\app`, started, events.EventType("git.commit"))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.InsertEvents(ctx, evts); err != nil {
			b.Fatal(err)
		}
		if _, err := db.SearchEvents(ctx, events.EventFilter{}); err != nil {
			b.Fatal(err)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
