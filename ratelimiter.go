package main

import "time"

type rateLimiter struct {
	lastReadBytes int64     // Bytes read so far
	lastCheckTime time.Time // Time of the last check
	limit         int64     // Byte limit per second
}

// wait enforces the rate limit by pausing if the read bytes exceed the limit within a 1-second interval.
func (rl *rateLimiter) wait(currentReadBytes int64) {
	now := time.Now()

	// Calculate time elapsed since the last check
	elapsedTime := now.Sub(rl.lastCheckTime)

	// If the elapsed time is less than one second, enforce the rate limit
	if elapsedTime <= time.Second {
		bytesReadSinceLastCheck := currentReadBytes - rl.lastReadBytes

		// If the bytes read exceed the limit, calculate sleep time
		if bytesReadSinceLastCheck >= rl.limit {
			sleepDuration := time.Second - elapsedTime
			time.Sleep(sleepDuration)
			rl.lastReadBytes = currentReadBytes
			rl.lastCheckTime = time.Now()
		}
	} else {
		// Reset counters if more than one second has passed
		rl.lastReadBytes = currentReadBytes
		rl.lastCheckTime = now
	}
}
