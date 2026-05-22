package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type FileObserver struct {
	path string
	mu   sync.Mutex
	f    *os.File
}

func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("audit file open %q: %w", path, err)
	}
	return &FileObserver{path: path, f: f}, nil
}

func (o *FileObserver) Name() string { return "file:" + o.path }

func (o *FileObserver) Notify(_ context.Context, event entity.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit file marshal: %w", err)
	}
	data = append(data, '\n')

	o.mu.Lock()
	defer o.mu.Unlock()
	if _, err := o.f.Write(data); err != nil {
		return fmt.Errorf("audit file write: %w", err)
	}
	return nil
}

func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.f == nil {
		return nil
	}
	err := o.f.Close()
	o.f = nil
	return err
}
