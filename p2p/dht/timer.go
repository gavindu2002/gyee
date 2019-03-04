/*
 *  Copyright (C) 2017 gyee authors
 *
 *  This file is part of the gyee library.
 *
 *  the gyee library is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  the gyee library is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with the gyee library.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package dht

import (
	"time"
	"fmt"
	"container/list"
	p2plog	"github.com/yeeco/gyee/p2p/logger"
)


//
// debug
//
type tmMgrLogger struct {
	debug__		bool
}

var tmLog = tmMgrLogger  {
	debug__:	false,
}

func (log tmMgrLogger)Debug(fmt string, args ... interface{}) {
	if log.debug__ {
		p2plog.Debug(fmt, args ...)
	}
}

const (
	oneTick			= time.Second					// unit tick to driver the timer manager
	OneTick			= oneTick
	xsecondBits		= 5
	xsecondCycle	= 1 << xsecondBits				// x-second cycle in tick
	xminuteBits		= 5
	xminuteCycle	= 1 << xminuteBits				// x-minute cycle in x-second
	xhourBits		= 5
	xhourCycle		= 1 << xhourBits				// x-hour cycle in x-minute
	xdayBits		= 5
	xdayCycle		= 1 << xdayBits					// x-day cycle in x-hour

	// min and max duration
	minDur			= oneTick
	maxDur			= (1 << (xsecondBits + xminuteBits + xhourBits + xdayBits)) * time.Second
)

type timerEno int

const (
	TmEnoNone			timerEno = iota		// none of errors
	TmEnoPara								// invalid parameters
	TmEnoDurTooBig							// duration too big
	TmEnoDurTooSmall						// duration too small
	TmEnoInternal							// internal errors
	TmEnoNotsupport							// not supported
	TmEnoBadTimer							// bad timer parameters
)

func (eno timerEno)Error() string {
	return fmt.Sprintf("%d", eno)
}

type TimerCallback = func(el *list.Element, data interface{})interface{}

type timer struct {
	s			int							// seconds remain
	m			int							// minutes remain
	h			int							// hours remain
	d			int							// day remain
	data		interface{}					// pointer passed to callback
	tcb			TimerCallback				// callback when timer expired
	li			*list.List					// list pointer
	el			*list.Element				// element pointer
	to			time.Time					// absolute time moment to be expired
	k			[]byte						// key attached to this timer
}

type TimerManager struct {
	sp			int							// second pointer
	mp			int							// minute pointer
	hp			int							// hour pointer
	dp			int							// day pointer
	sTmList		[xsecondCycle]*list.List	// second timer list
	mTmList		[xminuteCycle]*list.List	// minute timer list
	hTmList		[xhourCycle]*list.List		// hour timer list
	dTmList		[xdayCycle]*list.List		// day timer list
}

func NewTimerManager() *TimerManager {
	return &TimerManager{}
}

func (mgr *TimerManager)GetTimer(dur time.Duration, dat interface{}, tcb TimerCallback) (interface{}, error) {
	if dur < minDur {
		return nil, TmEnoDurTooSmall
	}
	if dur > maxDur {
		return nil, TmEnoDurTooBig
	}
	if tcb == nil {
		return nil, TmEnoPara
	}

	ss := int64(dur.Seconds() / oneTick.Seconds()) + int64(mgr.sp)
	xs := ss & (xsecondCycle - 1)
	ss = ss >> xsecondBits
	xm := ss & (xminuteCycle - 1)
	ss = ss >> xminuteBits
	xh := ss & (xhourCycle - 1)
	ss = ss >> xhourBits
	xd := ss & (xdayCycle - 1)

	tm := timer {
		s:		int(xs),
		m:		int(xm),
		h:		int(xh),
		d:		int(xd),
		data:	dat,
		tcb:	tcb,
		li:		nil,
		el:		nil,
		k:		nil,
	}

	return &tm, TmEnoNone
}

func (mgr *TimerManager)SetTimerHandler(ptm interface{}, tcb TimerCallback) error {
	tm := ptm.(*timer)
	tm.tcb = tcb
	return TmEnoNone
}

func (mgr *TimerManager)SetTimerData(ptm interface{}, data interface{}) error {
	tm := ptm.(*timer)
	tm.data = data
	return TmEnoNone
}

func (mgr *TimerManager)SetTimerKey(ptm interface{}, key []byte) error {
	tm := ptm.(*timer)
	tm.k = append(tm.k, key...)
	return TmEnoNone
}

func (mgr *TimerManager)StartTimer(ptm interface{}) error {
	tm, ok := ptm.(*timer)
	if tm == nil || !ok {
		return TmEnoPara
	}

	targetLi := (*list.List)(nil)

	if tm.s > 0 {
		sp := (mgr.sp + tm.s) & (xsecondCycle - 1)
		if mgr.sTmList[sp] == nil {
			mgr.sTmList[sp] = list.New()
		}
		targetLi = mgr.sTmList[sp]
	} else if tm.m > 0 {
		mp := (mgr.mp + tm.m) & (xminuteCycle- 1)
		if mgr.mTmList[mp] == nil {
			mgr.mTmList[mp] = list.New()
		}
		targetLi = mgr.mTmList[mp]
	} else if tm.h > 0 {
		hp := (mgr.hp + tm.h) & (xhourCycle - 1)
		if mgr.hTmList[hp] == nil {
			mgr.hTmList[hp] = list.New()
		}
		targetLi = mgr.hTmList[hp]
	} else if tm.d > 0 {
		dp := (mgr.dp + tm.d) & (xdayCycle - 1)
		if mgr.dTmList[dp] == nil {
			mgr.dTmList[dp] = list.New()
		}
		targetLi = mgr.dTmList[dp]
	} else {
		return TmEnoBadTimer
	}

	targetEl := targetLi.PushBack(tm)
	tm.li = targetLi
	tm.el = targetEl

	return TmEnoNone
}

func (mgr *TimerManager)KillTimer(ptm interface{}) error {

	if ptm == nil {
		return TmEnoPara
	}

	tm := ptm.(*timer)
	if tm.li == nil || tm.el == nil {
		return TmEnoPara
	}
	tm.li.Remove(tm.el)

	return TmEnoNone
}

func (mgr *TimerManager)TickProc() error {
	if mgr.sp = (mgr.sp + 1) & (xsecondCycle - 1); mgr.sp == 0 {
		if mgr.mp = (mgr.mp + 1) & (xminuteCycle - 1); mgr.mp == 0 {
			if mgr.hp = (mgr.hp + 1) & (xhourCycle - 1); mgr.hp == 0 {
				mgr.dp = (mgr.dp + 1) & (xdayCycle - 1)
			}
		}
	}
	serr := mgr.spHandler(mgr.sTmList[mgr.sp])
	merr := mgr.mpHandler(mgr.mTmList[mgr.mp])
	herr := mgr.hpHandler(mgr.hTmList[mgr.hp])
	derr := mgr.dpHandler(mgr.dTmList[mgr.dp])

	if serr != TmEnoNone || merr != TmEnoNone || herr != TmEnoNone || derr != TmEnoNone {
		tmLog.Debug("TickProc: serr: %s, merr: %s, herr: %s, derr: %s",
			serr.Error(), merr.Error(), herr.Error(), derr.Error())
		return TmEnoInternal
	}

	return TmEnoNone
}

func (mgr *TimerManager)spHandler(li *list.List) error {
	if li != nil {
		for {
			if el := li.Front(); el == nil {
				break
			} else {
				tm, _ := el.Value.(*timer)
				if tm.m > 0 {
					mp := (mgr.mp + tm.m) & (xminuteCycle - 1)
					if mgr.mTmList[mp] == nil {
						mgr.mTmList[mp] = list.New()
					}
					mgr.mTmList[mp].PushBack(tm)
				} else if tm.h > 0 {
					hp := (mgr.hp + tm.h) & (xhourCycle - 1)
					if mgr.hTmList[hp] == nil {
						mgr.hTmList[hp] = list.New()
					}
					mgr.hTmList[hp].PushBack(tm)
				} else if tm.d > 0 {
					dp := (mgr.dp + tm.h) & (xdayCycle - 1)
					if mgr.dTmList[dp] == nil {
						mgr.dTmList[dp] = list.New()
					}
					mgr.dTmList[dp].PushBack(tm)
				} else {
					tm.tcb(el, tm.data)
				}
				li.Remove(el)
			}
		}
	}
	return TmEnoNone
}

func (mgr *TimerManager)mpHandler(li *list.List) error {
	if li != nil {
		for {
			if el := li.Front(); el == nil {
				break
			} else {
				tm, _ := el.Value.(*timer)
				if tm.h > 0 {
					hp := (mgr.hp + tm.h) & (xhourCycle - 1)
					if mgr.hTmList[hp] == nil {
						mgr.hTmList[hp] = list.New()
					}
					mgr.hTmList[hp].PushBack(tm)
				} else if tm.d > 0 {
					dp := (mgr.dp + tm.h) & (xdayCycle - 1)
					if mgr.dTmList[dp] == nil {
						mgr.dTmList[dp] = list.New()
					}
					mgr.dTmList[dp].PushBack(tm)
				} else {
					tm.tcb(el, tm.data)
				}
				li.Remove(el)
			}
		}
	}
	return TmEnoNone
}

func (mgr *TimerManager)hpHandler(li *list.List) error {
	if li != nil {
		for {
			if el := li.Front(); el == nil {
				break
			} else {
				tm, _ := el.Value.(*timer)
				if tm.d > 0 {
					dp := (mgr.dp + tm.h) & (xdayCycle - 1)
					if mgr.dTmList[dp] == nil {
						mgr.dTmList[dp] = list.New()
					}
					mgr.dTmList[dp].PushBack(tm)
				} else {
					tm.tcb(el, tm.data)
				}
				li.Remove(el)
			}
		}
	}
	return TmEnoNone
}

func (mgr *TimerManager)dpHandler(li *list.List) error {
	if li != nil {
		for {
			if el := li.Front(); el == nil {
				break
			} else {
				tm, _ := el.Value.(*timer)
				tm.tcb(el, tm.data)
				li.Remove(el)
			}
		}
	}
	return TmEnoNone
}


