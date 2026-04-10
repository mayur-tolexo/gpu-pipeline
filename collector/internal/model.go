package internal

import "time"

type Telemetry struct {
	ID        uint      `gorm:"primaryKey"`
	GPUID     string    `gorm:"column:gpu_id;index"`
	Timestamp time.Time `gorm:"column:timestamp;index"`
	Data      []byte    `gorm:"column:data;type:jsonb"`
}

// TableName overrides the table name used by Telemetry to `telemetry`
func (Telemetry) TableName() string {
	return "telemetry"
}
