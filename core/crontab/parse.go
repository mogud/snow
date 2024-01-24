package crontab

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	layoutWildcard            = `^\*$|^\?$`
	layoutValue               = `^(%value%)$`
	layoutRange               = `^(%value%)-(%value%)$`
	layoutWildcardAndInterval = `^\*/(\d+)$`
	layoutValueAndInterval    = `^(%value%)/(\d+)$`
	layoutRangeAndInterval    = `^(%value%)-(%value%)/(\d+)$`
	fieldFinder               = regexp.MustCompile(`\S+`)
	entryFinder               = regexp.MustCompile(`[^,]+`)
	layoutRegexp              = make(map[string]*regexp.Regexp)
	layoutRegexpLock          sync.Mutex
)

type descriptorName string

const (
	_descriptorSecond descriptorName = "second"
	_minuteDescriptor descriptorName = "minute"
	_hourDescriptor   descriptorName = "hour"
	_domDescriptor    descriptorName = "day-of-month"
	_monthDescriptor  descriptorName = "month"
	_dowDescriptor    descriptorName = "day-of-week"
	_yearDescriptor   descriptorName = "year"
)

type fieldDescriptor struct {
	name         descriptorName
	min          int
	max          int
	valuePattern string
}

var (
	secondDescriptor = fieldDescriptor{
		name:         _descriptorSecond,
		min:          0,
		max:          59,
		valuePattern: `0?[0-9]|[1-5][0-9]`,
	}
	minuteDescriptor = fieldDescriptor{
		name:         _minuteDescriptor,
		min:          0,
		max:          59,
		valuePattern: `0?[0-9]|[1-5][0-9]`,
	}
	hourDescriptor = fieldDescriptor{
		name:         _hourDescriptor,
		min:          0,
		max:          23,
		valuePattern: `0?[0-9]|1[0-9]|2[0-3]`,
	}
	domDescriptor = fieldDescriptor{
		name:         _domDescriptor,
		min:          1,
		max:          31,
		valuePattern: `0?[1-9]|[12][0-9]|3[01]`,
	}
	monthDescriptor = fieldDescriptor{
		name:         _monthDescriptor,
		min:          1,
		max:          12,
		valuePattern: `0?[1-9]|1[012]|jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec|january|february|march|april|march|april|june|july|august|september|october|november|december`,
	}
	dowDescriptor = fieldDescriptor{
		name:         _dowDescriptor,
		min:          0,
		max:          6,
		valuePattern: `0?[0-7]|sun|mon|tue|wed|thu|fri|sat|sunday|monday|tuesday|wednesday|thursday|friday|saturday`,
	}
	yearDescriptor = fieldDescriptor{
		name:         _yearDescriptor,
		min:          1970,
		max:          math.MaxInt32,
		valuePattern: `19[789][0-9]`,
	}

	cronNormalizer = strings.NewReplacer(
		"@yearly", "0 0 0 1 1 * *",
		"@annually", "0 0 0 1 1 * *",
		"@monthly", "0 0 0 1 * * *",
		"@weekly", "0 0 0 * * 0 *",
		"@daily", "0 0 0 * * * *",
		"@hourly", "0 0 * * * * *")
)

func main() {
	ct := MustParse("0 2 * * * * * *")
	ct.printRaw()
	now := time.Now()
	now = time.Date(now.Year(), time.Month(now.Month()), now.Day(), now.Hour(), 25, 4, 0, now.Location())
	ctN := ct.Norm(now)

	fmt.Println(now)
	fmt.Println(ctN)
	old := ctN
	for {
		now = time.Now()
		if ctN.Before(now) {
			nw := ct.Norm(now)
			if !old.Equal(nw) {
				old = ctN
				ctN = nw
				fmt.Println(old)
				fmt.Println(ctN)
			}
		}
	}

}

func MustParse(cronLine string) *CTDesc {
	expr, err := Parse(cronLine)
	if err != nil {
		panic(err)
	}
	return expr
}

func Parse(cronLine string) (*CTDesc, error) {
	desc := &CTDesc{}
	desc.init()

	cronLine = cronNormalizer.Replace(cronLine)
	// fmt.Println(cronLine)
	indices := fieldFinder.FindAllStringIndex(cronLine, -1)

	fieldCount := len(indices)
	if fieldCount < 5 {
		return nil, fmt.Errorf("missing field(s)")
	}

	if fieldCount > 7 {
		fieldCount = 7
	}

	var field = 0
	var err error

	str := ""
	// second field (optional)
	if fieldCount == 7 {
		str = cronLine[indices[field][0]:indices[field][1]]
		err = desc.genericFieldParse(str, secondDescriptor)
		// fmt.Print("second field:", str)
		if err != nil {
			return nil, err
		}
		field += 1
	}

	// minute field
	str = cronLine[indices[field][0]:indices[field][1]]
	err = desc.genericFieldParse(str, minuteDescriptor)
	// fmt.Print("  minute field:", str)
	if err != nil {
		return nil, err
	}
	field += 1

	// hour field
	str = cronLine[indices[field][0]:indices[field][1]]
	err = desc.genericFieldParse(str, hourDescriptor)
	// fmt.Print("   hour field:", str)
	if err != nil {
		return nil, err
	}
	field += 1

	// day of month field
	str = cronLine[indices[field][0]:indices[field][1]]
	err = desc.genericFieldParse(str, domDescriptor)
	//  fmt.Print("  day of month field:", str)
	if err != nil {
		return nil, err
	}
	field += 1

	// month field
	str = cronLine[indices[field][0]:indices[field][1]]
	err = desc.genericFieldParse(str, monthDescriptor)
	//  fmt.Print("  month field:", str)
	if err != nil {
		return nil, err
	}
	field += 1

	// day of week field
	str = cronLine[indices[field][0]:indices[field][1]]
	err = desc.genericFieldParse(str, dowDescriptor)
	//  fmt.Print("  day of week field:", str)
	if err != nil {
		return nil, err
	}
	field += 1

	// year field
	if field < fieldCount {
		str = cronLine[indices[field][0]:indices[field][1]]
		err = desc.genericFieldParse(str, yearDescriptor)
		// fmt.Println("  year field:", str)
		if err != nil {
			return nil, err
		}
	}
	// fmt.Println()
	return desc, nil
}

