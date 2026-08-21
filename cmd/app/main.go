package main

import (
	"log"

	"ups-monitor/internal/apcupsd"
	"ups-monitor/internal/config"
	"ups-monitor/internal/gui"
	"ups-monitor/internal/monitor"
	"ups-monitor/internal/tray"
)

func main() {
	cfg, err := config.LoadDefault()
	if err != nil {
		log.Fatal(err)
	}

	pollInterval, err := cfg.PollDuration()
	if err != nil {
		log.Fatal(err)
	}

	onBatteryDelay, err := cfg.OnBatteryDuration()
	if err != nil {
		log.Fatal(err)
	}

	gracePeriod, err := cfg.GraceDuration()
	if err != nil {
		log.Fatal(err)
	}

	client := apcupsd.New(cfg.Server)

	m := monitor.New(
		client,
		pollInterval,
		onBatteryDelay,
		gracePeriod,
		cfg.Shutdown.Enabled,
	)

	g := gui.New(m, cfg)

	t := tray.New(
		m,
		g.Show,
		g.ShowAbout,
		func() {
			g.Quit()
		},
	)

	t.Start()
	g.Run()
}
