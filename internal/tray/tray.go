package tray

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/systray"

	"ups-monitor/internal/monitor"
)

type Tray struct {
	monitor *monitor.Monitor

	showWindow func()
	showAbout  func()

	mu    sync.Mutex
	ready bool

	start func()
	end   func()

	showItem   *systray.MenuItem
	pauseItem  *systray.MenuItem
	resumeItem *systray.MenuItem
	cancelItem *systray.MenuItem
	quitItem   *systray.MenuItem
	aboutItem  *systray.MenuItem
}

func New(m *monitor.Monitor, showWindow func(), showAbout func(), quit func()) *Tray {
	t := &Tray{
		monitor:    m,
		showWindow: showWindow,
		showAbout:  showAbout,
	}

	start, end := systray.RunWithExternalLoop(
		func() {
			t.onReady(quit)
		},
		func() {
			// Tray завершён.
		},
	)

	t.start = start
	t.end = end

	m.SetStateCallback(t.updateState)

	return t
}

func (t *Tray) Start() {
	t.start()
}

func (t *Tray) Stop() {
	if t.end != nil {
		t.end()
	}
}

func (t *Tray) onReady(quit func()) {
	t.mu.Lock()
	t.ready = true
	t.mu.Unlock()

	systray.SetIcon(makeIcon(iconOnline))

	systray.SetTooltip("UPS Monitor")

	t.showItem = systray.AddMenuItem(
		"Открыть",
		"Открыть UPS Monitor",
	)

	t.pauseItem = systray.AddMenuItem(
		"Пауза мониторинга",
		"Приостановить автоматическую реакцию на UPS",
	)

	t.resumeItem = systray.AddMenuItem(
		"Возобновить мониторинг",
		"Возобновить мониторинг UPS",
	)

	t.cancelItem = systray.AddMenuItem(
		"Отменить выключение",
		"Отменить запланированное выключение",
	)

	systray.AddSeparator()

	t.aboutItem = systray.AddMenuItem(
		"О программе",
		"Информация о UPS Monitor",
	)

	systray.AddSeparator()

	t.quitItem = systray.AddMenuItem(
		"Выход",
		"Закрыть UPS Monitor",
	)

	go t.handleMenu(quit)

	// Начальное состояние.
	t.updateState(t.monitor.State())
}

func (t *Tray) handleMenu(quit func()) {
	for {
		select {
		case <-t.showItem.ClickedCh:
			if t.showWindow != nil {
				t.showWindow()
			}

		case <-t.pauseItem.ClickedCh:
			t.monitor.Pause()

		case <-t.resumeItem.ClickedCh:
			t.monitor.Resume()

		case <-t.cancelItem.ClickedCh:
			t.monitor.CancelShutdown()

		case <-t.aboutItem.ClickedCh:
			if t.showAbout != nil {
				t.showAbout()
			}

		case <-t.quitItem.ClickedCh:
			quit()
			return
		}
	}
}

func (t *Tray) updateState(state monitor.State) {
	t.mu.Lock()
	ready := t.ready
	t.mu.Unlock()

	if !ready {
		return
	}

	iconState := iconOnline

	if state.Monitoring == monitor.MonitoringPaused {
		iconState = iconPaused
	} else {
		switch state.UPS {
		case monitor.UPSOnline:
			iconState = iconOnline

		case monitor.UPSOnBatt:
			iconState = iconOnBatt

		case monitor.UPSUnknown:
			if state.Connection == monitor.ConnectionDisconnected {
				iconState = iconDisconnected
			}
		}
	}

	systray.SetIcon(makeIcon(iconState))

	statusText := "UPS Monitor"

	if state.Status != nil {
		statusText = state.Status.Model
	}

	switch state.UPS {

	case monitor.UPSOnline:
		tooltip := statusText

		if state.Status != nil {
			tooltip = fmt.Sprintf(
				"%s • Сеть • %.0f%% • %s",
				statusText,
				state.Status.BatteryPercent,
				state.Status.TimeLeft.Round(time.Second),
			)
		}

		systray.SetTooltip(tooltip)

	case monitor.UPSOnBatt:
		tooltip := statusText

		if state.Status != nil {
			tooltip = fmt.Sprintf(
				"%s • БАТАРЕЯ • %.0f%% • %s",
				statusText,
				state.Status.BatteryPercent,
				state.Status.TimeLeft.Round(time.Second),
			)
		}

		systray.SetTooltip(tooltip)

	case monitor.UPSUnknown:
		if state.Connection == monitor.ConnectionDisconnected {
			age := "нет данных"

			if !state.LastUpdate.IsZero() {
				age = time.Since(state.LastUpdate).
					Round(time.Second).
					String()
			}

			systray.SetTooltip(
				fmt.Sprintf(
					"UPS Monitor • НЕТ СВЯЗИ • последние данные %s назад",
					age,
				),
			)
		} else {
			systray.SetTooltip("UPS Monitor • состояние неизвестно")
		}
	}

	if state.Monitoring == monitor.MonitoringPaused {
		systray.SetTooltip(
			"UPS Monitor • МОНИТОРИНГ ПРИОСТАНОВЛЕН",
		)
	}

	t.updateMenu(state)
}

func (t *Tray) updateMenu(state monitor.State) {
	if t.pauseItem == nil {
		return
	}

	switch state.Monitoring {

	case monitor.MonitoringActive:
		t.pauseItem.Show()
		t.resumeItem.Hide()

	case monitor.MonitoringPaused:
		t.pauseItem.Hide()
		t.resumeItem.Show()
	}

	if state.Shutdown == monitor.ShutdownPending {
		t.cancelItem.Show()
	} else {
		t.cancelItem.Hide()
	}
}

func trayIcon() []byte {
	// Временно.
	// На следующем шаге сюда поставим нормальный .ico.
	return []byte{}
}
