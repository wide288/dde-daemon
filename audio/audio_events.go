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

package audio

import "pkg.deepin.io/lib/pulse"
import "time"

func (a *Audio) handleEvent() {
	for {
		select {
		case event := <-a.eventChan:
			switch event.Facility {
			case pulse.FacilityServer:
				a.handleServerEvent(event.Type)
				a.saveConfig()
			case pulse.FacilityCard:
				a.handleCardEvent(event.Type, event.Index)
				a.saveConfig()
			case pulse.FacilitySink:
				a.handleSinkEvent(event.Type, event.Index)
				a.saveConfig()
			case pulse.FacilitySource:
				a.handleSourceEvent(event.Type, event.Index)
				a.saveConfig()
			case pulse.FacilitySinkInput:
				a.handleSinkInputEvent(event.Type, event.Index)
			}

		case <-a.quit:
			logger.Debug("handleEvent return")
			return
		}
	}
}

func (a *Audio) initCtxChan() {
	a.core.AddEventChan(a.eventChan)
	a.core.AddStateChan(a.stateChan)
}

func (a *Audio) handleStateChanged() {
	for {
		select {
		case state := <-a.stateChan:
			switch state {
			case pulse.ContextStateFailed:
				logger.Warning("Pulse context connection failed, try again")
				ctx := pulse.GetContextForced()
				if ctx == nil {
					logger.Warning("failed to connect pulseaudio server")
					break
				}

				a.destroyCtxRelated()
				a.PropsMu.Lock()
				a.core = ctx
				a.PropsMu.Unlock()
				a.update()
				a.initCtxChan()
			}

		case <-a.quit:
			logger.Debug("handleStateChanged return")
			return
		}
	}
}

func (a *Audio) handleCardEvent(eType int, idx uint32) {
	switch eType {
	case pulse.EventTypeNew:
		logger.Debugf("[Event] card #%d added", idx)
		card, err := a.core.GetCard(idx)
		if nil != err {
			logger.Warning("get card info failed: ", err)
			return
		}
		infos, added := a.cards.add(newCardInfo(card))
		if added {
			a.PropsMu.Lock()
			a.setPropCards(infos.string())
			a.PropsMu.Unlock()
			a.cards = infos
		}
		// fix change profile not work
		time.AfterFunc(time.Millisecond*500, func() {
			selectNewCardProfile(card)
			logger.Debug("After select profile:", card.ActiveProfile.Name)

			if !autoSwitchPort {
				return
			}
			port := hasPortAvailable(card.Ports, pulse.DirectionSink, true)
			if port.Name != "" {
				logger.Debug("New card, found available sink port:", port)
				a.handlePortChanged(card.Index, pulse.CardPortInfo{}, port)
				time.Sleep(time.Millisecond * 300)
			}
			port = hasPortAvailable(card.Ports, pulse.DirectionSource, true)
			if port.Name != "" {
				logger.Debug("New card, found available source port:", port)
				a.handlePortChanged(card.Index, pulse.CardPortInfo{}, port)
				time.Sleep(time.Millisecond * 300)
			}
		})
	case pulse.EventTypeRemove:
		logger.Debugf("[Event] card #%d removed", idx)
		infos, deleted := a.cards.delete(idx)
		if deleted {
			a.PropsMu.Lock()
			a.setPropCards(infos.string())
			a.PropsMu.Unlock()
			a.cards = infos
		}
	case pulse.EventTypeChange:
		logger.Debugf("[Event] card #%d changed", idx)
		card, err := a.core.GetCard(idx)
		if nil != err {
			logger.Warning("get card info failed: ", err)
			return
		}
		info, _ := a.cards.get(idx)
		if info != nil {
			oldPorts := info.Ports
			info.update(card)
			a.PropsMu.Lock()
			a.setPropCards(a.cards.string())
			a.PropsMu.Unlock()

			if !autoSwitchPort {
				return
			}
			old, port := hasPortChanged(oldPorts, info.Ports)
			if port.Name == "" {
				logger.Debugf("No available port found, old: %#v, new: %#v",
					oldPorts, info.Ports)
				return
			}
			a.handlePortChanged(info.Id, old, port)
		}
	}
}
func (a *Audio) handlePortChanged(cardId uint32, old, port pulse.CardPortInfo) {
	logger.Debugf("Will switch to port: %#v", port)
	var err error
	if port.Available == pulse.AvailableTypeYes {
		// switch to port
		err = a.SetPort(cardId, port.Name, int32(port.Direction))
	} else if old.Available == pulse.AvailableTypeYes &&
		port.Available == pulse.AvailableTypeNo {
		// switch from port
		id, p := a.cards.getAvailablePort(port.Direction)
		if p.Name == "" {
			logger.Warningf("Not found available port: %#v", a.cards)
			return
		}
		logger.Debugf("Will switch from port: %#v, switch to: %#v", port, p)
		err = a.SetPort(id, p.Name, int32(p.Direction))
	}
	if err != nil {
		logger.Warning("Failed to set port:", err)
	}
}
func (a *Audio) handleSinkEvent(eType int, idx uint32) {
	switch eType {
	case pulse.EventTypeNew:
		logger.Debugf("[Event] sink #%d added", idx)
		sinfo, _ := a.core.GetServer()
		if sinfo != nil {
			a.updateDefaultSink(sinfo.DefaultSinkName, false)
		}
	case pulse.EventTypeRemove:
		logger.Debugf("[Event] sink #%d removed", idx)
		sinfo, _ := a.core.GetServer()
		if sinfo != nil {
			a.updateDefaultSink(sinfo.DefaultSinkName, false)
		}
	case pulse.EventTypeChange:
		logger.Debugf("[Event] sink #%d changed", idx)
		if a.defaultSink != nil && a.defaultSink.index == idx {
			info, err := a.core.GetSink(idx)
			if err != nil {
				logger.Warning(err)
				return
			}
			a.defaultSink.core = info
			a.defaultSink.update()
		}
	default:
		logger.Debugf("[Event] sink #%d unknown type %d", eType, idx)
		return
	}
	if a.defaultSink != nil {
		a.moveSinkInputsToSink(a.defaultSink.index)
	}
}

