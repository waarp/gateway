package jsontypes

import "encoding/json"

type Object map[string]any

func (j *Object) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err //nolint:wrapcheck //wrapping here adds nothing
	}

	if len(m) == 0 {
		return nil
	}

	*j = m

	return nil
}
