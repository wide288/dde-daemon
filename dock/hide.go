package dock

import (
	"pkg.linuxdeepin.com/lib/dbus"
	"time"
)

const (
	HideStateShowing int32 = iota
	HideStateShown
	HideStateHidding
	HideStateHidden
)

var (
	HideStateMap map[int32]string = map[int32]string{
		HideStateShowing: "HideStateShowing",
		HideStateShown:   "HideStateShown",
		HideStateHidding: "HideStateHidding",
		HideStateHidden:  "HideStateHidden",
	}
)

type HideStateManager struct {
	state int32

	StateChanged func(int32)
}

func NewHideStateManager(mode string) *HideStateManager {
	h := &HideStateManager{}

	if mode == HideModeKeepHidden {
		h.state = HideStateHidden
	} else {
		h.state = HideStateShown
	}

	return h
}

func (e *HideStateManager) GetDBusInfo() dbus.DBusInfo {
	return dbus.DBusInfo{
		"com.deepin.daemon.Dock",
		"/dde/dock/HideStateManager",
		"dde.dock.HideStateManager",
	}
}

func (m *HideStateManager) SetState(s int32) int32 {
	if m.state == s {
		return s
	}

	logger.Debug("SetState m.state:", HideStateMap[m.state], "new state:", HideStateMap[s])
	m.state = s
	logger.Debug("SetState emit StateChanged signal", HideStateMap[m.state])
	m.StateChanged(s)

	return s
}

func (m *HideStateManager) UpdateState() {
	state := m.state
	switch setting.GetHideMode() {
	case HideModeKeepShowing:
		logger.Debug("KeepShowing Mode")
		state = HideStateShowing
	case HideModeAutoHide:
		logger.Debug("AutoHide Mode")
		state = HideStateShowing

		<-time.After(time.Millisecond * 100)
		if region.mouseInRegion() {
			logger.Debug("MouseInDockRegion")
			break
		}

		if hasMaximizeClient() {
			logger.Debug("has maximized client")
			state = HideStateHidding
		}
	case HideModeKeepHidden:
		logger.Debug("KeepHidden Mode")
		<-time.After(time.Millisecond * 100)
		if region.mouseInRegion() {
			logger.Debug("MouseInDockRegion")
			state = HideStateShowing
			break
		}

		state = HideStateHidding
	}

	if lastActive == DDELauncher {
		logger.Info(lastActive)
		state = HideStateHidding
	}

	m.SetState(state)
}
