package courier

import (
	"log"

	"github.com/qolors/gosrs/internal/core"
	"github.com/qolors/gosrs/internal/core/model"
)

type BaseCourier interface {
	Start()
}

type Courier struct {
	buffer   []model.StampedData
	notifier core.Notifier
	Pack     chan (model.StampedData)
	Receive  chan (model.StampedData)
	Send     chan ([]model.StampedData)
	Running  bool
}

func NewCourier(n core.Notifier) *Courier {
	return &Courier{
		Running:  false,
		Pack:     make(chan model.StampedData),
		Send:     make(chan []model.StampedData),
		Receive:  make(chan model.StampedData, 1),
		notifier: n,
	}
}

func (c *Courier) Start() {
	c.Running = true
	go func() {
		for {
			select {
			case stamped := <-c.Pack:
				c.buffer = append(c.buffer, stamped)
				log.Println("Packed minute frame data")
			case stamps := <-c.Send:
				log.Println("Session Over Building Report")
				if err := c.build(stamps); err != nil {
					log.Printf("Error building %s", err.Error())
				}
				return
			}
		}
	}()
}

func (c *Courier) build(day_data []model.StampedData) error {
	// Stop the courier while building the webhook.
	c.Running = false

	err := c.notifier.SendNotification(day_data, c.buffer)

	c.buffer = c.buffer[:0]

	return err
}
