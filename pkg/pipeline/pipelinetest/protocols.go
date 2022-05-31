package pipelinetest

type features struct {
	transID, ruleName, size bool
}

//nolint:gochecknoglobals // global var is necessary here
var protocols = map[string]features{
	"sftp":  {transID: false, ruleName: false, size: false},
	"r66":   {transID: true, ruleName: true, size: true},
	"http":  {transID: true, ruleName: true, size: true},
	"https": {transID: true, ruleName: true, size: true},
}
