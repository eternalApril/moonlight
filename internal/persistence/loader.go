package persistence

import (
	"io"
	"os"

	"github.com/eternalApril/moonlight/internal/resp"
)

// Load reads the AOF file and returns a channel of commands to be replayed
func (a *AOF) Load() ([]resp.Value, error) {
	file, err := os.Open(a.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Fresh start
		}
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	reader := resp.NewDecoder(file)
	var commands []resp.Value

	for {
		val, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		commands = append(commands, val)
	}

	return commands, nil
}
