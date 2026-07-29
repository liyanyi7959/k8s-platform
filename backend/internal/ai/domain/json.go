package domain

import (
	"database/sql/driver"
	"encoding/json"
)

// JSONMap is the JSON-column value object used by AI persistence entities.
// Keeping it in the bounded context prevents the domain from depending on the
// legacy persistence package while preserving GORM's Scanner/Valuer contract.
type JSONMap map[string]any

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
