package retry

import (
	"math"
	"time"
)

// Constant sleeps for delta duration
func Constant(delta time.Duration) Delayer {
	return func(a Attempt) {
		time.Sleep(delta)
	}
}

// Linear sleeps for delta * the number of attempts
func Linear(delta time.Duration) Delayer {
	return func(a Attempt) {
		time.Sleep(delta * time.Duration(a.Count))
	}
}

// Linear sleeps for delta * 2^attempts
func Exponential(base time.Duration) Delayer {
	return func(a Attempt) {
		time.Sleep(time.Duration(float64(base) * math.Exp(float64(a.Count))))
	}
}

// Fibonacci sleeps for delta * fib(attempts)
func Fibonacci(delta time.Duration) Delayer {
	return func(a Attempt) {
		time.Sleep(delta * time.Duration(fib(a.Count)))
	}
}

func fib(max uint) int64 {
	var (
		pre int64
		cur int64
		i   uint
	)
	for pre, cur, i = 0, 1, 0; i < max; i++ {
		pre, cur = cur, pre+cur
	}
	return pre
}
