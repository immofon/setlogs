package setlogs

import (
	"encoding/csv"
	"io"
	"log"
	"strings"
)

func ReadCSV(r io.Reader, logs_type Type) SetLog {
	setlog := New(logs_type)
	setlog.Comment = "Read from csv"

	csv_r := csv.NewReader(r)
	_keys, err := csv_r.Read()
	if err != nil {
		return setlog
	}
	keys := make([]string, 0, len(_keys))
	for _, key := range _keys {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}

	for {
		raw, err := csv_r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		record := make(Record)
		for i, key := range keys {
			value := raw[i]
			record[key] = strings.TrimSpace(value)
		}

		setlog.AppendRecords(record)
	}
	return setlog
}
