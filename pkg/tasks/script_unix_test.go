// +build !windows

package tasks

var lineSeparator = "\n"

var execScriptFile = "./exec_test_script.sh"
var execMoveScriptFile = "./execmove_test_script.sh"
var execOutputScriptFile = "./execoutput_test_script.sh"

const scriptExecOK = `#!/bin/sh
echo $1
exit 0`

const scriptExecWarn = `#!/bin/sh
echo $1
exit 1`

const scriptExecFail = `#!/bin/sh
echo $1
exit 2`

const scriptExecInfinite = `#!/bin/sh
while [ true ]; do
  echo $1
done`

const scriptExecOutputFail = `#!/bin/sh
echo "This is a message"
echo "NEWFILENAME:new_name.file"
exit 2`
