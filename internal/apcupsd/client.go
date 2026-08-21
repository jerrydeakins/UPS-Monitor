package apcupsd

import (
	"time"

	"github.com/acaylor/apcupsd"
)

type Status struct {
	Model          string
	State          string
	BatteryPercent float64
	TimeLeft       time.Duration
	LoadPercent    float64
}

type Client struct {
	address string
}

func New(address string) *Client {
	return &Client{
		address: address,
	}
}

func (c *Client) Status() (Status, error) {
	client, err := apcupsd.Dial("tcp", c.address)
	if err != nil {
		return Status{}, err
	}

	defer client.Close()

	s, err := client.Status()
	if err != nil {
		return Status{}, err
	}

	return Status{
		Model:          s.Model,
		State:          s.Status,
		BatteryPercent: s.BatteryChargePercent,
		TimeLeft:       s.TimeLeft,
		LoadPercent:    s.LoadPercent,
	}, nil
}
