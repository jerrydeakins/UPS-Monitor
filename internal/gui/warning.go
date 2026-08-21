package gui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"ups-monitor/assets"
	"ups-monitor/internal/monitor"
)

type WarningWindow struct {
	gui *GUI

	window *fyne.Window

	titleLabel     *widget.Label
	modelLabel     *widget.Label
	countdownLabel *widget.Label
	infoLabel      *widget.Label

	cancelButton *widget.Button
	pauseButton  *widget.Button

	tickerStop chan struct{}

	mu      sync.Mutex
	visible bool
}

func newWarningWindow(g *GUI) *WarningWindow {
	w := &WarningWindow{
		gui: g,
	}

	title := widget.NewLabel("⚡ Питание от батареи")
	title.TextStyle = fyne.TextStyle{
		Bold: true,
	}

	w.titleLabel = title

	w.modelLabel = widget.NewLabel("UPS")

	w.countdownLabel = widget.NewLabel("01:00")
	w.countdownLabel.TextStyle = fyne.TextStyle{
		Bold: true,
	}

	w.infoLabel = widget.NewLabel(
		"Компьютер будет выключен автоматически.",
	)

	w.cancelButton = widget.NewButton(
		"Отменить выключение",
		func() {
			g.monitor.CancelShutdown()
		},
	)

	w.pauseButton = widget.NewButton(
		"Пауза мониторинга",
		func() {
			g.monitor.Pause()
		},
	)

	content := container.NewVBox(
		w.titleLabel,
		w.modelLabel,

		widget.NewSeparator(),

		widget.NewLabel("Выключение через:"),
		w.countdownLabel,

		w.infoLabel,

		container.NewGridWithColumns(
			2,
			w.cancelButton,
			w.pauseButton,
		),
	)

	window := g.app.NewWindow("UPS Monitor — предупреждение")
	warningIcon := fyne.NewStaticResource(
		"onbatt.png",
		assets.TrayOnBattPNG,
	)

	window.SetIcon(warningIcon)

	window.SetContent(
		container.NewPadded(content),
	)

	window.Resize(fyne.NewSize(430, 260))
	window.SetFixedSize(true)
	window.CenterOnScreen()

	// Закрытие крестиком не должно просто скрывать
	// предупреждение и оставлять пользователя без информации.
	window.SetCloseIntercept(func() {
		g.monitor.CancelShutdown()
	})

	w.window = &window

	return w
}

func (w *WarningWindow) Show(state monitor.State) {
	w.mu.Lock()

	if w.visible {
		w.mu.Unlock()
		w.Update(state)
		return
	}

	w.visible = true
	w.tickerStop = make(chan struct{})

	w.mu.Unlock()

	w.Update(state)

	fyne.Do(func() {
		(*w.window).Show()
		(*w.window).RequestFocus()
	})

	go w.runCountdown()
}

func (w *WarningWindow) Hide() {
	w.mu.Lock()

	if !w.visible {
		w.mu.Unlock()
		return
	}

	w.visible = false

	if w.tickerStop != nil {
		close(w.tickerStop)
		w.tickerStop = nil
	}

	w.mu.Unlock()

	fyne.Do(func() {
		(*w.window).Hide()
	})
}

func (w *WarningWindow) Update(state monitor.State) {
	if state.Shutdown != monitor.ShutdownPending ||
		state.ShutdownAt == nil {
		return
	}

	model := "UPS"

	if state.Status != nil {
		model = state.Status.Model
	}

	remaining := time.Until(*state.ShutdownAt)

	if remaining < 0 {
		remaining = 0
	}

	countdown := formatCountdown(remaining)

	fyne.Do(func() {
		w.modelLabel.SetText(model)
		w.countdownLabel.SetText(countdown)
		w.infoLabel.SetText(
			"Компьютер будет выключен автоматически.",
		)
	})
}

func formatCountdown(d time.Duration) string {
	totalSeconds := int(d / time.Second)

	if totalSeconds < 0 {
		totalSeconds = 0
	}

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf(
		"%02d:%02d",
		minutes,
		seconds,
	)
}

func (w *WarningWindow) runCountdown() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			state := w.gui.monitor.State()

			if state.Shutdown != monitor.ShutdownPending {
				return
			}

			w.Update(state)

		case <-w.tickerStop:
			return
		}
	}
}
