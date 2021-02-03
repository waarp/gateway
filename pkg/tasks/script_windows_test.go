// +build windows

package tasks

var execScriptFile = "exec_test_script.bat"
var execMoveScriptFile = "execmove_test_script.bat"
var execOutputScriptFile = "execoutput_test_script.bat"

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

const scriptExecMove = `@ECHO OFF
MOVE %1 %2
ECHO %2
EXIT /B 0`

const scriptExecOutputFail = `@ECHO OFF
ECHO This is a message
ECHO NEWFILENAME:new_name.file
EXIT /B 2`
