package tasks

func fileNotFound(path string, op ...string) *errFileNotFound {
	if len(op) > 0 {
		return &errFileNotFound{op[0], path}
	}
	return &errFileNotFound{"open", path}
}
