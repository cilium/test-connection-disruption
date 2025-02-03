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
