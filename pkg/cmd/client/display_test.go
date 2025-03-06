package wg

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestColors(t *testing.T) {
	t.Log(Style0.sprintf("=== Main title ==="))
	t.Log(Style1.sprintL("Item", "value"))
	t.Log(Style22.sprintL("Property", "value"))
	t.Log(Style22.sprintL("Empty property", empty))
	t.Log(Style333.sprintL("Sub-property", "value"))
	t.Log(Style4444.sprintL("Sub-sub-property", "value"))

	t.Logf("Status [%s]", coloredStatus(types.StatusPlanned))
	t.Logf("Status [%s]", coloredStatus(types.StatusRunning))
	t.Logf("Status [%s]", coloredStatus(types.StatusPaused))
	t.Logf("Status [%s]", coloredStatus(types.StatusInterrupted))
	t.Logf("Status [%s]", coloredStatus(types.StatusCancelled))
	t.Logf("Status [%s]", coloredStatus(types.StatusError))
	t.Logf("Status [%s]", coloredStatus(types.StatusDone))
	t.Logf("Status [%s]", coloredStatus("OTHER"))
}
