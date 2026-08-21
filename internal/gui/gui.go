package gui

import (
	"fmt"
	"image/color"
	"log"
	"os/exec"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"ups-monitor/assets"
	"ups-monitor/internal/config"
	"ups-monitor/internal/monitor"
	"ups-monitor/internal/version"
)

type GUI struct {
	app    fyne.App
	window fyne.Window

	// Main status
	statusTitle *canvas.Text
	upsLabel    *canvas.Text
	mainIcon    *canvas.Image

	// Metrics
	batteryValue *canvas.Text
	loadValue    *canvas.Text
	runtimeValue *canvas.Text

	// Connection
	connectionLabel *widget.Label
	connectionIcon  *canvas.Image

	// Buttons
	pauseButton    *widget.Button
	settingsButton *widget.Button

	warning *WarningWindow

	monitor *monitor.Monitor

	updateTickerStop chan struct{}

	mu              sync.Mutex
	shutdownRunning bool

	config config.Config
}

var (
	statusOnlineColor       = color.NRGBA{R: 56, G: 201, B: 74, A: 255}
	statusOnBattColor       = color.NRGBA{R: 229, G: 161, B: 26, A: 255}
	statusDisconnectedColor = color.NRGBA{R: 217, G: 74, B: 74, A: 255}
	statusPausedColor       = color.NRGBA{R: 122, G: 122, B: 122, A: 255}
)

func New(m *monitor.Monitor, cfg config.Config) *GUI {
	g := &GUI{
		monitor: m,
		config:  cfg,
	}

	g.app = app.NewWithID("ups-monitor")

	appIcon := fyne.NewStaticResource(
		"app.png",
		assets.AppIconPNG,
	)

	g.app.SetIcon(appIcon)

	g.window = g.app.NewWindow("UPS Monitor")
	g.window.SetIcon(appIcon)

	g.build()

	g.warning = newWarningWindow(g)

	m.SetStateCallback(g.updateState)

	m.SetShutdownCallback(func() {
		go g.waitForShutdown()
	})

	g.startUpdateTicker()

	return g
}

