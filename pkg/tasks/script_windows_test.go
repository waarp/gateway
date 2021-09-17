// +build windows

package tasks

const lineSeparator = "\r\n"

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
ECHO %1
GOTO loop`

const scriptExecOutputFail = `@ECHO OFF
ECHO This is a message
ECHO NEWFILENAME:new_name.file
EXIT /B 2`
