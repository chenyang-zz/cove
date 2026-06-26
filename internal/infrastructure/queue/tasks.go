package queue

const (
	TaskParseDocument     = "parse:document"
	TaskParseImage        = "parse:image"
	TaskMemoryExtract     = "memory:extract"
	TaskMemoryConsolidate = "memory:consolidate"
	TaskResearchRun       = "research:run"
	QueueDefault          = "default"
	QueueParse            = "parse"
	QueueMemory           = "memory"
	QueueResearch         = "research"
	QueueBeat             = "beat"
)

func TaskNames() []string {
	return []string{
		TaskParseDocument,
		TaskParseImage,
		TaskMemoryExtract,
		TaskMemoryConsolidate,
		TaskResearchRun,
	}
}
