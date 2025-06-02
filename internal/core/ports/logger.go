package ports

type LoggerPort interface {
	Info(msg string)
	Error(msg string, err error)
	Warning(msg string)
	Close()
}
