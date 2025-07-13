package systemd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// DBusConnection manages the connection to systemd via D-Bus
type DBusConnection struct {
	conn     *dbus.Conn
	mu       sync.RWMutex
	closed   bool
	systemd  dbus.BusObject
	manager  dbus.BusObject
	lastPing time.Time
}

// ServiceState represents the current state of a systemd service
type ServiceState struct {
	Name            string
	LoadState       string // loaded, not-found, masked, error
	ActiveState     string // active, inactive, activating, deactivating, failed
	SubState        string // running, exited, failed, dead, etc.
	Description     string
	MainPID         int32
	MemoryCurrent   uint64
	CPUUsageNSec    uint64
	RestartCount    int
	LastStateChange time.Time
	Dependencies    []string
	Path            string
}

// StateChange represents a systemd service state change event
type StateChange struct {
	Timestamp   time.Time
	ServiceName string
	OldState    string
	NewState    string
	SubState    string
	Reason      string
	JobID       uint32
}

// NewDBusConnection creates a new D-Bus connection to systemd
func NewDBusConnection() (*DBusConnection, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}

	systemd := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	manager := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1/Manager")

	dbc := &DBusConnection{
		conn:     conn,
		systemd:  systemd,
		manager:  manager,
		lastPing: time.Now(),
	}

	// Test connection
	if err := dbc.ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping systemd: %w", err)
	}

	return dbc, nil
}

// Close closes the D-Bus connection
func (d *DBusConnection) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.closed = true
	return d.conn.Close()
}

// ping tests the connection to systemd
func (d *DBusConnection) ping() error {
	var version string
	err := d.systemd.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Manager", "Version").Store(&version)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	d.lastPing = time.Now()
	return nil
}

// ListUnits returns all systemd units
func (d *DBusConnection) ListUnits(ctx context.Context) ([]ServiceState, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil, fmt.Errorf("connection closed")
	}

	var units [][]interface{}
	call := d.manager.CallWithContext(ctx, "org.freedesktop.systemd1.Manager.ListUnits", 0)
	if err := call.Store(&units); err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}

	services := make([]ServiceState, 0, len(units))
	for _, unit := range units {
		// Only include service units
		name, ok := unit[0].(string)
		if !ok || !isServiceUnit(name) {
			continue
		}

		service := ServiceState{
			Name:            name,
			Description:     toString(unit[1]),
			LoadState:       toString(unit[2]),
			ActiveState:     toString(unit[3]),
			SubState:        toString(unit[4]),
			Path:            toString(unit[6]),
			LastStateChange: time.Now(), // Will be updated with actual time
		}

		// Get additional details
		if err := d.enrichServiceState(ctx, &service); err != nil {
			// Log error but continue
			continue
		}

		services = append(services, service)
	}

	return services, nil
}

// GetServiceState gets the state of a specific service
func (d *DBusConnection) GetServiceState(ctx context.Context, serviceName string) (*ServiceState, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil, fmt.Errorf("connection closed")
	}

	// Ensure service name has .service suffix
	if !hasServiceSuffix(serviceName) {
		serviceName += ".service"
	}

	// Get unit path
	var unitPath dbus.ObjectPath
	err := d.manager.CallWithContext(ctx, "org.freedesktop.systemd1.Manager.GetUnit", 0, serviceName).Store(&unitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get unit: %w", err)
	}

	unit := d.conn.Object("org.freedesktop.systemd1", unitPath)

	state := &ServiceState{
		Name: serviceName,
		Path: string(unitPath),
	}

	// Get properties
	props := map[string]interface{}{
		"LoadState":     &state.LoadState,
		"ActiveState":   &state.ActiveState,
		"SubState":      &state.SubState,
		"Description":   &state.Description,
		"MainPID":       &state.MainPID,
		"MemoryCurrent": &state.MemoryCurrent,
		"CPUUsageNSec":  &state.CPUUsageNSec,
	}

	for prop, dest := range props {
		if err := unit.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.systemd1.Unit", prop).Store(dest); err != nil {
			// Some properties might not be available
			continue
		}
	}

	// Get restart count
	if err := d.getRestartCount(ctx, unit, state); err != nil {
		// Non-critical, continue
	}

	// Get dependencies
	if err := d.getDependencies(ctx, unit, state); err != nil {
		// Non-critical, continue
	}

	// Get state change time
	if err := d.getStateChangeTime(ctx, unit, state); err != nil {
		// Non-critical, continue
	}

	return state, nil
}

