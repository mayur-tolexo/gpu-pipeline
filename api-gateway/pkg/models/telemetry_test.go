package models_test

import (
	"testing"
	"time"

	"gpu-pipeline/api-gateway/pkg/models"
)

// TestTelemetryRecord_TableName tests the TableName method
func TestTelemetryRecord_TableName(t *testing.T) {
	record := &models.TelemetryRecord{}
	expectedTable := "telemetry"

	tableName := record.TableName()

	if tableName != expectedTable {
		t.Errorf("TableName() = %q, want %q", tableName, expectedTable)
	}
}

// TestTelemetryRecordTableNameVariations tests TableName with different record states
func TestTelemetryRecordTableNameVariations(t *testing.T) {
	testTime := time.Now()

	tests := []struct {
		name     string
		record   *models.TelemetryRecord
		expected string
	}{
		{
			name:     "empty record",
			record:   &models.TelemetryRecord{},
			expected: "telemetry",
		},
		{
			name: "populated record",
			record: &models.TelemetryRecord{
				ID:        1,
				GPUID:     "gpu-001",
				Timestamp: testTime,
				Data:      []byte(`{"temp": 65.5}`),
				CreatedAt: testTime,
			},
			expected: "telemetry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableName := tt.record.TableName()
			if tableName != tt.expected {
				t.Errorf("TableName() = %q, want %q", tableName, tt.expected)
			}
		})
	}
}
