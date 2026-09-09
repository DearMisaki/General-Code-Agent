package teams

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync/atomic"
	"testing"
)

const benchmarkMailboxTimestamp = "2026-09-09T00:00:00.000000000Z"

func BenchmarkFileMailBoxSend(b *testing.B) {
	mb := NewFileMailBox(filepath.Join(b.TempDir(), "inboxes"))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := mb.Send("agent-a", FileMailMessage{
			ID:        fmt.Sprintf("msg-%d", i),
			From:      "lead",
			Text:      "benchmark message",
			Timestamp: benchmarkMailboxTimestamp,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileMailBoxSendParallel(b *testing.B) {
	mb := NewFileMailBox(filepath.Join(b.TempDir(), "inboxes"))
	var ids atomic.Int64
	var lockErrors atomic.Int64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := ids.Add(1)
			err := mb.Send("agent-a", FileMailMessage{ID: fmt.Sprintf("msg-%d", id), From: "lead", Text: "x", Timestamp: benchmarkMailboxTimestamp})
			if errors.Is(err, fs.ErrExist) {
				lockErrors.Add(1)
				continue
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.ReportMetric(float64(lockErrors.Load())/float64(b.N), "lock_errors/op")
}

func BenchmarkFileMailBoxReadUnread(b *testing.B) {
	for _, count := range []int{10, 1000, 10000} {
		b.Run(fmt.Sprintf("messages_%d", count), func(b *testing.B) {
			mb := NewFileMailBox(filepath.Join(b.TempDir(), "inboxes"))
			if err := seedMailboxBenchmarkMessages(mb, count); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := mb.ReadUnread("agent-a"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFileMailBoxMarkAllRead(b *testing.B) {
	for _, count := range []int{10, 1000, 10000} {
		b.Run(fmt.Sprintf("messages_%d", count), func(b *testing.B) {
			mb := NewFileMailBox(filepath.Join(b.TempDir(), "inboxes"))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				if err := seedMailboxBenchmarkMessages(mb, count); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				if err := mb.MarkAllRead("agent-a"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func seedMailboxBenchmarkMessages(mb *FileMailBox, count int) error {
	messages := make([]FileMailMessage, count)
	for i := range messages {
		messages[i] = FileMailMessage{ID: fmt.Sprintf("seed-%d", i), From: "lead", Text: "x", Timestamp: benchmarkMailboxTimestamp}
	}
	return mb.writeInbox("agent-a", messages)
}
