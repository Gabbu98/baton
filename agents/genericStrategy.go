package agents

type GenericStrategy struct {
	directory string
}

func NewGenericStrategy(dir string) *GenericStrategy {
	return &GenericStrategy{directory: dir}
}
