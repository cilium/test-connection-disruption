package internal

import (
	"fmt"
	"math"
	"os"
	"syscall"
	"time"

	"golang.org/x/exp/constraints"
)

const MsgSize = 16

// ErrExit prints the message and error to stderr and exits with status 1 if err
// is not nil.
func ErrExit(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
		os.Exit(1)
	}
}

// BeNice sets the calling process' priority to the lowest possible value.
func BeNice() error {
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, os.Getpid(), 19); err != nil {
		return fmt.Errorf("set priority: %w", err)
	}
	return nil
}

// Sleep for the specified duration. No-op if d is negative or zero.
func Sleep(d time.Duration) {
	if d <= 0 {
		return
	}

	ts := syscall.NsecToTimespec(d.Nanoseconds())
	err := syscall.Nanosleep(&ts, nil)
	switch err {
	case nil, syscall.EINTR:
		// Interrupted sleep is fine for our purposes.
	case syscall.EINVAL:
		panic(fmt.Sprintf("nanosleep interval %s out of range: %s", d, err))
	default:
		panic(fmt.Sprintf("nanosleep: %s", err))
	}
}

func Pause(interval time.Duration, pauseDuration time.Duration, startTime time.Time) (nextPauseDuration time.Duration) {
	// Sleep for the duration determined during the previous round. Use a
	// direct call to nanosleep(2) since the regular [time.Sleep] is
	// implemented by the Go runtime and gets coalesced to reduce syscall
	// overhead. This leads to wildly unexpected sleep durations.
	Sleep(pauseDuration)

	// Adjust the sleep interval for the next cycle based on the time it took
	// to write to the socket and when the OS scheduler woke us up.
	delta := interval - time.Since(startTime)

	// Smoothen the approach to the target interval by adjusting the pause
	// interval by half the delta.
	pauseDuration += (delta / 2)

	// Ensure pause stays within bounds. On a permanent deficit, it would
	// run negative and overflow at some point.
	return min(max(pauseDuration, -interval), interval)
}

// ByteString returns a human-readable string representation of the given byte
// count. Taken from https://stackoverflow.com/a/1094933/1333724.
func ByteString[T constraints.Integer](b T) string {
	bf := float64(b)
	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}