func (ss *CTDesc) genericFieldParse(s string, desc fieldDescriptor) error {
	// At least one entry must be present
	indices := entryFinder.FindAllStringIndex(s, -1)
	if len(indices) == 0 {
		return fmt.Errorf("%s field: missing directive", desc.name)
	}

	units := make([]*ctDescUnit, 0, len(indices))
	for i := range indices {

		unit := &ctDescUnit{
			step: 1,
		}
		snormal := strings.ToLower(s[indices[i][0]:indices[i][1]])
		// `*`
		if ss.makeLayoutRegexp(layoutWildcard, desc.valuePattern).MatchString(snormal) {
			unit.begin = desc.min
			unit.end = desc.max
			units = append(units, unit)
			continue
		}
		// `5`
		if ss.makeLayoutRegexp(layoutValue, desc.valuePattern).MatchString(snormal) {
			unit.begin = ss.atoi(snormal)
			unit.step = 0
			units = append(units, unit)
			continue
		}
		// `5-20`
		pairs := ss.makeLayoutRegexp(layoutRange, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			unit.begin = ss.atoi(snormal[pairs[2]:pairs[3]])
			unit.end = ss.atoi(snormal[pairs[4]:pairs[5]])
			units = append(units, unit)
			continue
		}
		// `*/2`
		pairs = ss.makeLayoutRegexp(layoutWildcardAndInterval, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			unit.begin = desc.min
			unit.end = desc.max
			unit.step = ss.atoi(snormal[pairs[2]:pairs[3]])
			if unit.step < 1 || unit.step > desc.max {
				return fmt.Errorf("invalid interval %s", snormal)
			}
			units = append(units, unit)
			continue
		}
		// `5/2`
		pairs = ss.makeLayoutRegexp(layoutValueAndInterval, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			unit.begin = ss.atoi(snormal[pairs[2]:pairs[3]])
			unit.end = desc.max

			unit.step = ss.atoi(snormal[pairs[4]:pairs[5]])
			if unit.step < 1 || unit.step > desc.max {
				return fmt.Errorf("invalid interval %s", snormal)
			}
			units = append(units, unit)
			continue
		}
		// `5-20/2`
		pairs = ss.makeLayoutRegexp(layoutRangeAndInterval, desc.valuePattern).FindStringSubmatchIndex(snormal)
		if len(pairs) > 0 {
			unit.begin = ss.atoi(snormal[pairs[2]:pairs[3]])
			unit.end = ss.atoi(snormal[pairs[4]:pairs[5]])
			unit.step = ss.atoi(snormal[pairs[6]:pairs[7]])
			if unit.step < 1 || unit.step > desc.max {
				return fmt.Errorf("invalid interval %s", snormal)
			}
			units = append(units, unit)
			continue
		}
	}

	switch desc.name {
	case _descriptorSecond:
		ss.second = units
		break
	case _minuteDescriptor:
		ss.minute = units
		break
	case _hourDescriptor:
		ss.hour = units
		break
	case _domDescriptor:
		ss.day = units
		break
	case _monthDescriptor:
		ss.month = units
		break
	case _dowDescriptor:
		ss.week = units
		break
	case _yearDescriptor:
		ss.year = units
		break
	}
	return nil
}

func (ss *CTDesc) makeLayoutRegexp(layout, value string) *regexp.Regexp {
	ss.layoutRegexpLock.Lock()
	defer ss.layoutRegexpLock.Unlock()

	layout = strings.Replace(layout, `%value%`, value, -1)
	re := layoutRegexp[layout]
	if re == nil {
		re = regexp.MustCompile(layout)
		layoutRegexp[layout] = re
	}
	return re
}

func (ss *CTDesc) atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (ss *CTDesc) printRaw() {
	// second field (optional)
	fmt.Print("second field:")
	for _, s := range ss.second {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Print("\t")
	// minute field

	fmt.Print("minute field:")
	for _, s := range ss.minute {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Print("\t")

	// hour field
	fmt.Print("hour field:")
	for _, s := range ss.hour {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Print("\t")

	// day of month field
	fmt.Print("day month field:")
	for _, s := range ss.day {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Print("\t")

	// month field

	fmt.Print("month field:")
	for _, s := range ss.month {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Print("\t")

	fmt.Print("day week field:")
	for _, s := range ss.week {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Print("\t")

	fmt.Print("year field:")
	for _, s := range ss.year {
		fmt.Print(fmt.Sprintf(" begin:%d end:%d step:%d  ", s.begin, s.end, s.step))
	}
	fmt.Println()
}
