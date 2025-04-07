package queue

import (
	"fmt"
	"sync"
)

type RingBuffer struct {
	buffer []Interface{}
	size   int
	next   int
	full   bool
	mu     sync.Mutex
}

// Creates a new buffer with the provided capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]Interface{}, capacity),
		size:   capacity,
	}
}

func (rb *RingBuffer) GetAll() []Interface{} {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		return rb.buffer[:rb.next]
	}

	result := make([]Interface{}, rb.size)
	copy(result, rb.buffer[rb.next:])
	copy(result[rb.size-rb.next:], rb.buffer[:rb.next])
	return result
}

func (rb *RingBuffer) Add(item Interface{}) bool {
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

func haschanges(i1 Interface{}, i2 Interface{}) bool {
	var change bool
	//return i1.Skills[0].XP != i2.Skills[0].XP
	for i, skill := range i1.Skills {
		if skill.XP != i2.Skills[i].XP {
			change = true
			break
		}
	}

	return change

}
