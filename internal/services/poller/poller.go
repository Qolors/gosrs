package poller

import (
	"github.com/qolors/gosrs/internal/core"
	"github.com/qolors/gosrs/internal/services/courier"
)

type Poller struct {
	courier *courier.Courier
	client  core.Client
	storage core.Storage
}

func NewPoller(c core.Client, s core.Storage, cr *courier.Courier) *Poller {
	return &Poller{client: c, storage: s, courier: cr}
}

func (p *Poller) Poll() error {

	sd, err := p.client.GetPlayerData()

	if err != nil {
		return err
	}

	changes := p.storage.Add(sd)

	if changes {
		if !p.courier.Running {
			p.courier.Start()
			p.courier.Pack <- sd
		} else {
			p.courier.Pack <- sd
		}
	} else {
		if p.courier.Running {
			p.courier.Send <- p.storage.GetAll()
		}
	}

	return err

}
