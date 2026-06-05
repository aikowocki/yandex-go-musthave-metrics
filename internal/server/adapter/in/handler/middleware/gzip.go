package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/pool"
	"go.uber.org/zap"
)

const EncodingGzip = "gzip"

var gzipWriterPool = pool.NewFunc(
	func() *gzip.Writer { return gzip.NewWriter(io.Discard) },
	func(gz *gzip.Writer) {
		if err := gz.Close(); err != nil {
			zap.S().Debugw("gzip writer close failed", "error", err)
		}
		gz.Reset(io.Discard)
	},
)
var gzipReaderPool = pool.NewFunc(
	func() *gzip.Reader { return new(gzip.Reader) },
	func(zr *gzip.Reader) {
		if err := zr.Close(); err != nil {
			zap.S().Debugw("gzip reader close failed", "error", err)
		}
	},
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	zw := gzipWriterPool.Get()
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

// Close возвращает gzip.Writer в пул. resetFn выполнит Close и Reset.
func (c *compressWriter) Close() error {
	gzipWriterPool.Put(c.zw)
	return nil
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	body io.ReadCloser // Оригинальное тело HTTP-запроса
	zr   *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr := gzipReaderPool.Get()
	//Reset(r) — это не очистка, это инициализация: то бишь начни читать gzip-данные из r
	if err := zr.Reset(r); err != nil {
		// Reset не удался (например, тело — не валидный gzip)
		// но сам reader исправен возвращаем его в пул.
		gzipReaderPool.Put(zr)
		return nil, err
	}
	return &compressReader{
		body: r,
		zr:   zr,
	}, nil
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	gzipReaderPool.Put(c.zr) // вернуть gzip.Reader в пул
	return c.body.Close()    // закрыть оригинальное HTTP body
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
				defer func() { _ = cw.Close() }()

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
				defer func() { _ = cr.Close() }()
			}

			// передаём управление хендлеру
			h.ServeHTTP(ow, r)
		})
	}
}
