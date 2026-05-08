package agents

type agentStrategy interface {
	ExtractContext(current_working_directory string) string
	LatestSessionID(current_working_directory string) string
}
