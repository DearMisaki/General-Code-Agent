package teams

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkFileMailBoxSend(b *testing.B) {
	mb := NewFileMailBox(filepath.Join(b.TempDir(), "inboxes"))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := mb.Send("agent-a", FileMailMessage{
			ID:        fmt.Sprintf("msg-%d", i),
			From:      "lead",
			Text:      "benchmark message",
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileMailBoxSendParallel(b *testing.B) {
	mb := NewFileMailBox(filepath.Join(b.TempDir(), "inboxes"))
	var ids atomic.Int64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := ids.Add(1)
			if err := mb.Send(fmt.Sprintf("agent-%d", id), FileMailMessage{ID: fmt.Sprintf("msg-%d", id), From: "lead", Text: "x"}); err != nil {
				b.Fatal(err)
			}
		}
	})
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
			if err := seedMailboxBenchmarkMessages(mb, count); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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
		messages[i] = FileMailMessage{ID: fmt.Sprintf("seed-%d", i), From: "lead", Text: "x"}
	}
	return mb.writeInbox("agent-a", messages)
}
