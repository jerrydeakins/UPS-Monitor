package monitor

import (
	"log"
	"sync"
	"time"

	"ups-monitor/internal/apcupsd"
)

type ConnectionState string
type UPSState string
type MonitoringState string
type ShutdownState string

const (
	ConnectionConnected    ConnectionState = "CONNECTED"
	ConnectionDisconnected ConnectionState = "DISCONNECTED"

	UPSOnline  UPSState = "ONLINE"
	UPSOnBatt  UPSState = "ONBATT"
	UPSUnknown UPSState = "UNKNOWN"

	MonitoringActive MonitoringState = "ACTIVE"
	MonitoringPaused MonitoringState = "PAUSED"

	ShutdownNone      ShutdownState = "NONE"
	ShutdownPending   ShutdownState = "PENDING"
	ShutdownCancelled ShutdownState = "CANCELLED"
	ShutdownExecuted  ShutdownState = "EXECUTED"
)

const reconnectInterval = 5 * time.Second

type State struct {
	Connection ConnectionState
	UPS        UPSState
	Monitoring MonitoringState
	Shutdown   ShutdownState

	Status     *apcupsd.Status
	LastUpdate time.Time
	LastStatus *apcupsd.Status

	OnBatterySince *time.Time
	ShutdownAt     *time.Time
}

type Monitor struct {
	Client            *apcupsd.Client
	PollInterval      time.Duration
	ReconnectInterval time.Duration
	OnBatteryDelay    time.Duration
	GracePeriod       time.Duration
	ShutdownEnabled   bool

	mu    sync.RWMutex
	state State

	onStateChange []func(State)
	onShutdown    func()
}

func New(
	client *apcupsd.Client,
	pollInterval time.Duration,
	onBatteryDelay time.Duration,
	gracePeriod time.Duration,
	shutdownEnabled bool,
) *Monitor {
	return &Monitor{
		Client:            client,
		PollInterval:      pollInterval,
		ReconnectInterval: reconnectInterval,
		OnBatteryDelay:    onBatteryDelay,
		GracePeriod:       gracePeriod,
		ShutdownEnabled:   shutdownEnabled,

		state: State{
			Connection: ConnectionDisconnected,
			UPS:        UPSUnknown,
			Monitoring: MonitoringActive,
			Shutdown:   ShutdownNone,
		},
	}
}

func (m *Monitor) SetStateCallback(callback func(State)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if callback != nil {
		m.onStateChange = append(m.onStateChange, callback)
	}
}

func (m *Monitor) SetShutdownCallback(callback func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onShutdown = callback
}

func (m *Monitor) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.state
}

func (m *Monitor) Pause() {
	m.mu.Lock()

	if m.state.Monitoring == MonitoringPaused {
		m.mu.Unlock()
		return
	}

	m.state.Monitoring = MonitoringPaused
	m.state.Shutdown = ShutdownNone
	m.state.OnBatterySince = nil
	m.state.ShutdownAt = nil

	m.mu.Unlock()

	log.Println("Monitoring paused.")

	m.publishState()
}

func (m *Monitor) Resume() {
	m.mu.Lock()

	if m.state.Monitoring == MonitoringActive {
		m.mu.Unlock()
		return
	}

	m.state.Monitoring = MonitoringActive
	m.state.Shutdown = ShutdownNone
	m.state.OnBatterySince = nil
	m.state.ShutdownAt = nil
	m.state.UPS = UPSUnknown

	m.mu.Unlock()

	log.Println("Monitoring resumed.")

	m.publishState()
}

func (m *Monitor) CancelShutdown() {
	m.mu.Lock()

	if m.state.Shutdown != ShutdownPending {
		m.mu.Unlock()
		return
	}

	m.state.Shutdown = ShutdownCancelled
	m.state.ShutdownAt = nil
	m.state.OnBatterySince = nil

	m.mu.Unlock()

	log.Println("Shutdown cancelled by user.")

	m.publishState()
}

