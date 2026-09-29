package npc

import "testing"

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
