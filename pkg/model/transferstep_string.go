// Code generated by "stringer -type=TransferStep"; DO NOT EDIT.

package model

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[StepNone-0]
	_ = x[StepSetup-1]
	_ = x[StepPreTasks-2]
	_ = x[StepData-3]
	_ = x[StepPostTasks-4]
	_ = x[StepErrorTasks-5]
	_ = x[StepFinalization-6]
}

const _TransferStep_name = "StepNoneStepSetupStepPreTasksStepDataStepPostTasksStepErrorTasksStepFinalization"

var _TransferStep_index = [...]uint8{0, 8, 17, 29, 37, 50, 64, 80}

func (i TransferStep) String() string {
	if i >= TransferStep(len(_TransferStep_index)-1) {
		return "TransferStep(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TransferStep_name[_TransferStep_index[i]:_TransferStep_index[i+1]]
}
