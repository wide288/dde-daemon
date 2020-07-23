/*
 * Copyright (C) 2014 ~ 2018 Deepin Technology Co., Ltd.
 *
 * Author:     jouyouyun <jouyouwen717@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package bluetooth

import (
	"fmt"
	"os"

	bluez "github.com/linuxdeepin/go-dbus-factory/org.bluez"
	dbus "pkg.deepin.io/lib/dbus1"
	"pkg.deepin.io/lib/dbusutil"
	"pkg.deepin.io/lib/dbusutil/proxy"
)

type adapter struct {
	core    *bluez.HCI
	address string

	Path                dbus.ObjectPath
	Name                string
	Alias               string
	Powered             bool
	Discovering         bool
	Discoverable        bool
	DiscoverableTimeout uint32
}

func newAdapter(systemSigLoop *dbusutil.SignalLoop, apath dbus.ObjectPath) (a *adapter) {
	a = &adapter{Path: apath}
	systemConn := systemSigLoop.Conn()
	a.core, _ = bluez.NewHCI(systemConn, apath)
	a.core.InitSignalExt(systemSigLoop, true)
	a.connectProperties()
	a.address, _ = a.core.Address().Get(0)

	// fix alias
	alias, _ := a.core.Alias().Get(0)
	if alias == "first-boot-hostname" {
		hostname, err := os.Hostname()
		if err == nil {
			if hostname != "first-boot-hostname" {
				// reset alias
				err = a.core.Alias().Set(0, "")
				if err != nil {
					logger.Warning(err)
				}
			}
		} else {
			logger.Warning("failed to get hostname:", err)
		}
	}

	a.Alias, _ = a.core.Alias().Get(0)
	a.Name, _ = a.core.Name().Get(0)
	a.Powered, _ = a.core.Powered().Get(0)
	a.Discovering, _ = a.core.Discovering().Get(0)
	a.Discoverable, _ = a.core.Discoverable().Get(0)
	a.DiscoverableTimeout, _ = a.core.DiscoverableTimeout().Get(0)
	return
}

func (a *adapter) destroy() {
	a.core.RemoveHandler(proxy.RemoveAllHandlers)
}

func (a *adapter) String() string {
	return fmt.Sprintf("adapter %s [%s]", a.Alias, a.address)
}

func (a *adapter) notifyAdapterAdded() {
	logger.Debug("AdapterAdded", a)
	globalBluetooth.service.Emit(globalBluetooth, "AdapterAdded", marshalJSON(a))
	globalBluetooth.updateState()
}

func (a *adapter) notifyAdapterRemoved() {
	logger.Debug("AdapterRemoved", a)
	globalBluetooth.service.Emit(globalBluetooth, "AdapterRemoved", marshalJSON(a))
	globalBluetooth.updateState()
}

func (a *adapter) notifyPropertiesChanged() {
	globalBluetooth.service.Emit(globalBluetooth, "AdapterPropertiesChanged", marshalJSON(a))
	globalBluetooth.updateState()
}

func (a *adapter) connectProperties() {
	a.core.Name().ConnectChanged(func(hasValue bool, value string) {
		if !hasValue {
			return
		}
		a.Name = value
		logger.Debugf("%s Name: %v", a, value)
		a.notifyPropertiesChanged()
	})

	a.core.Alias().ConnectChanged(func(hasValue bool, value string) {
		if !hasValue {
			return
		}
		a.Alias = value
		logger.Debugf("%s Alias: %v", a, value)
		a.notifyPropertiesChanged()
	})
	a.core.Powered().ConnectChanged(func(hasValue bool, value bool) {
		if !hasValue {
			return
		}
		// 如果上层还是关闭状态，就不能让适配器启动
		PoweredCfg := globalBluetooth.config.getAdapterConfigPowered(a.address)
		if PoweredCfg != value && !PoweredCfg {
			// 上层未开启蓝牙，需要把后端适配器给power off
			err := a.core.Powered().Set(0, PoweredCfg)
			if err != nil {
				logger.Warningf("failed to set %s powered: %v,config is %v", a, err, PoweredCfg)
				return
			}
			return
		}
		a.Powered = value
		logger.Debugf("%s Powered: %v", a, value)
		a.notifyPropertiesChanged()
	})
	a.core.Discovering().ConnectChanged(func(hasValue bool, value bool) {
		if !hasValue {
			return
		}
		a.Discovering = value
		logger.Debugf("%s Discovering: %v", a, value)
		a.notifyPropertiesChanged()
	})
	a.core.Discoverable().ConnectChanged(func(hasValue bool, value bool) {
		if !hasValue {
			return
		}
		a.Discoverable = value
		logger.Debugf("%s Discoverable: %v", a, value)
		a.notifyPropertiesChanged()
	})
	a.core.DiscoverableTimeout().ConnectChanged(func(hasValue bool, value uint32) {
		if !hasValue {
			return
		}
		a.DiscoverableTimeout = value
		logger.Debugf("%s DiscoverableTimeout: %v", a, value)
		a.notifyPropertiesChanged()
	})
}
