package persistence

import (
	"bufio"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type fsyncStrategy int

const (
	fsyncAlways fsyncStrategy = iota + 1
	fsyncEverySec
	fsyncNo
)

// AOF Append Only File persistence
type AOF struct {
	file     *os.File
	writer   *bufio.Writer
	filename string
	strategy fsyncStrategy

	commandsChan chan []byte

	stopChan chan struct{}
	wg       sync.WaitGroup
	logger   *zap.Logger
}

// NewAOF construct AOF structure
func NewAOF(filename string, strategyStr string, logger *zap.Logger) (*AOF, error) {
	strategy := parseStrategy(strategyStr)

	// open file in Append mode, Create if not exists, Read/Write
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	aof := &AOF{
		file:         f,
		writer:       bufio.NewWriter(f), // default 4KB buffer
		filename:     filename,
		strategy:     strategy,
		commandsChan: make(chan []byte, 10000), // buffer for burst writes
		stopChan:     make(chan struct{}),
		logger:       logger,
	}

	// background disk writer
	aof.wg.Add(1)
	go aof.listen()

	return aof, nil
}

// Write send command in channel
func (a *AOF) Write(payload []byte) {
	// if channel is full, this WILL block, providing backpressure
	a.commandsChan <- payload
}

func (a *AOF) listen() {
	defer a.wg.Done()

	var ticker = time.NewTicker(1 * time.Second)

	switch a.strategy {
	case fsyncAlways:
		ticker.Stop()
	case fsyncNo:
		ticker.Stop()
		return
	default:
		defer ticker.Stop()
	}

	for {
		select {
		case p, ok := <-a.commandsChan:
			if !ok {
				return
			}
			if _, err := a.writer.Write(p); err != nil {
				a.logger.Error("AOF write error", zap.Error(err))
				continue
			}

			if a.strategy == fsyncAlways {
				a.flush()
				a.file.Sync() //nolint:errcheck
			}

		case <-ticker.C:
			if a.strategy == fsyncEverySec {
				a.flush()
				a.file.Sync() //nolint:errcheck
			}

		case <-a.stopChan:
			a.flush()
			a.file.Sync() //nolint:errcheck
			return
		}
	}
}

func (a *AOF) flush() {
	if err := a.writer.Flush(); err != nil {
		a.logger.Error("AOF flush error", zap.Error(err))
	}
}

// Close AOF persistence
func (a *AOF) Close() error {
	close(a.stopChan)

	a.wg.Wait() // wait for background routine to finish last flush
	return a.file.Close()
}

func parseStrategy(s string) fsyncStrategy {
	switch s {
	case "always":
		return fsyncAlways
	case "no":
		return fsyncNo
	default:
		return fsyncEverySec
	}
}
