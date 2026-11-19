package protocols

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestNoDuplicateInfoKeys(t *testing.T) {
	t.Parallel()

	keys := make(map[string]struct{}, model.ReservedInfoKeys.Len())
	for _, key := range model.ReservedInfoKeys.Iter() {
		if _, exists := keys[key.Name]; exists {
			t.Errorf("Duplicate key: %s", key.Name)
		}

		keys[key.Name] = struct{}{}
	}
}
