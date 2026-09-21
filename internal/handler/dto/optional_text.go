package dto

import "encoding/json"

// OptionalText distinguishes an omitted update from an explicit null.
// A *string alone cannot represent all three states of a JSON field.
type OptionalText struct {
	Present bool
	Value   *string
}

func (v *OptionalText) UnmarshalJSON(data []byte) error {
	var value *string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Present, v.Value = true, value
	return nil
}

func (v OptionalText) MarshalJSON() ([]byte, error) { return json.Marshal(v.Value) }

func (v OptionalText) IsZero() bool { return !v.Present }