func (g *GUI) build() {
	g.mainIcon = canvas.NewImageFromResource(
		fyne.NewStaticResource(
			"main-online.png",
			assets.MainOnlinePNG,
		),
	)

	g.mainIcon.FillMode = canvas.ImageFillContain

	// Размер иконки в главном окне.
	g.mainIcon.SetMinSize(fyne.NewSize(110, 110))

	statusTitle := canvas.NewText(
		"ONLINE",
		theme.Color(theme.ColorNameForeground),
	)

	statusTitle.TextSize = 20
	statusTitle.TextStyle = fyne.TextStyle{
		Bold: true,
	}

	g.statusTitle = statusTitle

	g.upsLabel = canvas.NewText(
		"UPS",
		theme.Color(theme.ColorNameForeground),
	)
	g.upsLabel.TextSize = 16

	g.batteryValue = newMetricValue("—", metricBatteryColor)
	g.loadValue = newMetricValue("—", metricValueColor)
	g.runtimeValue = newMetricValue("—", metricValueColor)

	g.connectionLabel = widget.NewLabel("Питание от сети")

	g.pauseButton = widget.NewButton("Пауза", func() {
		state := g.monitor.State()

		if state.Monitoring == monitor.MonitoringPaused {
			g.monitor.Resume()
		} else {
			g.monitor.Pause()
		}
	})

	pauseButton := container.NewGridWrap(
		fyne.NewSize(120, 34),
		g.pauseButton,
	)

	g.settingsButton = widget.NewButton("Настройки", func() {
		g.showSettings()
	})

	settingsButton := container.NewGridWrap(
		fyne.NewSize(120, 34),
		g.settingsButton,
	)

	// Заголовок.
	headerGap := canvas.NewRectangle(color.Transparent)
	headerGap.SetMinSize(fyne.NewSize(14, 1))

	header := container.NewVBox(
		container.NewHBox(
			g.statusTitle,
			headerGap,
		),
		g.upsLabel,
	)

	// Метрики.
	battery := metricCard("БАТАРЕЯ", g.batteryValue)
	load := metricCard("НАГРУЗКА", g.loadValue)
	runtime := metricCard("РЕЗЕРВ", g.runtimeValue)

	// Вертикальные разделители между колонками.
	battery = container.NewBorder(
		nil,
		nil,
		nil,
		metricDivider(),
		battery,
	)

	load = container.NewBorder(
		nil,
		nil,
		nil,
		metricDivider(),
		load,
	)

	metricsTopGap := canvas.NewRectangle(color.Transparent)
	metricsTopGap.SetMinSize(fyne.NewSize(1, 10))

	metricsBottomGap := canvas.NewRectangle(color.Transparent)
	metricsBottomGap.SetMinSize(fyne.NewSize(1, 18))

	metricsGrid := container.NewGridWithColumns(
		3,
		battery,
		load,
		runtime,
	)

	metrics := container.NewVBox(
		metricsTopGap,
		metricsGrid,
		metricsBottomGap,
	)

	// Нижняя часть.
	g.connectionIcon = canvas.NewImageFromResource(
		connectionIconResource("#2DB641"),
	)

	g.connectionIcon.FillMode = canvas.ImageFillContain
	g.connectionIcon.SetMinSize(fyne.NewSize(24, 24))

	connection := container.NewHBox(
		g.connectionIcon,
		g.connectionLabel,
	)

	buttons := container.NewHBox(
		settingsButton,
		pauseButton,
	)

	bottom := container.NewBorder(
		nil,
		nil,
		nil,
		buttons,
		connection,
	)

	// Иконка слева, информация справа.
	left := container.NewGridWrap(
		fyne.NewSize(110, 110),
		g.mainIcon,
	)

	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(20, 1))

	topGap := canvas.NewRectangle(color.Transparent)
	topGap.SetMinSize(fyne.NewSize(1, 30))

	leftGap := canvas.NewRectangle(color.Transparent)
	leftGap.SetMinSize(fyne.NewSize(12, 1))

	leftSide := container.NewHBox(
		leftGap,
		container.NewVBox(
			topGap,
			container.NewHBox(
				left,
				gap,
			),
		),
	)

	// Верхняя часть: иконка + правая информационная часть.
	rightTop := container.NewVBox(
		header,
		widget.NewSeparator(),
		metrics,
	)

	topContent := container.NewBorder(
		nil,
		nil,
		leftSide,
		nil,
		container.NewPadded(rightTop),
	)

	// Нижняя часть на всю ширину окна.
	bottomTopGap := canvas.NewRectangle(color.Transparent)
	bottomTopGap.SetMinSize(fyne.NewSize(1, 8))

	bottomContent := container.NewVBox(
		bottomTopGap,
		bottom,
	)

	// Общий контент.
	// Этот Separator уже не ограничен правой колонкой
	// и проходит на всю ширину окна.
	upperContent := container.NewVBox(
		topContent,
		widget.NewSeparator(),
	)

	bottomGap := canvas.NewRectangle(color.Transparent)
	bottomGap.SetMinSize(fyne.NewSize(1, 20))

	bottomArea := container.NewVBox(
		bottomContent,
		bottomGap,
	)

	content := container.NewBorder(
		upperContent,
		bottomArea,
		nil,
		nil,
		nil,
	)

	g.window.SetContent(
		container.NewPadded(content),
	)

	g.window.Resize(fyne.NewSize(660, 290))
	g.window.SetFixedSize(true)
	g.window.CenterOnScreen()

	g.window.SetCloseIntercept(func() {
		g.window.Hide()
	})
}

var (
	metricBatteryColor = color.NRGBA{R: 56, G: 160, B: 70, A: 255}
	metricValueColor   = color.NRGBA{R: 35, G: 95, B: 190, A: 255}
)

func newMetricValue(value string, valueColor color.Color) *canvas.Text {
	valueText := canvas.NewText(value, valueColor)
	valueText.TextSize = 18
	valueText.TextStyle = fyne.TextStyle{
		Bold: true,
	}
	return valueText
}

func metricCard(title string, value *canvas.Text) fyne.CanvasObject {
	titleText := canvas.NewText(
		title,
		theme.Color(theme.ColorNameForeground),
	)
	titleText.TextSize = 15
	titleText.TextStyle = fyne.TextStyle{
		Bold: true,
	}
	titleText.Alignment = fyne.TextAlignCenter

	value.Alignment = fyne.TextAlignCenter

	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(1, 7))

	return container.NewVBox(
		titleText,
		gap,
		value,
	)
}

func metricDivider() fyne.CanvasObject {
	divider := canvas.NewRectangle(
		theme.Color(theme.ColorNameSeparator),
	)
	divider.SetMinSize(fyne.NewSize(1, 1))
	return divider
}

