package config

type AlgoTarget uint8

const (
	Both AlgoTarget = iota
	OnlyClient
	OnlyServer
)

type Algo struct {
	Name              string
	ValidFor          AlgoTarget
	DisabledByDefault bool
}

func (a Algo) isValid(forServer bool) bool {
	if forServer {
		return a.ValidFor == Both || a.ValidFor == OnlyServer
	} else {
		return a.ValidFor == Both || a.ValidFor == OnlyClient
	}
}

type Algos []Algo

func (a Algos) isAlgoValid(name string, forServer bool) bool {
	for _, algo := range a {
		if name == algo.Name {
			return algo.isValid(forServer)
		}
	}

	return false
}

func (a Algos) ServerDefaults() []string {
	list := make([]string, 0, len(a))

	for _, algo := range a {
		if !algo.DisabledByDefault && algo.isValid(true) {
			list = append(list, algo.Name)
		}
	}

	return list
}

func (a Algos) ClientDefaults() []string {
	list := make([]string, 0, len(a))

	for _, algo := range a {
		if !algo.DisabledByDefault && algo.isValid(false) {
			list = append(list, algo.Name)
		}
	}

	return list
}
