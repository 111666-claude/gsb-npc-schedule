package npc

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestDaytimeSlot(t *testing.T) {
	slots := []Slot{{Name: "shop", Start: 9 * 60, End: 18 * 60}}
	if got := StateAt(slots, 10*60); got != "shop" {
		t.Fatalf("白天应该是 shop：%s", got)
	}
}

func TestNoSlot(t *testing.T) {
	if got := StateAt(nil, 0); got != "none" {
		t.Fatalf("没有日程应该是 none：%s", got)
	}
}

func TestValidateRange(t *testing.T) {
	if err := Validate([]Slot{{Name: "x", Start: -1, End: 10}}); err == nil {
		t.Fatal("越界应该报错")
	}
	if err := Validate([]Slot{{Name: "x", Start: 0, End: 100}}); err != nil {
		t.Fatalf("合法区间不该报错：%v", err)
	}
}

func TestBoundaryIsHalfOpen(t *testing.T) {
	slots := []Slot{{Name: "shop", Start: 540, End: 1080}}
	if got := StateAt(slots, 1080); got != "none" {
		t.Fatalf("结束时刻应该不算在内：%s", got)
	}
}

func TestOvernightSlot(t *testing.T) {
	slots := []Slot{{Name: "sleep", Start: 22 * 60, End: 6 * 60}}
	cases := []struct {
		minute int
		want   string
	}{
		{2 * 60, "sleep"},
		{22 * 60, "sleep"},    // 起点左闭
		{23*60 + 59, "sleep"}, // 23:59
		{0, "sleep"},          // 零点
		{5*60 + 59, "sleep"},  // 05:59
		{6 * 60, "none"},      // 终点右开
		{12 * 60, "none"},
		{21*60 + 59, "none"}, // 21:59
	}
	for _, c := range cases {
		if got := StateAt(slots, c.minute); got != c.want {
			t.Fatalf("minute=%d 应为 %s，实际 %s", c.minute, c.want, got)
		}
	}
}

func TestValidateOverlap(t *testing.T) {
	overlap := [][]Slot{
		{{Name: "a", Start: 9 * 60, End: 18 * 60}, {Name: "b", Start: 17 * 60, End: 20 * 60}},
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 5 * 60, End: 8 * 60}},   // 跨零点尾部重叠
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 21 * 60, End: 23 * 60}}, // 跨零点头部重叠
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 23 * 60, End: 24 * 60}}, // 跨零点与普通区间
		{{Name: "a", Start: 0, End: 1440}, {Name: "b", Start: 100, End: 200}},                 // 全天与任意区间
	}
	for _, slots := range overlap {
		if err := Validate(slots); err != ErrOverlap {
			t.Fatalf("%v 应报 ErrOverlap，实际 %v", slots, err)
		}
	}
	ok := [][]Slot{
		{{Name: "a", Start: 9 * 60, End: 17 * 60}, {Name: "b", Start: 17 * 60, End: 20 * 60}}, // 首尾相接不算重叠
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 6 * 60, End: 22 * 60}},  // 互补拼满全天
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 8 * 60, End: 18 * 60}},
	}
	for _, slots := range ok {
		if err := Validate(slots); err != nil {
			t.Fatalf("%v 不该报错，实际 %v", slots, err)
		}
	}
}

func TestBoundaryMidnight(t *testing.T) {
	slots := []Slot{{Name: "a", Start: 0, End: 60}, {Name: "b", Start: 1380, End: 1440}}
	if got := StateAt(slots, 0); got != "a" {
		t.Fatalf("0 点应命中 a：%s", got)
	}
	if got := StateAt(slots, 1439); got != "b" {
		t.Fatalf("1439 应命中 b：%s", got)
	}
	if got := StateAt(slots, 60); got != "none" {
		t.Fatalf("60 应为 none：%s", got)
	}
}

func TestServerTimezone(t *testing.T) {
	server := time.FixedZone("server", 8*3600) // 服务器 UTC+8
	// UTC 2026-09-30 18:00 即服务器时间 2026-10-01 02:00，跨零点以服务器日期为准。
	at := time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC)
	if got := ServerMinuteOfDay(at, server); got != 2*60 {
		t.Fatalf("服务器分钟数应为 120，实际 %d", got)
	}
	slots := []Slot{{Name: "sleep", Start: 22 * 60, End: 6 * 60}}
	if got := StateAtTime(slots, at, server); got != "sleep" {
		t.Fatalf("服务器 02:00 应为 sleep：%s", got)
	}
	// 同一时刻按 UTC 解释是 18:00，不应命中 sleep。
	if got := StateAtTime(slots, at, time.UTC); got != "none" {
		t.Fatalf("UTC 18:00 应为 none：%s", got)
	}
}

// refStateAt 是参考实现：跨零点归一为 [0,1440) 内的段后逐个判断。
func refStateAt(slots []Slot, minute int) string {
	for _, s := range slots {
		for _, seg := range refSegments(s) {
			if minute >= seg[0] && minute < seg[1] {
				return s.Name
			}
		}
	}
	return "none"
}

func refSegments(s Slot) [][2]int {
	if s.Start == s.End {
		return nil
	}
	if s.Start < s.End {
		return [][2]int{{s.Start, s.End}}
	}
	return [][2]int{{s.Start, 1440}, {0, s.End}}
}

// refValidate 是参考实现：归一化后两两比较查重叠。
func refValidate(slots []Slot) error {
	var segs [][2]int
	for _, s := range slots {
		segs = append(segs, refSegments(s)...)
	}
	for i := range segs {
		for j := i + 1; j < len(segs); j++ {
			if segs[i][0] < segs[j][1] && segs[j][0] < segs[i][1] {
				return ErrOverlap
			}
		}
	}
	return nil
}

func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	for round := 0; round < 200; round++ {
		n := rng.Intn(9)
		slots := make([]Slot, n)
		for i := range slots {
			slots[i] = Slot{
				Name:  fmt.Sprintf("s%d", i),
				Start: rng.Intn(1440),
				End:   rng.Intn(1441),
			}
		}
		if gotErr, refErr := Validate(slots), refValidate(slots); (gotErr == nil) != (refErr == nil) {
			t.Fatalf("第 %d 组校验不一致：got=%v ref=%v slots=%v", round, gotErr, refErr, slots)
		}
		for q := 0; q < 20; q++ {
			minute := rng.Intn(1440)
			got, want := StateAt(slots, minute), refStateAt(slots, minute)
			if got != want {
				t.Fatalf("第 %d 组 minute=%d 状态不一致：got=%s want=%s slots=%v", round, minute, got, want, slots)
			}
			if again := StateAt(slots, minute); again != got {
				t.Fatalf("第 %d 组 minute=%d 重复执行不一致：%s vs %s", round, minute, got, again)
			}
		}
	}
}

func BenchmarkStateAt(b *testing.B) {
	slots := make([]Slot, 100)
	for i := range slots {
		slots[i] = Slot{Name: fmt.Sprintf("s%d", i), Start: i * 14, End: i*14 + 14}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		StateAt(slots, i%1440)
	}
}
