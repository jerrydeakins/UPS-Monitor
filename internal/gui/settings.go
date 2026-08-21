package gui

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (g *GUI) showSettings() {
	cfg := g.config

	serverEntry := widget.NewEntry()
	serverEntry.SetText(cfg.Server)

	shutdownEnabled := widget.NewCheck(
		"Автоматическое выключение",
		nil,
	)
	shutdownEnabled.SetChecked(cfg.Shutdown.Enabled)

	onBatteryEntry := widget.NewEntry()
	onBatteryEntry.SetText(cfg.Shutdown.OnBatteryDelay)

	graceEntry := widget.NewEntry()
	graceEntry.SetText(cfg.Shutdown.GracePeriod)

	form := container.NewVBox(
		widget.NewLabel("Сервер APCUPSD:"),
		serverEntry,

		shutdownEnabled,

		widget.NewLabel("Выключать через:"),
		onBatteryEntry,

		widget.NewLabel("Предупреждать перед выключением:"),
		graceEntry,
	)

	cancelButton := widget.NewButton("Отмена", nil)
	saveButton := widget.NewButton("Сохранить", nil)

	buttons := container.NewHBox(
		cancelButton,
		saveButton,
	)

	content := container.NewBorder(
		nil,
		buttons,
		nil,
		nil,
		container.NewPadded(form),
	)

	window := g.app.NewWindow("UPS Monitor — Настройки")
	window.SetContent(content)
	window.Resize(fyne.NewSize(400, 300))
	window.SetFixedSize(true)
	window.CenterOnScreen()

	cancelButton.OnTapped = func() {
		window.Close()
	}

	saveButton.OnTapped = func() {
		server := strings.TrimSpace(serverEntry.Text)
		onBatteryDelay := strings.TrimSpace(onBatteryEntry.Text)
		gracePeriod := strings.TrimSpace(graceEntry.Text)

		// Проверяем сервер.
		if err := validateServer(server); err != nil {
			dialog.ShowError(
				fmt.Errorf("некорректный сервер APCUPSD: %v", err),
				window,
			)
			return
		}

		// Проверяем длительность до начала выключения.
		if err := validateDuration(onBatteryDelay); err != nil {
			dialog.ShowError(
				fmt.Errorf(
					"некорректное значение «Выключать через»: %v",
					err,
				),
				window,
			)
			return
		}

		// Проверяем время предупреждения.
		if err := validateDuration(gracePeriod); err != nil {
			dialog.ShowError(
				fmt.Errorf(
					"некорректное значение «Предупреждать перед выключением»: %v",
					err,
				),
				window,
			)
			return
		}

		// Создаём новую конфигурацию только после успешной проверки.
		cfg.Server = server
		cfg.Shutdown.Enabled = shutdownEnabled.Checked
		cfg.Shutdown.OnBatteryDelay = onBatteryDelay
		cfg.Shutdown.GracePeriod = gracePeriod

		// Сохраняем на диск.
		if err := cfg.SaveDefault(); err != nil {
			dialog.ShowError(
				fmt.Errorf("не удалось сохранить настройки: %v", err),
				window,
			)
			return
		}

		// Обновляем конфигурацию GUI.
		onBatteryDuration, err := cfg.OnBatteryDuration()
		if err != nil {
			dialog.ShowError(
				fmt.Errorf("некорректное значение «Выключать через»: %v", err),
				window,
			)
			return
		}

		graceDuration, err := cfg.GraceDuration()
		if err != nil {
			dialog.ShowError(
				fmt.Errorf("некорректное значение «Предупреждать перед выключением»: %v", err),
				window,
			)
			return
		}

		g.monitor.ApplySettings(
			cfg.Server,
			onBatteryDuration,
			graceDuration,
			cfg.Shutdown.Enabled,
		)

		g.config = cfg

		window.Close()

	}

	window.Show()
}

func validateDuration(value string) error {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	if duration <= 0 {
		return fmt.Errorf("значение должно быть больше нуля")
	}

	return nil
}

func validateServer(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("ожидается формат host:port")
	}

	if host == "" {
		return fmt.Errorf("не указан хост")
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("порт должен быть числом")
	}

	if portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("порт должен быть от 1 до 65535")
	}

	return nil
}
