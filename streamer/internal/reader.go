package internal

import (
	"encoding/csv"
	"os"
)

type Record map[string]string

func ReadCSV(filePath string) ([]Record, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := rows[0]
	var records []Record

	for _, row := range rows[1:] {
		rec := make(Record)
		for i, val := range row {
			rec[headers[i]] = val
		}
		records = append(records, rec)
	}

	return records, nil
}
