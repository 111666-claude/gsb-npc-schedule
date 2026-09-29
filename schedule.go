// Package npc 是 NPC 日程：按时间段切换状态，只用 Go 标准库。
// 时间段以服务器时区的当天分钟数（0..1439）解释，支持跨零点区间，
// 区间左闭右开；校验通过的日程任意时刻状态唯一。
package npc

import (
	"errors"
	"sort"
	"time"
)

// ErrOverlap 表示时间段重叠。
var ErrOverlap = errors.New("npc: 时间段重叠")

// errRange 表示时间段越界。
var errRange = errors.New("npc: 时间越界")

// minutesPerDay 是一天的分钟数。
const minutesPerDay = 1440

// Slot 是一个时间段（以服务器时区当天分钟数表示，Start 0..1439，End 0..1440）。
// Start > End 表示跨零点区间，如 22:00-06:00。
type Slot struct {
	Name  string
	Start int
	End   int
}

// contains 判断 minute 是否落在左闭右开区间内，支持跨零点：
// Start > End 时按「分钟落在 [Start,1440) 或 [0,End)」判断。
func (s Slot) contains(minute int) bool {
	switch {
	case s.Start < s.End:
		return minute >= s.Start && minute < s.End
	case s.Start > s.End:
		return minute >= s.Start || minute < s.End
	default:
		return false
	}
}

// segments 把区间归一化为 [0,1440) 内的一到两个左闭右开段。
func (s Slot) segments() [][2]int {
	switch {
	case s.Start < s.End:
		return [][2]int{{s.Start, s.End}}
	case s.Start > s.End:
		return [][2]int{{s.Start, minutesPerDay}, {0, s.End}}
	default:
		return nil
	}
}

// StateAt 返回 minute 时刻（服务器时区当天分钟数）的状态。
// 命中多个区间时返回日程中排最前的那个；校验通过的日程不会命中多个。
func StateAt(slots []Slot, minute int) string {
	for _, slot := range slots {
		if slot.contains(minute) {
			return slot.Name
		}
	}
	return "none"
}

// Validate 校验日程：范围越界报错，任意两个时间段重叠返回 ErrOverlap。
func Validate(slots []Slot) error {
	segs := make([][2]int, 0, len(slots))
	for _, slot := range slots {
		if slot.Start < 0 || slot.Start > 1439 || slot.End < 0 || slot.End > minutesPerDay {
			return errRange
		}
		segs = append(segs, slot.segments()...)
	}
	sort.Slice(segs, func(i, j int) bool {
		if segs[i][0] != segs[j][0] {
			return segs[i][0] < segs[j][0]
		}
		return segs[i][1] < segs[j][1]
	})
	for i := 1; i < len(segs); i++ {
		if segs[i][0] < segs[i-1][1] {
			return ErrOverlap
		}
	}
	return nil
}

// ServerMinuteOfDay 返回 t 在服务器时区 loc 下的当天分钟数。
// 客户端本地时间不参与；loc 为 nil 时使用服务器本地时区。
func ServerMinuteOfDay(t time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.Local
	}
	serverTime := t.In(loc)
	return serverTime.Hour()*60 + serverTime.Minute()
}

// StateAtTime 把 t 换算到服务器时区（跨零点以服务器日期为准），返回对应状态。
func StateAtTime(slots []Slot, t time.Time, loc *time.Location) string {
	return StateAt(slots, ServerMinuteOfDay(t, loc))
}
