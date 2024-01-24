package crontab

import (
	"math"
	"sync"
	"time"
)

type ctDescUnit struct {
	begin int
	end   int
	step  int // 0: only begin value

}

type CTDesc struct {
	year   []*ctDescUnit
	month  []*ctDescUnit
	day    []*ctDescUnit
	week   []*ctDescUnit
	hour   []*ctDescUnit
	minute []*ctDescUnit
	second []*ctDescUnit

	layoutRegexpLock sync.Mutex
}

func (ss *CTDesc) init() {
	if ss.year == nil {
		ss.year = []*ctDescUnit{{begin: 0, end: math.MaxInt32, step: 1}}
	}
	if ss.month == nil {
		ss.month = []*ctDescUnit{{begin: 1, end: 12, step: 1}}
	}
	if ss.day == nil {
		ss.day = []*ctDescUnit{{begin: 1, end: 31, step: 1}}
	}
	if ss.week == nil {
		ss.week = []*ctDescUnit{{begin: 0, end: 6, step: 1}}
	}
	if ss.hour == nil {
		ss.hour = []*ctDescUnit{{begin: 0, end: 24, step: 1}}
	}
	if ss.minute == nil {
		ss.minute = []*ctDescUnit{{begin: 0, end: 59, step: 1}}
	}
	if ss.second == nil {
		ss.second = []*ctDescUnit{{begin: 0, end: 59, step: 1}}
	}
}

func (ss *CTDesc) normUnit(params []*ctDescUnit, val int) (carry bool, newVal int) {
	for _, v := range params {
		b, e, s := v.begin, v.end, v.step
		if v.step == 0 {
			if val <= b {
				return false, b
			}
			continue
		}

		if val > e {
			continue
		}

		if val <= b {
			return false, b
		}

		nv := val
		if (val-b)%s > 0 {
			nv = ((val-b)/s+1)*s + b
		}
		if nv > e {
			continue
		}

		return false, nv
	}
	return true, params[0].begin
}

func (ss *CTDesc) Norm(t time.Time) time.Time {
	year, _, day := t.Date()
	month := int(t.Month())
	hour, minute, second := t.Clock()

	var c bool
	var ov int

	ov = year
YEAR:
	c, year = ss.normUnit(ss.year, year)
	if c { // year cannot have carry
		return time.Time{}
	}
	if ov != year {
		month, day, hour, minute, second = 1, 1, 0, 0, 0
	}

	ov = month
MONTH:
	c, month = ss.normUnit(ss.month, month)
	if c {
		ov = year
		year++
		goto YEAR
	} else if ov != month {
		day, hour, minute, second = 1, 0, 0, 0
	}

	ov = day
DAY:
	c, day = ss.normUnit(ss.day, day)
	if c || time.Date(year, time.Month(month), day, 0, 0, 0, 0, t.Location()).Day() != day {
		ov = month
		month++
		goto MONTH
	} else if ov != day {
		hour, minute, second = 0, 0, 0
	}

	week := int(time.Date(year, time.Month(month), day, 0, 0, 0, 0, t.Location()).Weekday())
	ov = week
	c, week = ss.normUnit(ss.week, week)
	if c {
		nt := time.Date(year, time.Month(month), day+week-ov+7, 0, 0, 0, 0, t.Location())
		day = nt.Day()
		if int(nt.Month()) != month {
			ov = month
			month++
			goto MONTH
		}
		hour, minute, second = 0, 0, 0
	} else if ov != week {
		nt := time.Date(year, time.Month(month), day+week-ov, 0, 0, 0, 0, t.Location())
		day = nt.Day()
		if int(nt.Month()) != month {
			ov = month
			month++
			goto MONTH
		}
		hour, minute, second = 0, 0, 0
	}

	ov = hour
HOUR:
	c, hour = ss.normUnit(ss.hour, hour)
	if c {
		ov = day
		day++
		goto DAY
	} else if ov != hour {
		minute, second = 0, 0
	}

	ov = minute
MINUTE:
	c, minute = ss.normUnit(ss.minute, minute)
	if c {
		ov = hour
		hour++
		goto HOUR
	} else if ov != minute {
		second = 0
	}

	ov = second

	c, second = ss.normUnit(ss.second, second)
	if c {
		ov = minute
		minute++
		goto MINUTE
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, t.Location())
}
