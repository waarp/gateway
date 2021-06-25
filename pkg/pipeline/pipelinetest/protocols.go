package pipelinetest

type features struct {
	transID, size bool
}

var protocols = map[string]features{
	"sftp": {transID: false, size: false},
	"r66":  {transID: true, size: true},
	"http": {transID: true, size: true},
}
