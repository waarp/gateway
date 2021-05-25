package pipeline

import (
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func WaitEndClientTransfer(c C, pip *ClientPipeline) {
	waitEndTransfer(c, pip.pip)
}

func WaitEndServerTransfer(c C, pip *ServerPipeline) {
	waitEndTransfer(c, pip.Pipeline)
}

func waitEndTransfer(c C, pip *Pipeline) {
	timeout := time.NewTimer(time.Second * 300)
	ticker := time.NewTicker(time.Millisecond * 100)
	defer func() {
		timeout.Stop()
		ticker.Stop()
	}()

	for {
		select {
		case <-timeout.C:
			c.So("Error-tasks timeout exceeded", ShouldBeBlank)
		case <-ticker.C:
			switch pip.machine.Current() {
			case "in error", "all done":
				return
			default:
			}
		}
	}
}
