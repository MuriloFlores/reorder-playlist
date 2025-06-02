package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Logger interface {
	Info(msg string)
	Error(msg string, err error)
	Warning(msg string)
	Close()
}

type LogData struct {
	File      string `json:"file"`
	Function  string `json:"function"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Err       string `json:"err,omitempty"`
	Timestamp string `json:"timestamp"`
}

type fileLogger struct {
	mu        sync.Mutex
	logFile   *os.File
	encoder   *json.Encoder
	logDir    string
	logPrefix string
}

func NewFileLogger(logDir, logPrefix string) (Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar o diretório de log '%s': %w", logDir, err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := fmt.Sprintf("%s_%s.json", logPrefix, timestamp)

	logFilePath := filepath.Join(logDir, logFileName)

	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir/criar o arquivo de log '%s': %w", logFilePath, err)
	}

	return &fileLogger{
		logFile:   file,
		encoder:   json.NewEncoder(file),
		logDir:    logDir,
		logPrefix: logPrefix,
	}, nil
}

func (l *fileLogger) writeLogInternal(level string, msg string, errIn error, skip int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile == nil {
		fmt.Fprintf(os.Stderr, "Logger está fechado, não é possível escrever log: %s\n", msg)
		return
	}

	pc, filePath, line, ok := runtime.Caller(skip)

	var funcName string
	var shortFileName string

	if ok {
		shortFileName = filepath.Base(filePath)
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			parts := strings.Split(fn.Name(), ".")
			funcName = parts[len(parts)-1]
		} else {
			funcName = "???"
		}

		_ = line
	} else {
		shortFileName = "???"
		funcName = "???"
	}

	logEntry := LogData{
		Timestamp: time.Now().Format(time.RFC3339), // Para o conteúdo do log, RFC3339 é bom
		File:      shortFileName,
		Function:  funcName,
		Level:     level,
		Message:   msg,
	}

	if errIn != nil {
		logEntry.Err = errIn.Error()
	}

	if err := l.encoder.Encode(logEntry); err != nil {
		fmt.Fprintf(os.Stderr, "Falha ao escrever log no arquivo: %v\n", err)
	}
}

func (l *fileLogger) Info(msg string) {
	l.writeLogInternal("INFO", msg, nil, 2) // Níveis de log em maiúsculo por convenção
}

func (l *fileLogger) Error(msg string, err error) {
	l.writeLogInternal("ERROR", msg, err, 2)
}

func (l *fileLogger) Warning(msg string) {
	l.writeLogInternal("WARNING", msg, nil, 2)
}

func (l *fileLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		err := l.logFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao fechar arquivo de log: %v\n", err)
		}
		l.logFile = nil
	}
}
