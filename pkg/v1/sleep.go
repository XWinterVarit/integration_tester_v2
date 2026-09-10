package v1

import (
	"fmt"
	"time"
)

// Sleep pauses execution for the given duration.
// In dry-run mode it records the action but skips the actual delay.
func Sleep(d time.Duration) {
	RecordAction(fmt.Sprintf("Sleep %s", d), func() { Sleep(d) })
	if IsDryRun() {
		return
	}
	Log(LogTypeInfo, "Sleep", fmt.Sprintf("Duration: %s", d))
	time.Sleep(d)
}
