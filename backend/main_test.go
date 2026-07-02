package main

import (
	"testing"
	"time"
)

func TestBuildDay(t *testing.T) {
	tests := []struct {
		name        string
		date        string
		wantBlocks  int
		wantFirstID string
	}{
		{"Sunday weekday", "2026-07-05", 7, "workout"},
		{"Monday weekday", "2026-07-06", 7, "workout"},
		{"Thursday weekday", "2026-07-09", 7, "workout"},
		{"Friday rest", "2026-07-10", 1, "rest"},
		{"Saturday rest", "2026-07-11", 1, "rest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			day := buildDay(tt.date)
			if day.Date != tt.date {
				t.Errorf("Date = %q, want %q", day.Date, tt.date)
			}
			if len(day.Blocks) != tt.wantBlocks {
				t.Fatalf("len(Blocks) = %d, want %d", len(day.Blocks), tt.wantBlocks)
			}
			if day.Blocks[0].ID != tt.wantFirstID {
				t.Errorf("first block ID = %q, want %q", day.Blocks[0].ID, tt.wantFirstID)
			}
		})
	}
}

func TestBuildDayBlockBounds(t *testing.T) {
	day := buildDay("2026-07-06")
	workout := day.Blocks[0]
	if workout.StartMin != 6*60+15 {
		t.Errorf("workout StartMin = %d, want 375", workout.StartMin)
	}
	if workout.EndMin != 7*60 {
		t.Errorf("workout EndMin = %d, want 420", workout.EndMin)
	}
	if workout.StartStr != "06:15" {
		t.Errorf("workout StartStr = %q, want 06:15", workout.StartStr)
	}
	if workout.EndStr != "07:00" {
		t.Errorf("workout EndStr = %q, want 07:00", workout.EndStr)
	}
	if workout.Planned.DurationMin != 45 {
		t.Errorf("workout DurationMin = %d, want 45", workout.Planned.DurationMin)
	}
}

func TestBuildDayRestBounds(t *testing.T) {
	day := buildDay("2026-07-10")
	rest := day.Blocks[0]
	if rest.StartMin != 0 || rest.EndMin != 23*60+59 {
		t.Errorf("rest bounds = %d-%d, want 0-1439", rest.StartMin, rest.EndMin)
	}
}

func TestCurrentBlock(t *testing.T) {
	day := buildDay("2026-07-06")

	tests := []struct {
		name     string
		hour     int
		min      int
		wantID   string
		wantNull bool
	}{
		{"06:15 boundary inclusive", 6, 15, "workout", false},
		{"06:14 free time before first block", 6, 14, "", true},
		{"08:00 go-block start", 8, 0, "go-block", false},
		{"09:30 work-am start", 9, 30, "work-am", false},
		{"12:59 still work-am", 12, 59, "work-am", false},
		{"13:00 lunch start", 13, 0, "lunch", false},
		{"18:59 wind-down", 18, 59, "wind-down", false},
		{"19:01 free time after last block", 19, 1, "", true},
		{"03:00 free time midnight", 3, 0, "", true},
		{"23:30 free time late night", 23, 30, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Date(2026, 7, 6, tt.hour, tt.min, 0, 0, dhaka)
			block := currentBlock(day, ts)
			if tt.wantNull {
				if block != nil {
					t.Errorf("currentBlock = %q, want nil", block.ID)
				}
				return
			}
			if block == nil {
				t.Fatalf("currentBlock = nil, want %q", tt.wantID)
			}
			if block.ID != tt.wantID {
				t.Errorf("currentBlock ID = %q, want %q", block.ID, tt.wantID)
			}
		})
	}
}

func TestSecondsRemainingInBlock(t *testing.T) {
	day := buildDay("2026-07-06")
	workout := &day.Blocks[0]
	ts := time.Date(2026, 7, 6, 6, 15, 0, 0, dhaka)
	rem := secondsRemainingInBlock(workout, ts)
	if rem != 45*60 {
		t.Errorf("at 06:15:00, rem = %d, want %d", rem, 45*60)
	}
	ts2 := time.Date(2026, 7, 6, 6, 59, 30, 0, dhaka)
	rem2 := secondsRemainingInBlock(workout, ts2)
	if rem2 != 30 {
		t.Errorf("at 06:59:30, rem = %d, want 30", rem2)
	}
}

func TestNextBlockStart(t *testing.T) {
	day := buildDay("2026-07-06")
	ts := time.Date(2026, 7, 6, 7, 30, 0, 0, dhaka)
	startMin, ok := nextBlockStart(day, ts)
	if !ok {
		t.Fatal("expected ok, got false")
	}
	if startMin != 8*60 {
		t.Errorf("nextBlockStart = %d, want 480", startMin)
	}
	ts2 := time.Date(2026, 7, 6, 19, 30, 0, 0, dhaka)
	_, ok2 := nextBlockStart(day, ts2)
	if ok2 {
		t.Error("after last block, expected ok=false")
	}
}

func TestPomodoroSecondsRemaining(t *testing.T) {
	now := time.Date(2026, 7, 6, 9, 30, 0, 0, dhaka)
	block := &Block{
		Pomodoro: &Pomodoro{
			StartedAt: now,
			EndsAt:    now.Add(25 * time.Minute),
		},
	}
	rem := pomodoroSecondsRemaining(block, now)
	if rem == nil || *rem != 25*60 {
		t.Errorf("at start, rem = %v, want 1500", rem)
	}
	mid := now.Add(10 * time.Minute)
	rem2 := pomodoroSecondsRemaining(block, mid)
	if rem2 == nil || *rem2 != 15*60 {
		t.Errorf("at +10min, rem = %v, want 900", rem2)
	}
	after := now.Add(26 * time.Minute)
	rem3 := pomodoroSecondsRemaining(block, after)
	if rem3 == nil || *rem3 != 0 {
		t.Errorf("at +26min, rem = %v, want 0", rem3)
	}
	noPomo := &Block{}
	rem4 := pomodoroSecondsRemaining(noPomo, now)
	if rem4 != nil {
		t.Errorf("no pomodoro, rem = %v, want nil", rem4)
	}
}

func TestStoreLazyEval(t *testing.T) {
	store := newStore()
	date := "2026-07-06"
	day1 := store.getOrBuild(date)
	if day1 == nil {
		t.Fatal("first getOrBuild returned nil")
	}
	day2 := store.getOrBuild(date)
	if day1 != day2 {
		t.Error("second getOrBuild returned a different Day pointer — not cached")
	}
}
