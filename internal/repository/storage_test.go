package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemStorage_UpdateCounter(t *testing.T) {
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name    string
		storage *MemStorage
		args    args
		want    int64
	}{
		{
			name:    "Init counter",
			storage: NewMemStorage(),
			args: args{
				name:  "test",
				value: 1,
			},
			want: 1,
		},
		{
			name: "Update counter",
			storage: &MemStorage{
				gauges:   make(map[string]float64),
				counters: map[string]int64{"test": 5},
			},
			args: args{
				name:  "test",
				value: 1,
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.storage.UpdateCounter(tt.args.name, tt.args.value)
			assert.Equal(t, tt.want, tt.storage.counters[tt.args.name])
		})
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name    string
		storage *MemStorage
		args    args
		want    float64
	}{
		{
			name:    "Create new gauge",
			storage: NewMemStorage(),
			args: args{
				name:  "test",
				value: 3.14,
			},
			want: 3.14,
		},
		{
			name: "Update gauge",
			storage: &MemStorage{
				gauges:   map[string]float64{"test": 1.0},
				counters: make(map[string]int64),
			},
			args: args{
				name:  "test",
				value: 5.5,
			},
			want: 5.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.storage.UpdateGauge(tt.args.name, tt.args.value)
			assert.Equal(t, tt.want, tt.storage.gauges[tt.args.name])
		})
	}
}
