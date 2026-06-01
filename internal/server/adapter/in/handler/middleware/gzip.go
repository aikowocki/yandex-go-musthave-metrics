package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	"go.uber.org/zap"
)

const EncodingGzip = "gzip"

var gzipWriterPool = sync.Pool{New: func() any { return gzip.NewWriter(io.Discard) }}
var gzipReaderPool sync.Pool

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(w)
	return &compressWriter{
		w:  w,
		zw: zw,
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	c.w.Header().Del(constants.HeaderContentLength)
	c.w.Header().Set(constants.HeaderContentEncoding, EncodingGzip)
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	c.w.Header().Set(constants.HeaderContentEncoding, EncodingGzip)
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	err := c.zw.Close()
	gzipWriterPool.Put(c.zw)
	return err
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	if v := gzipReaderPool.Get(); v != nil {
		zr := v.(*gzip.Reader)
		if err := zr.Reset(r); err != nil {
			// Reset не удался (например, тело — не валидный gzip)
			// но сам reader исправен возвращаем его в пул.
			gzipReaderPool.Put(zr)
			return nil, err
		}
		return &compressReader{
			r:  r,
			zr: zr,
		}, nil
	}

	// пул пустой - создаём новый
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	err := c.zr.Close()
	gzipReaderPool.Put(c.zr)
	if closeErr := c.r.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}

// WithGzipCompression возвращает middleware для прозрачного сжатия/распаковки HTTP-трафика.
// Если клиент поддерживает gzip (Accept-Encoding: gzip) — ответ сжимается.
// Если клиент отправляет сжатые данные (Content-Encoding: gzip) — тело распаковывается.
// Использует sync.Pool для переиспользования gzip.Writer и gzip.Reader.
func WithGzipCompression() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
			// который будем передавать следующей функции
			ow := w

			// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
			acceptEncoding := r.Header.Get("Accept-Encoding")
			supportsGzip := strings.Contains(acceptEncoding, "gzip")
			if supportsGzip {
				// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
				cw := newCompressWriter(w)
				// меняем оригинальный http.ResponseWriter на новый
				ow = cw
				// не забываем отправить клиенту все сжатые данные после завершения middleware
				defer func() {
					if err := cw.Close(); err != nil {
						zap.S().Warnw("failed to close gzip writer", zap.Error(err))
					}
				}()
			}

			// проверяем, что клиент отправил серверу сжатые данные в формате gzip
			contentEncoding := r.Header.Get("Content-Encoding")
			sendsGzip := strings.Contains(contentEncoding, "gzip")
			if sendsGzip {
				// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
				cr, err := newCompressReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// меняем тело запроса на новое
				r.Body = cr
				defer func() {
					if err := cr.Close(); err != nil {
						zap.S().Warnw("failed to close gzip reader", zap.Error(err))
					}
				}()
			}

			// передаём управление хендлеру
			h.ServeHTTP(ow, r)
		})
	}
}
