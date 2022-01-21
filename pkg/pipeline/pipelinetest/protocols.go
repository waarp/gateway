package pipelinetest

type features struct {
	transID, size bool
}

//nolint:gochecknoglobals // global var is necessary here
var protocols = map[string]features{
	"sftp":  {transID: false, size: false},
	"r66":   {transID: true, size: true},
	"http":  {transID: true, size: true},
	"https": {transID: true, size: true},
}
