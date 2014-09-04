/**
 * Copyright (c) 2011 ~ 2014 Deepin, Inc.
 *               2013 ~ 2014 jouyouyun
 *
 * Author:      jouyouyun <jouyouwen717@gmail.com>
 * Maintainer:  jouyouyun <jouyouwen717@gmail.com>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, see <http://www.gnu.org/licenses/>.
 **/

package screenedge

import (
	libarea "dbus/com/deepin/api/xmousearea"
	libdsp "dbus/com/deepin/daemon/display"
	"dbus/com/deepin/dde/launcher"
	"pkg.linuxdeepin.com/lib/dbus"
	"pkg.linuxdeepin.com/lib/gio-2.0"
	"pkg.linuxdeepin.com/lib/log"
	"sync"
)

var (
	dspObj       *libdsp.Display
	areaObj      *libarea.XMouseArea
	launchObj    *launcher.Launcher
	logger       = log.NewLogger(ZONE_DEST)
	zoneSettings = gio.NewSettings("com.deepin.dde.zone")

	mutex         = new(sync.Mutex)
	edgeActionMap = make(map[string]string)
)

func (op *Manager) EnableZoneDetected(enable bool) {
	if enable {
		unregisterZoneArea()
		registerZoneArea()
	} else {
		unregisterZoneArea()
	}
}

func (op *Manager) SetTopLeft(value string) {
	mutex.Lock()
	defer mutex.Unlock()
	edgeActionMap[EDGE_TOPLEFT] = value
}

func (op *Manager) TopLeftAction() string {
	return zoneSettings.GetString("left-up")
}

func (op *Manager) SetBottomLeft(value string) {
	mutex.Lock()
	defer mutex.Unlock()
	edgeActionMap[EDGE_BOTTOMLEFT] = value
}

func (op *Manager) BottomLeftAction() string {
	return zoneSettings.GetString("left-down")
}

func (op *Manager) SetTopRight(value string) {
	mutex.Lock()
	defer mutex.Unlock()
	edgeActionMap[EDGE_TOPRIGHT] = value
}

func (op *Manager) TopRightAction() string {
	return zoneSettings.GetString("right-up")
}

func (op *Manager) SetBottomRight(value string) {
	mutex.Lock()
	defer mutex.Unlock()
	edgeActionMap[EDGE_BOTTOMRIGHT] = value
}

func (op *Manager) BottomRightAction() string {
	return zoneSettings.GetString("right-down")
}

func Stop() {
	logger.EndTracing()
}
func Start() {
	logger.BeginTracing()

	var err error
	dspObj, err = libdsp.NewDisplay("com.deepin.daemon.Display",
		"/com/deepin/daemon/Display")
	if err != nil {
		logger.Fatal("New Display Failed: ", err)
	}

	areaObj, err = libarea.NewXMouseArea("com.deepin.api.XMouseArea",
		"/com/deepin/api/XMouseArea")
	if err != nil {
		logger.Fatal("New XMouseArea Failed: ", err)
	}

	launchObj, err = launcher.NewLauncher("com.deepin.dde.launcher",
		"/com/deepin/dde/launcher")
	if err != nil {
		logger.Fatal("New DDE Launcher Failed: ", err)
	}

	m := newManager()
	err = dbus.InstallOnSession(m)
	if err != nil {
		logger.Fatal("Install Zone Session Failed: ", err)
	}

	m.SetTopLeft(m.TopLeftAction())
	m.SetBottomLeft(m.BottomLeftAction())
	m.SetTopRight(m.TopRightAction())
	m.SetBottomRight(m.BottomRightAction())
}