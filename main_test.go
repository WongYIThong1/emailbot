package main

import (
	"os"
	"testing"
)

func TestUsageCounterAccumulates(t *testing.T) {
	backup, err := os.ReadFile(usageFilePath)
	hasBackup := err == nil

	defer func() {
		if hasBackup {
			if err := os.WriteFile(usageFilePath, backup, 0600); err != nil {
				t.Fatalf("failed to restore usage file: %v", err)
			}
		} else {
			_ = os.Remove(usageFilePath)
		}
	}()

	_ = os.Remove(usageFilePath)

	total := readUsageCount()
	if total != 0 {
		t.Fatalf("expected zero when usage file missing, got %d", total)
	}

	total = updateUsageCount(total, 1_000_000) // 1000k
	if total < 1_000_000 {
		t.Fatalf("expected at least 1_000_000 after first update, got %d", total)
	}

	total = updateUsageCount(total, 5_000_000)
	if total < 6_000_000 {
		t.Fatalf("expected cumulative total >= 6_000_000, got %d", total)
	}

	persisted := readUsageCount()
	if persisted != total {
		t.Fatalf("usage counter not persisted, want %d got %d", total, persisted)
	}
}
