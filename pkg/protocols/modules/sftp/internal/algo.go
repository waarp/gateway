package internal

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

func (a Algo) IsValid(forServer bool) bool {
	if forServer {
		return a.ValidFor == Both || a.ValidFor == OnlyServer
	} else {
		return a.ValidFor == Both || a.ValidFor == OnlyClient
	}
}

type Algos []Algo

func (a Algos) IsAlgoValid(name string, forServer bool) bool {
	for _, algo := range a {
		if name == algo.Name {
			return algo.IsValid(forServer)
		}
	}

	return false
}

func (a Algos) ServerDefaults() []string {
	list := make([]string, 0, len(a))

	for _, algo := range a {
		if !algo.DisabledByDefault && algo.IsValid(true) {
			list = append(list, algo.Name)
		}
	}

	return list
}

func (a Algos) ClientDefaults() []string {
	list := make([]string, 0, len(a))

	for _, algo := range a {
		if !algo.DisabledByDefault && algo.IsValid(false) {
			list = append(list, algo.Name)
		}
	}

	return list
}