func (a *Audio) handleSinkInputEvent(eType int, idx uint32) {
	switch eType {
	case pulse.EventTypeNew:
		a.addSinkInput(idx)
	case pulse.EventTypeRemove:
		a.removeSinkInput(idx)

	case pulse.EventTypeChange:
		for _, s := range a.sinkInputs {
			if s.index == idx {
				info, err := a.core.GetSinkInput(idx)
				if err != nil {
					logger.Warning(err)
					break
				}

				s.core = info
				s.update()
				break
			}
		}
	}
}

func (a *Audio) handleSourceEvent(eType int, idx uint32) {
	switch eType {
	case pulse.EventTypeNew:
		logger.Debugf("[Event] source #%d added", idx)
		sinfo, _ := a.core.GetServer()
		if sinfo != nil {
			a.updateDefaultSource(sinfo.DefaultSourceName, false)
		}
	case pulse.EventTypeRemove:
		logger.Debugf("[Event] source #%d removed", idx)
		sinfo, _ := a.core.GetServer()
		if sinfo != nil {
			a.updateDefaultSource(sinfo.DefaultSourceName, false)
		}
	case pulse.EventTypeChange:
		logger.Debugf("[Event] source #%d changed", idx)
		if a.defaultSource != nil && a.defaultSource.index == idx {
			info, err := a.core.GetSource(idx)
			if err != nil {
				logger.Warning(err)
				return
			}
			a.defaultSource.core = info
			a.defaultSource.update()
		}
	default:
		logger.Debugf("[Event] source #%d unknown type %d", idx, eType)
		return
	}
}

func (a *Audio) handleServerEvent(eType int) {
	sinfo, err := a.core.GetServer()
	if err != nil {
		logger.Error(err)
		return
	}

	logger.Debug("[Event] server changed:", sinfo.DefaultSinkName, sinfo.DefaultSourceName)
	a.updateDefaultSink(sinfo.DefaultSinkName, true)
	a.updateDefaultSource(sinfo.DefaultSourceName, true)
}
