// Package npc 是 NPC 日程：按时间段切换状态。
// 缺陷：跨零点区间被判空、重叠区间不报错、时间段用本地时间而不是服务器时区。
package npc

import "errors"

// ErrOverlap 表示时间段重叠。
var ErrOverlap = errors.New("npc: 时间段重叠")

// Slot 是一个时间段（以当天分钟数表示，0..1439）。
type Slot struct {
	Name  string
	Start int
	End   int
}

// StateAt 返回 minute 时刻的状态。缺陷：只处理 Start <= minute < End，跨零点区间永远判空。
func StateAt(slots []Slot, minute int) string {
	for _, slot := range slots {
		if minute >= slot.Start && minute < slot.End {
			return slot.Name
		}
	}
	return "none"
}

// Validate 校验日程。缺陷：完全不检查重叠。
func Validate(slots []Slot) error {
	for _, slot := range slots {
		if slot.Start < 0 || slot.Start > 1439 || slot.End < 0 || slot.End > 1440 {
			return errors.New("npc: 时间越界")
		}
	}
	return nil
}
