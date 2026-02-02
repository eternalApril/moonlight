package persistence

import (
	"bufio"
	"io"
	"os"
	"time"

	"github.com/eternalApril/moonlight/internal/storage"
	"go.uber.org/zap"
)

type RDB struct {
	filename string
	logger   *zap.Logger
}

func NewRDB(filename string, logger *zap.Logger) *RDB {
	return &RDB{
		filename: filename,
		logger:   logger,
	}
}

// Save performs an atomic save operation
func (r *RDB) Save(db storage.Storage) error {
	start := time.Now()
	tmpFile := r.filename + ".tmp"

	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriterSize(f, 4*1024*1024)

	if _, err := writer.WriteString("MOONRES1"); err != nil {
		return err
	}

	if err := db.Snapshot(writer); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}
	f.Close()

	if err := os.Rename(tmpFile, r.filename); err != nil {
		return err
	}

	r.logger.Info("RDB saved successfully",
		zap.String("file", r.filename),
		zap.Duration("duration", time.Since(start)),
	)
	return nil
}

func (r *RDB) Load(db storage.Storage) error {
	f, err := os.Open(r.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	header := make([]byte, 8)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	if string(header) != "MOONRES1" {
		r.logger.Warn("Invalid RDB header, assuming empty or incompatible", zap.String("header", string(header)))
		return nil
	}

	start := time.Now()
	if err := db.Restore(reader); err != nil {
		return err
	}

	r.logger.Info("RDB loaded", zap.Duration("duration", time.Since(start)))
	return nil
}
