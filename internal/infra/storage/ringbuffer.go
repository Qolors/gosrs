package storage

import (
	"fmt"
	"sync"

	"github.com/qolors/gosrs/internal/core/model"
)

// An In-Memory Storage Option
type RingBuffer struct {
	buffer []model.StampedData
	size   int
	next   int
	full   bool
	mu     sync.Mutex
}

// Creates a new buffer with the provided capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]model.StampedData, capacity),
		size:   capacity,
	}
}

func (rb *RingBuffer) GetAll() []model.StampedData {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		return rb.buffer[:rb.next]
	}

	result := make([]model.StampedData, rb.size)
	copy(result, rb.buffer[rb.next:])
	copy(result[rb.size-rb.next:], rb.buffer[:rb.next])
	return result
}

func (rb *RingBuffer) Add(item model.StampedData) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	change := false

	if rb.full || rb.next > 0 {
		prevIndex := (rb.next - 1 + rb.size) % rb.size
		previous := rb.buffer[prevIndex]

		if haschanges(previous, item) {
			fmt.Println("Change in xp detected")
			change = true
		}
	}

	rb.buffer[rb.next] = item
	rb.next = (rb.next + 1) % rb.size

	if rb.next == 0 {
		rb.full = true
	}

	fmt.Println("Item Added")

	return change
}

func haschanges(i1 model.StampedData, i2 model.StampedData) bool {

	var change bool

	if i1.Skills[0].XP != i2.Skills[0].XP {
		change = true
	}

	return change

}
