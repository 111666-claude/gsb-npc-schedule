package npc

import (
	"errors"
	"math/rand"
	"testing"
	"time"
)

func TestOvernightSlot(t *testing.T) {
	slots := []Slot{{Name: "sleep", Start: 22 * 60, End: 6 * 60}}
	cases := []struct {
		minute int
		want   string
	}{
		{2 * 60, "sleep"},   // 凌晨两点在跨零点区间内
		{22 * 60, "sleep"},  // 起点左闭
		{23 * 60, "sleep"},  // 午夜前
		{0, "sleep"},        // 零点
		{6*60 - 1, "sleep"}, // 结束前一分钟
		{6 * 60, "none"},    // 终点右开
		{12 * 60, "none"},   // 白天不在区间内
		{21 * 60, "none"},   // 开始前
	}
	for _, c := range cases {
		if got := StateAt(slots, c.minute); got != c.want {
			t.Errorf("minute=%d 期望 %s，得到 %s", c.minute, c.want, got)
		}
	}
}

func TestValidateOverlap(t *testing.T) {
	overlap := [][]Slot{
		{{Name: "a", Start: 9 * 60, End: 18 * 60}, {Name: "b", Start: 17 * 60, End: 20 * 60}},
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 5 * 60, End: 7 * 60}},   // 跨零点与普通区间重叠
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 21 * 60, End: 23 * 60}}, // 跨零点尾段重叠
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 20 * 60, End: 8 * 60}},  // 双跨零点重叠
		{{Name: "a", Start: 0, End: 1440}, {Name: "b", Start: 100, End: 200}},                 // 全天与任意区间重叠
	}
	for i, slots := range overlap {
		if err := Validate(slots); !errors.Is(err, ErrOverlap) {
			t.Errorf("用例 %d 期望 ErrOverlap，得到 %v", i, err)
		}
	}
	ok := [][]Slot{
		{{Name: "a", Start: 9 * 60, End: 18 * 60}, {Name: "b", Start: 18 * 60, End: 20 * 60}}, // 相邻不算重叠
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 6 * 60, End: 22 * 60}},  // 跨零点与补集相邻
		{{Name: "a", Start: 22 * 60, End: 6 * 60}, {Name: "b", Start: 12 * 60, End: 13 * 60}},
		{{Name: "a", Start: 100, End: 100}}, // 空区间不与任何区间重叠
	}
	for i, slots := range ok {
		if err := Validate(slots); err != nil {
			t.Errorf("用例 %d 不该报错：%v", i, err)
		}
	}
}

func TestBoundaryMinutes(t *testing.T) {
	slots := []Slot{{Name: "day", Start: 0, End: 1440}}
	if got := StateAt(slots, 0); got != "day" {
		t.Fatalf("00:00 应该命中：%s", got)
	}
	if got := StateAt(slots, 1439); got != "day" {
		t.Fatalf("23:59 应该命中：%s", got)
	}
	if err := Validate([]Slot{{Name: "x", Start: 0, End: 1440}}); err != nil {
		t.Fatalf("End=1440 合法：%v", err)
	}
	if err := Validate([]Slot{{Name: "x", Start: 1439, End: 1440}}); err != nil {
		t.Fatalf("最后一分钟合法：%v", err)
	}
	if err := Validate([]Slot{{Name: "x", Start: 1440, End: 1440}}); err == nil {
		t.Fatal("Start=1440 越界应该报错")
	}
}

func TestServerTimezone(t *testing.T) {
	slots := []Slot{{Name: "sleep", Start: 22 * 60, End: 6 * 60}}
	server := time.FixedZone("server", 8*3600)
	// UTC 2026-09-30 18:30 即服务器时间 2026-10-01 02:30，跨零点以服务器日期为准。
	utc := time.Date(2026, 9, 30, 18, 30, 0, 0, time.UTC)
	if got := StateAtTime(slots, utc, server); got != "sleep" {
		t.Fatalf("服务器时区 02:30 应该是 sleep：%s", got)
	}
	// 同一时刻按 UTC 解释是 18:30，不在区间内——证明用的是服务器时区。
	if got := StateAtTime(slots, utc, time.UTC); got != "none" {
		t.Fatalf("UTC 18:30 应该是 none：%s", got)
	}
}

func TestDeterministic(t *testing.T) {
	slots := []Slot{{Name: "sleep", Start: 22 * 60, End: 6 * 60}, {Name: "shop", Start: 9 * 60, End: 18 * 60}}
	for minute := 0; minute < 1440; minute++ {
		if StateAt(slots, minute) != StateAt(slots, minute) {
			t.Fatalf("minute=%d 重复执行结果不一致", minute)
		}
	}
}

func TestScale(t *testing.T) {
	slots := make([]Slot, 100)
	for i := range slots {
		slots[i] = Slot{Name: "s", Start: i * 14, End: i*14 + 10} // 100 个不重叠区间
	}
	if err := Validate(slots); err != nil {
		t.Fatalf("构造的日程应合法：%v", err)
	}
	start := time.Now()
	for q := 0; q < 100000; q++ {
		StateAt(slots, q%1440)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("10 万次查询耗时 %v，超过 1 秒", d)
	}
}

// 参考实现：跨零点归一 + 重叠校验（与实现独立编写，用于对照自证）。
func refValidate(slots []Slot) error {
	type interval struct{ lo, hi int }
	var ivs []interval
	for _, s := range slots {
		if s.Start < 0 || s.Start > 1439 || s.End < 0 || s.End > 1440 {
			return errors.New("npc: 时间越界")
		}
	}
	for _, s := range slots {
		switch {
		case s.Start < s.End:
			ivs = append(ivs, interval{s.Start, s.End})
		case s.Start > s.End:
			ivs = append(ivs, interval{s.Start, 1440}, interval{0, s.End})
		}
	}
	for i := range ivs {
		for j := i + 1; j < len(ivs); j++ {
			if ivs[i].lo < ivs[j].hi && ivs[j].lo < ivs[i].hi {
				return ErrOverlap
			}
		}
	}
	return nil
}

func refStateAt(slots []Slot, minute int) string {
	for _, s := range slots {
		switch {
		case s.Start < s.End:
			if minute >= s.Start && minute < s.End {
				return s.Name
			}
		case s.Start > s.End:
			if minute >= s.Start || minute < s.End {
				return s.Name
			}
		}
	}
	return "none"
}

func TestDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	diff := 0
	for round := 0; round < 200; round++ {
		n := rng.Intn(9)
		slots := make([]Slot, n)
		for i := range slots {
			slots[i] = Slot{
				Name:  string(rune('a' + i)),
				Start: rng.Intn(1445) - 2, // 覆盖越界输入
				End:   rng.Intn(1445) - 2,
			}
		}
		gotErr := Validate(slots)
		wantErr := refValidate(slots)
		if (gotErr == nil) != (wantErr == nil) ||
			(gotErr != nil && errors.Is(gotErr, ErrOverlap) != errors.Is(wantErr, ErrOverlap)) {
			t.Errorf("第 %d 组 Validate 不一致：got=%v want=%v slots=%v", round, gotErr, wantErr, slots)
			diff++
		}
		for q := 0; q < 20; q++ {
			minute := rng.Intn(1440)
			if got, want := StateAt(slots, minute), refStateAt(slots, minute); got != want {
				t.Errorf("第 %d 组 minute=%d 状态不一致：got=%s want=%s", round, minute, got, want)
				diff++
			}
		}
	}
	if diff != 0 {
		t.Fatalf("对照差异 %d 处，必须为 0", diff)
	}
}