func (g *GUI) updateState(state monitor.State) {
	fyne.Do(func() {
		g.mu.Lock()
		defer g.mu.Unlock()

		g.updateStatus(state)
		g.updateMetrics(state)
		g.updateConnection(state)
		g.updateMonitoring(state)
		g.updateWarning(state)
	})
}

func (g *GUI) updateStatus(state monitor.State) {
	iconState := "online"

	switch state.UPS {
	case monitor.UPSOnBatt:
		iconState = "onbatt"

	case monitor.UPSUnknown:
		if state.Connection == monitor.ConnectionDisconnected {
			iconState = "disconnected"
		}
	}

	if state.Monitoring == monitor.MonitoringPaused {
		iconState = "paused"
	}

	switch iconState {
	case "online":
		g.mainIcon.Resource = fyne.NewStaticResource(
			"online-main.png",
			assets.MainOnlinePNG,
		)

	case "onbatt":
		g.mainIcon.Resource = fyne.NewStaticResource(
			"onbatt-main.png",
			assets.MainOnBattPNG,
		)

	case "disconnected":
		g.mainIcon.Resource = fyne.NewStaticResource(
			"disconnected-main.png",
			assets.MainDisconnectedPNG,
		)

	case "paused":
		g.mainIcon.Resource = fyne.NewStaticResource(
			"paused-main.png",
			assets.MainPausedPNG,
		)
	}

	g.mainIcon.Refresh()

	switch iconState {
	case "online":
		g.connectionIcon.Resource = connectionIconResource("#2DB641")

	case "onbatt":
		g.connectionIcon.Resource = connectionIconResource("#E5A11A")

	case "disconnected":
		g.connectionIcon.Resource = connectionIconResource("#D94A4A")

	case "paused":
		g.connectionIcon.Resource = connectionIconResource("#7A7A7A")
	}

	g.connectionIcon.Refresh()

	if state.Monitoring == monitor.MonitoringPaused {
		g.statusTitle.Text = "PAUSED"
		g.statusTitle.Color = statusPausedColor
	} else {
		switch state.UPS {
		case monitor.UPSOnline:
			g.statusTitle.Text = "ONLINE"
			g.statusTitle.Color = statusOnlineColor

		case monitor.UPSOnBatt:
			g.statusTitle.Text = "ON BATTERY"
			g.statusTitle.Color = statusOnBattColor

		case monitor.UPSUnknown:
			if state.Connection == monitor.ConnectionDisconnected {
				g.statusTitle.Text = "NO CONNECTION"
				g.statusTitle.Color = statusDisconnectedColor
			} else {
				g.statusTitle.Text = "UNKNOWN"
				g.statusTitle.Color = statusPausedColor
			}
		}
	}

	g.statusTitle.Refresh()

	if state.Status != nil {
		g.upsLabel.Text = state.Status.Model
		g.upsLabel.Refresh()
	}
}

func (g *GUI) updateMetrics(state monitor.State) {
	status := state.Status

	if status == nil {
		return
	}

	g.batteryValue.Text = fmt.Sprintf(
		"%.0f %%",
		status.BatteryPercent,
	)

	g.loadValue.Text = fmt.Sprintf(
		"%.0f %%",
		status.LoadPercent,
	)

	minutes := int(status.TimeLeft / time.Minute)

	if minutes >= 60 {
		hours := minutes / 60
		minutes %= 60

		if minutes == 0 {
			g.runtimeValue.Text = fmt.Sprintf(
				"%d ч",
				hours,
			)
		} else {
			g.runtimeValue.Text = fmt.Sprintf(
				"%d ч %02d мин",
				hours,
				minutes,
			)
		}
	} else if minutes > 0 {
		g.runtimeValue.Text = fmt.Sprintf(
			"%d мин",
			minutes,
		)
	} else {
		seconds := int(status.TimeLeft / time.Second)

		g.runtimeValue.Text = fmt.Sprintf(
			"%d с",
			seconds,
		)
	}

	g.batteryValue.Refresh()
	g.loadValue.Refresh()
	g.runtimeValue.Refresh()
}

func (g *GUI) updateConnection(state monitor.State) {
	switch state.Connection {
	case monitor.ConnectionConnected:
		if state.UPS == monitor.UPSOnBatt {
			g.connectionLabel.SetText("Питание от батареи")
		} else {
			g.connectionLabel.SetText("Питание от сети")
		}

	case monitor.ConnectionDisconnected:
		g.connectionLabel.SetText("⚠ Нет связи с APCUPSD")

	default:
		g.connectionLabel.SetText("Состояние неизвестно")
	}
}

