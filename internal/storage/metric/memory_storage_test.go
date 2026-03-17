package metric

import "testing"

func TestMemStorage(t *testing.T) {
	runStorageTests(t, NewMetricMemoryStorage())
}
