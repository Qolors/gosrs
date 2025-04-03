package queue

import (
	"sync"
	"time"

	"github.com/qolors/gosrs/internal/osrsclient"
)

type StampedData struct {
	Timestamp  time.Time
	Activities []osrsclient.Activity
	Skills     []osrsclient.Skill
}

type RingBuffer struct {
	buffer []StampedData
	size   int
	next   int
	full   bool
	mu     sync.Mutex
}

// Creates a new buffer with the provided capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]StampedData, capacity),
		size:   capacity,
	}
}

func (rb *RingBuffer) Add(item StampedData) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.next] = item
	rb.next = (rb.next + 1) % rb.size

	if rb.next == 0 {
		rb.full = true
	}
}