func (g *GUI) updateMonitoring(state monitor.State) {
	switch state.Monitoring {

	case monitor.MonitoringActive:
		g.pauseButton.SetText("Пауза")

	case monitor.MonitoringPaused:
		g.pauseButton.SetText("Возобновить")
	}
}

func (g *GUI) Run() {
	go g.monitor.Run()
	g.app.Run()
}

func (g *GUI) Quit() {
	if g.updateTickerStop != nil {
		close(g.updateTickerStop)
		g.updateTickerStop = nil
	}

	fyne.DoAndWait(func() {
		g.app.Quit()
	})
}

func (g *GUI) Show() {
	fyne.DoAndWait(func() {
		g.window.Show()
		g.window.RequestFocus()
	})
}

func (g *GUI) ShowAbout() {
	appIcon := fyne.NewStaticResource(
		"app.png",
		assets.AppIconPNG,
	)

	content := container.NewVBox(
		widget.NewLabel("UPS Monitor"),
		widget.NewLabel("Мониторинг ИБП APCUPSD"),
		widget.NewLabel("Версия "+version.AppVersion),
	)

	window := g.app.NewWindow("О программе")
	window.SetIcon(appIcon)

	window.SetContent(
		container.NewPadded(content),
	)

	window.Resize(fyne.NewSize(320, 160))
	window.SetFixedSize(true)
	window.CenterOnScreen()

	fyne.DoAndWait(func() {
		window.Show()
		window.RequestFocus()
	})
}

func (g *GUI) updateWarning(state monitor.State) {
	if g.warning == nil {
		return
	}

	if state.Shutdown == monitor.ShutdownPending {
		g.warning.Show(state)
		return
	}

	g.warning.Hide()
}

func (g *GUI) waitForShutdown() {
	for {
		state := g.monitor.State()

		if state.Shutdown != monitor.ShutdownPending ||
			state.ShutdownAt == nil {
			return
		}

		if time.Until(*state.ShutdownAt) <= 0 {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	state := g.monitor.State()

	if state.Shutdown != monitor.ShutdownPending {
		return
	}

	if err := g.shutdownWindows(); err != nil {
		log.Printf("Shutdown failed: %v", err)
	}
}

func (g *GUI) shutdownWindows() error {
	g.mu.Lock()

	if g.shutdownRunning {
		g.mu.Unlock()

		log.Println("Shutdown already requested, ignoring duplicate request.")
		return nil
	}

	g.shutdownRunning = true
	g.mu.Unlock()

	log.Println("Executing Windows shutdown.")

	cmd := exec.Command(
		"shutdown.exe",
		"/s",
		"/t",
		"0",
	)

	if err := cmd.Run(); err != nil {
		g.mu.Lock()
		g.shutdownRunning = false
		g.mu.Unlock()

		return err
	}

	return nil
}

func (g *GUI) startUpdateTicker() {
	g.updateTickerStop = make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				state := g.monitor.State()

				fyne.Do(func() {
					g.mu.Lock()
					defer g.mu.Unlock()

					g.updateConnection(state)
				})

			case <-g.updateTickerStop:
				return
			}
		}
	}()
}

func connectionIconResource(color string) fyne.Resource {
	svg := fmt.Sprintf(`
<svg xmlns="http://www.w3.org/2000/svg"
     width="24" height="24"
     viewBox="0 0 24 24">

  <!-- Два контакта -->
  <path d="M8 3 V9"
        stroke="%s"
        stroke-width="2.5"
        stroke-linecap="round"/>

  <path d="M16 3 V9"
        stroke="%s"
        stroke-width="2.5"
        stroke-linecap="round"/>

  <!-- Корпус вилки -->
  <path d="M6 8
           H18
           V11
           C18 14.3 15.3 17 12 17
           C8.7 17 6 14.3 6 11
           Z"
        fill="%s"/>

  <!-- Кабель -->
  <path d="M12 17 V21"
        stroke="%s"
        stroke-width="2.5"
        stroke-linecap="round"/>

</svg>
`,
		color,
		color,
		color,
		color,
	)

	return fyne.NewStaticResource(
		"connection-"+color+".svg",
		[]byte(svg),
	)
}
