// Package npc 是 NPC 日程：按时间段切换状态。
// 时间段按服务器时区解释（客户端本地时间不参与），跨零点以服务器日期为准。
package npc

import (
	"errors"
	"sort"
	"time"
)

// ErrOverlap 表示时间段重叠。
var ErrOverlap = errors.New("npc: 时间段重叠")

// Slot 是一个时间段（以当天分钟数表示，0..1439），左闭右开。
// Start > End 表示跨零点区间，如 22:00-06:00。
type Slot struct {
	Name  string
	Start int
	End   int
}

// contains 判断 minute 是否落在左闭右开区间内。
// 跨零点区间按「分钟落在 [Start, 1440) 或 [0, End)」判断。
func (s Slot) contains(minute int) bool {
	if s.Start <= s.End {
		return minute >= s.Start && minute < s.End
	}
	return minute >= s.Start || minute < s.End
}

// StateAt 返回 minute 时刻的状态。同一输入重复执行结果相同；
// 对通过 Validate 的日程，任意时刻状态唯一。
func StateAt(slots []Slot, minute int) string {
	for _, slot := range slots {
		if slot.contains(minute) {
			return slot.Name
		}
	}
	return "none"
}

// StateAtTime 按服务器时区解释 t（客户端本地时间不参与），
// 跨零点以服务器日期为准，返回该时刻的状态。
func StateAtTime(slots []Slot, t time.Time, serverLoc *time.Location) string {
	st := t.In(serverLoc)
	return StateAt(slots, st.Hour()*60+st.Minute())
}

// Validate 校验日程：范围必须在 [0,1439]/[0,1440] 内，且任意两个
// 时间段（跨零点归一后）不得重叠，否则返回 ErrOverlap。
func Validate(slots []Slot) error {
	for _, slot := range slots {
		if slot.Start < 0 || slot.Start > 1439 || slot.End < 0 || slot.End > 1440 {
			return errors.New("npc: 时间越界")
		}
	}
	// 跨零点归一：把每个时间段拆成 [0,1440) 内的一到两个半开区间。
	intervals := make([][2]int, 0, len(slots)*2)
	for _, slot := range slots {
		switch {
		case slot.Start < slot.End:
			intervals = append(intervals, [2]int{slot.Start, slot.End})
		case slot.Start > slot.End:
			intervals = append(intervals,
				[2]int{slot.Start, 1440},
				[2]int{0, slot.End})
		}
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i][0] != intervals[j][0] {
			return intervals[i][0] < intervals[j][0]
		}
		return intervals[i][1] < intervals[j][1]
	})
	for i := 1; i < len(intervals); i++ {
		if intervals[i][0] < intervals[i-1][1] {
			return ErrOverlap
		}
	}
	return nil
}