// WatchStateChanges watches for systemd service state changes
func (d *DBusConnection) WatchStateChanges(ctx context.Context, changes chan<- StateChange) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return fmt.Errorf("connection closed")
	}

	// Subscribe to systemd signals
	if err := d.conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/systemd1"),
		dbus.WithMatchInterface("org.freedesktop.systemd1.Manager"),
		dbus.WithMatchMember("UnitNew"),
		dbus.WithMatchMember("UnitRemoved"),
		dbus.WithMatchMember("JobNew"),
		dbus.WithMatchMember("JobRemoved"),
	); err != nil {
		return fmt.Errorf("failed to add match signals: %w", err)
	}

	// Also watch for PropertiesChanged signals
	if err := d.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
	); err != nil {
		return fmt.Errorf("failed to add properties changed signal: %w", err)
	}

	signals := make(chan *dbus.Signal, 100)
	d.conn.Signal(signals)

	// State tracking for detecting actual changes
	serviceStates := make(map[string]string)

	go func() {
		defer close(changes)

		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-signals:
				if sig == nil {
					continue
				}

				change := d.processSignal(sig, serviceStates)
				if change != nil && isServiceUnit(change.ServiceName) {
					select {
					case changes <- *change:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return nil
}

// processSignal processes a D-Bus signal and returns a state change if relevant
func (d *DBusConnection) processSignal(sig *dbus.Signal, states map[string]string) *StateChange {
	switch sig.Name {
	case "org.freedesktop.systemd1.Manager.UnitNew":
		if len(sig.Body) >= 2 {
			unitName, _ := sig.Body[0].(string)
			if isServiceUnit(unitName) {
				return &StateChange{
					Timestamp:   time.Now(),
					ServiceName: unitName,
					OldState:    "non-existent",
					NewState:    "loaded",
					Reason:      "Unit created",
				}
			}
		}

	case "org.freedesktop.systemd1.Manager.UnitRemoved":
		if len(sig.Body) >= 2 {
			unitName, _ := sig.Body[0].(string)
			if isServiceUnit(unitName) {
				oldState := states[unitName]
				delete(states, unitName)
				return &StateChange{
					Timestamp:   time.Now(),
					ServiceName: unitName,
					OldState:    oldState,
					NewState:    "removed",
					Reason:      "Unit removed",
				}
			}
		}

	case "org.freedesktop.DBus.Properties.PropertiesChanged":
		if len(sig.Body) >= 2 {
			iface, _ := sig.Body[0].(string)
			if iface == "org.freedesktop.systemd1.Unit" {
				// Extract unit name from path
				unitName := extractUnitName(string(sig.Path))
				if unitName != "" && isServiceUnit(unitName) {
					// Get current state
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					state, err := d.GetServiceState(ctx, unitName)
					cancel()

					if err == nil {
						oldState := states[unitName]
						newState := state.ActiveState

						if oldState != "" && oldState != newState {
							states[unitName] = newState
							return &StateChange{
								Timestamp:   time.Now(),
								ServiceName: unitName,
								OldState:    oldState,
								NewState:    newState,
								SubState:    state.SubState,
								Reason:      "State transition",
							}
						}
						states[unitName] = newState
					}
				}
			}
		}
	}

	return nil
}

// enrichServiceState adds additional details to a service state
func (d *DBusConnection) enrichServiceState(ctx context.Context, state *ServiceState) error {
	if state.Path == "" {
		return nil
	}

	unit := d.conn.Object("org.freedesktop.systemd1", dbus.ObjectPath(state.Path))

	// Get restart count
	d.getRestartCount(ctx, unit, state)

	// Get dependencies
	d.getDependencies(ctx, unit, state)

	// Get state change time
	d.getStateChangeTime(ctx, unit, state)

	// Get resource usage
	d.getResourceUsage(ctx, unit, state)

	return nil
}

// getRestartCount gets the restart count for a service
func (d *DBusConnection) getRestartCount(ctx context.Context, unit dbus.BusObject, state *ServiceState) error {
	var nRestarts uint32
	err := unit.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Service", "NRestarts").Store(&nRestarts)
	if err == nil {
		state.RestartCount = int(nRestarts)
	}
	return err
}

// getDependencies gets the dependencies for a service
func (d *DBusConnection) getDependencies(ctx context.Context, unit dbus.BusObject, state *ServiceState) error {
	var deps []string
	err := unit.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Unit", "Requires").Store(&deps)
	if err == nil {
		state.Dependencies = deps
	}

	// Also get Wants dependencies
	var wants []string
	err = unit.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Unit", "Wants").Store(&wants)
	if err == nil {
		state.Dependencies = append(state.Dependencies, wants...)
	}

	return nil
}

// getStateChangeTime gets the last state change time for a service
func (d *DBusConnection) getStateChangeTime(ctx context.Context, unit dbus.BusObject, state *ServiceState) error {
	var stateChangeTimestamp uint64
	err := unit.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Unit", "StateChangeTimestamp").Store(&stateChangeTimestamp)
	if err == nil && stateChangeTimestamp > 0 {
		state.LastStateChange = time.Unix(0, int64(stateChangeTimestamp)*1000)
	}
	return err
}

// getResourceUsage gets resource usage for a service
func (d *DBusConnection) getResourceUsage(ctx context.Context, unit dbus.BusObject, state *ServiceState) error {
	// Memory usage
	var memoryCurrent uint64
	err := unit.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Service", "MemoryCurrent").Store(&memoryCurrent)
	if err == nil {
		state.MemoryCurrent = memoryCurrent
	}

	// CPU usage
	var cpuUsageNSec uint64
	err = unit.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Service", "CPUUsageNSec").Store(&cpuUsageNSec)
	if err == nil {
		state.CPUUsageNSec = cpuUsageNSec
	}

	return nil
}

// Helper functions

func isServiceUnit(name string) bool {
	return hasServiceSuffix(name)
}

func hasServiceSuffix(name string) bool {
	return len(name) > 8 && name[len(name)-8:] == ".service"
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func extractUnitName(path string) string {
	// Path format: /org/freedesktop/systemd1/unit/service_2ename_2eservice
	const prefix = "/org/freedesktop/systemd1/unit/"
	if len(path) > len(prefix) && path[:len(prefix)] == prefix {
		escaped := path[len(prefix):]
		// Unescape systemd encoding (e.g., _2e -> .)
		unitName := unescapeSystemdName(escaped)
		return unitName
	}
	return ""
}

func unescapeSystemdName(escaped string) string {
	// Simple unescape for common cases
	// In production, this should handle all systemd escape sequences
	result := ""
	i := 0
	for i < len(escaped) {
		if i+2 < len(escaped) && escaped[i] == '_' && escaped[i+1] == '2' {
			switch escaped[i+2] {
			case 'e':
				result += "."
				i += 3
			case 'd':
				result += "-"
				i += 3
			case 'f':
				result += "/"
				i += 3
			default:
				result += string(escaped[i])
				i++
			}
		} else {
			result += string(escaped[i])
			i++
		}
	}
	return result
}
