package poller

import (
	"github.com/qolors/gosrs/internal/core"
	"github.com/qolors/gosrs/internal/services/courier"
)

type Poller struct {
	courier  courier.Courier
	client   core.Client
	notifier core.Notifier
	storage  core.Storage
}

func NewPoller(c core.Client, s core.Storage, n core.Notifier, cr courier.Courier) *Poller {
	return &Poller{client: c, storage: s, notifier: n}
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
