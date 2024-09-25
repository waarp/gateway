package tasks

const (
	execScriptFile       = "exec_test_script.bat"
	execMoveScriptFile   = "execmove_test_script.bat"
	execOutputScriptFile = "execoutput_test_script.bat"
)

const scriptExecOK = `@ECHO OFF
ECHO %1
EXIT /B 0`

const scriptExecWarn = `@ECHO OFF
ECHO %1
EXIT /B 1`

const scriptExecFail = `@ECHO OFF
ECHO %1
EXIT /B 2`

const scriptExecInfinite = `@ECHO OFF
:loop
GOTO loop`

const scriptExecMove = `@ECHO OFF
MOVE %1 %2
ECHO %2
EXIT /B 0`

const execOutputNewFilename = "C:/new_name.file"

const scriptExecOutputFail = `@ECHO OFF
ECHO This is a message
ECHO NEWFILENAME:` + execOutputNewFilename + `
EXIT /B 2`