func (m *Monitor) Run() {
	for {
		m.poll()

		m.mu.RLock()
		disconnected := m.state.Connection == ConnectionDisconnected
		m.mu.RUnlock()

		interval := m.PollInterval

		if disconnected {
			interval = m.ReconnectInterval
		}

		time.Sleep(interval)
	}
}

func (m *Monitor) poll() {
	m.mu.RLock()
	paused := m.state.Monitoring == MonitoringPaused
	client := m.Client
	m.mu.RUnlock()

	if paused {
		return
	}

	status, err := client.Status()
	if err != nil {
		m.handleConnectionError(err)
		return
	}

	m.handleStatus(status)
}

func (m *Monitor) handleConnectionError(err error) {
	m.mu.Lock()

	if m.state.Connection != ConnectionDisconnected {
		log.Printf("APCUPSD connection lost: %v", err)
	} else {
		log.Printf("APCUPSD unavailable: %v", err)
	}

	m.state.Connection = ConnectionDisconnected
	m.state.UPS = UPSUnknown

	// ВАЖНО:
	// если ONBATT уже был обнаружен, локальный таймер продолжается.
	// Если ONBATT ещё не было, потеря связи сама по себе
	// не запускает shutdown.
	m.mu.Unlock()

	m.publishState()
}

func (m *Monitor) handleStatus(status apcupsd.Status) {
	now := time.Now()

	m.mu.Lock()

	m.state.Connection = ConnectionConnected
	m.state.Status = &status
	m.state.LastStatus = &status
	m.state.LastUpdate = now

	if status.State == "ONBATT" {
		m.state.UPS = UPSOnBatt

		if m.state.OnBatterySince == nil {
			t := now
			m.state.OnBatterySince = &t

			log.Printf(
				"UPS switched to battery. Timer started: %s",
				m.OnBatteryDelay,
			)
		}

		if m.ShutdownEnabled &&
			m.state.Shutdown != ShutdownPending &&
			now.Sub(*m.state.OnBatterySince) >= m.OnBatteryDelay {

			shutdownAt := now.Add(m.GracePeriod)

			m.state.Shutdown = ShutdownPending
			m.state.ShutdownAt = &shutdownAt

			callback := m.onShutdown

			m.mu.Unlock()

			log.Println("ONBATT delay expired. Shutdown pending.")

			m.publishState()

			if callback != nil {
				callback()
			}

			return
		}

	} else {
		m.state.UPS = UPSOnline

		if m.state.Shutdown == ShutdownPending {
			m.state.Shutdown = ShutdownCancelled
			m.state.ShutdownAt = nil

			log.Println(
				"UPS returned online. Shutdown automatically cancelled.",
			)
		}

		if m.state.OnBatterySince != nil {
			log.Printf(
				"ONBATT timer cancelled. APCUPSD state: %q",
				status.State,
			)

			m.state.OnBatterySince = nil
		}
	}

	m.mu.Unlock()

	m.publishState()
}

func (m *Monitor) publishState() {
	m.mu.RLock()

	state := m.state

	callbacks := append(
		[]func(State){},
		m.onStateChange...,
	)

	m.mu.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(state)
		}
	}
}

func (m *Monitor) ApplySettings(
	server string,
	onBatteryDelay time.Duration,
	gracePeriod time.Duration,
	shutdownEnabled bool,
) {
	m.mu.Lock()

	m.Client = apcupsd.New(server)
	m.OnBatteryDelay = onBatteryDelay
	m.GracePeriod = gracePeriod
	m.ShutdownEnabled = shutdownEnabled

	// Если автоматическое выключение отключили,
	// уже запланированное выключение отменяем.
	if !shutdownEnabled && m.state.Shutdown == ShutdownPending {
		m.state.Shutdown = ShutdownCancelled
		m.state.ShutdownAt = nil
	}

	m.mu.Unlock()

	log.Println("Monitor settings applied.")
	m.publishState()
}
