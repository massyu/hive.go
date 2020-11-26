package daemon_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/massyu/hive.go/daemon"
	"github.com/massyu/hive.go/typeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// graceTime for go routines to start
const graceTime = 5 * time.Millisecond

func TestShutdown(t *testing.T) {
	d := daemon.New()
	d.Start()
	assert.True(t, d.IsRunning())
	d.ShutdownAndWait()
	assert.False(t, d.IsRunning())
	assert.True(t, d.IsStopped())
}

func TestShutdownWithoutStart(t *testing.T) {
	d := daemon.New()
	d.ShutdownAndWait()
	assert.True(t, d.IsStopped())
}

func TestStartShutdown(t *testing.T) {
	d := daemon.New()

	var isShutdown, wasStarted typeutils.AtomicBool
	err := d.BackgroundWorker("A", func(shutdownSignal <-chan struct{}) {
		wasStarted.Set()
		<-shutdownSignal
		isShutdown.Set()
	})
	require.NoError(t, err)
	time.Sleep(graceTime)

	assert.False(t, wasStarted.IsSet())
	assert.False(t, isShutdown.IsSet())

	d.Start()
	time.Sleep(graceTime)
	assert.True(t, wasStarted.IsSet())
	assert.False(t, isShutdown.IsSet())

	d.ShutdownAndWait()
	assert.True(t, wasStarted.IsSet())
	assert.True(t, isShutdown.IsSet())
}

func TestRun(t *testing.T) {
	d := daemon.New()

	var workerStarted typeutils.AtomicBool
	err := d.BackgroundWorker("A", func(shutdownSignal <-chan struct{}) {
		workerStarted.Set()
		<-shutdownSignal
	})
	require.NoError(t, err)

	assert.False(t, workerStarted.IsSet())

	var runFinished typeutils.AtomicBool
	go func() {
		d.Run()
		runFinished.Set()
	}()
	time.Sleep(graceTime)
	assert.False(t, runFinished.IsSet())

	d.ShutdownAndWait()
	time.Sleep(graceTime)
	assert.True(t, runFinished.IsSet())
}

func TestShutdownOrder(t *testing.T) {
	d := daemon.New()

	orders := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	feedback := make(chan int, len(orders))
	for _, order := range orders {
		o := order
		err := d.BackgroundWorker(strconv.Itoa(o), func(shutdownSignal <-chan struct{}) {
			<-shutdownSignal
			feedback <- o
		}, o)
		require.NoError(t, err)
	}

	d.Start()
	d.ShutdownAndWait()
	close(feedback)

	for i := len(orders) - 1; i >= 0; i-- {
		assert.Equal(t, i, <-feedback)
	}
}

func TestGetRunningBackgroundWorkers(t *testing.T) {
	d := daemon.New()

	err := d.BackgroundWorker("quick", func(shutdownSignal <-chan struct{}) {
		<-shutdownSignal
	})
	require.NoError(t, err)

	blocker := make(chan struct{})
	err = d.BackgroundWorker("blocked", func(shutdownSignal <-chan struct{}) {
		<-shutdownSignal
		<-blocker
	})
	require.NoError(t, err)

	d.Start()
	time.Sleep(graceTime)
	// both workers should be running
	assert.ElementsMatch(t, []string{"quick", "blocked"}, d.GetRunningBackgroundWorkers())

	d.Shutdown()
	time.Sleep(graceTime)
	// only the blocked worker should still be running
	assert.ElementsMatch(t, []string{"blocked"}, d.GetRunningBackgroundWorkers())
	// let the blocker close
	close(blocker)
}

func TestShutdownTwice(t *testing.T) {
	d := daemon.New()

	err := d.BackgroundWorker("A", func(shutdownSignal <-chan struct{}) {
		<-shutdownSignal
		// sleep longer than the grace time before shutting down
		time.Sleep(2 * graceTime)
	})
	require.NoError(t, err)

	d.Start()
	time.Sleep(graceTime)

	d.Shutdown()
	time.Sleep(graceTime)
	d.ShutdownAndWait()
	assert.False(t, d.IsRunning())
}

func TestReRun(t *testing.T) {
	d := daemon.New()

	terminate := make(chan struct{}, 1)
	require.NoError(t, d.BackgroundWorker("A", func(shutdownSignal <-chan struct{}) {
		<-terminate
	}))
	d.Start()

	terminate <- struct{}{}
	time.Sleep(graceTime)

	var wasStarted typeutils.AtomicBool
	require.NoError(t, d.BackgroundWorker("A", func(shutdownSignal <-chan struct{}) {
		wasStarted.Set()
		<-shutdownSignal
	}))

	d.ShutdownAndWait()
	assert.True(t, wasStarted.IsSet())
}
