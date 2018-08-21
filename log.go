//high level log wrapper, so it can output different log based on level
package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const (
	Ldate         = log.Ldate
	Llongfile     = log.Llongfile
	Lmicroseconds = log.Lmicroseconds
	Lshortfile    = log.Lshortfile
	LstdFlags     = log.LstdFlags
	Ltime         = log.Ltime
)

type (
	LogLevel int
	LogType  int
)

const (
	LOG_FATAL   = LogType(0x1)
	LOG_ERROR   = LogType(0x2)
	LOG_WARNING = LogType(0x4)
	LOG_INFO    = LogType(0x8)
	LOG_DEBUG   = LogType(0x10)
)

const (
	LOG_LEVEL_NONE  = LogLevel(0x0)
	LOG_LEVEL_FATAL = LOG_LEVEL_NONE | LogLevel(LOG_FATAL)
	LOG_LEVEL_ERROR = LOG_LEVEL_FATAL | LogLevel(LOG_ERROR)
	LOG_LEVEL_WARN  = LOG_LEVEL_ERROR | LogLevel(LOG_WARNING)
	LOG_LEVEL_INFO  = LOG_LEVEL_WARN | LogLevel(LOG_INFO)
	LOG_LEVEL_DEBUG = LOG_LEVEL_INFO | LogLevel(LOG_DEBUG)
	LOG_LEVEL_ALL   = LOG_LEVEL_DEBUG
)

const FORMAT_TIME_DAY string = "20060102"

const FORMAT_TIME_HOUR string = "2006010215"

type Logger struct {
	_log  *log.Logger
	level LogLevel

	TimeFormat string
	SuffixName string
	FileName   string
	logSuffix  string
	fd         *os.File

	lock sync.Mutex
}

func (l *Logger) Init(jsonConfig string) error {
	err := json.Unmarshal([]byte(jsonConfig), l)
	if err != nil {
		return err
	}
	if len(l.FileName) == 0 {
		return errors.New("jsonconfig must have filename")
	}
	return l.SetOutputByName(l.FileName)
}
func (l *Logger) SetLogger(configs ...string) error {
	config := append(configs, "{}")[0]
	err := l.Init(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logs.SetLogger: "+err.Error())
		return err
	}
	return nil
}
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) SetLevelByString(level string) {
	l.level = StringToLogLevel(level)
}

func (l *Logger) SetRotateByTimeFormat(format string) {
	l.TimeFormat = format
	l.logSuffix = time.Now().Format(l.TimeFormat)
}
func (l *Logger) rotate() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	var suffix string
	//异常处理
	suffix = time.Now().Format(l.TimeFormat)

	// Notice: if suffix is not equal to l.LogSuffix, then rotate
	if suffix != l.logSuffix {
		err := l.doRotate(suffix)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Logger) doRotate(suffix string) error {
	// Notice: Not check error, is this ok?
	l.fd.Close()

	//lastFileName := l.fileName + "." + l.logSuffix + l.SuffixName
	/*err := os.Rename(l.fileName, lastFileName)
	if err != nil {
		return err
	}*/

	err := l.SetOutputByName(l.FileName)
	if err != nil {
		return err
	}

	l.logSuffix = suffix

	return nil
}

func (l *Logger) SetOutput(out io.Writer) {
	l._log = log.New(out, l._log.Prefix(), l._log.Flags())
}

func (l *Logger) SetOutputByName(path string) error {
	f, err := os.OpenFile(path+"."+time.Now().Format(l.TimeFormat)+l.SuffixName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}

	l.SetOutput(f)

	l.FileName = path
	l.fd = f

	return err
}

func (l *Logger) log(t LogType, v ...interface{}) {
	if l.level|LogLevel(t) != l.level {
		return
	}

	err := l.rotate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return
	}

	v1 := make([]interface{}, len(v)+2)
	logStr := LogTypeToString(t)

	v1[0] = "[" + logStr + "]"
	copy(v1[1:], v)
	v1[len(v)+1] = ""

	s := fmt.Sprintln(v1...)
	l._log.Output(4, s)
}

func (l *Logger) logf(t LogType, format string, v ...interface{}) {
	if l.level|LogLevel(t) != l.level {
		return
	}

	err := l.rotate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return
	}

	logStr := LogTypeToString(t)
	var s string

	s = "[" + logStr + "] " + fmt.Sprintf(format, v...)

	l._log.Output(4, s)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.log(LOG_FATAL, v...)
	os.Exit(-1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logf(LOG_FATAL, format, v...)
	os.Exit(-1)
}

func (l *Logger) Error(v ...interface{}) {
	l.log(LOG_ERROR, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logf(LOG_ERROR, format, v...)
}

func (l *Logger) Warning(v ...interface{}) {
	l.log(LOG_WARNING, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.logf(LOG_WARNING, format, v...)
}

func (l *Logger) Debug(v ...interface{}) {
	l.log(LOG_DEBUG, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.logf(LOG_DEBUG, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.log(LOG_INFO, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.logf(LOG_INFO, format, v...)
}

func StringToLogLevel(level string) LogLevel {
	switch level {
	case "fatal":
		return LOG_LEVEL_FATAL
	case "error":
		return LOG_LEVEL_ERROR
	case "warn":
		return LOG_LEVEL_WARN
	case "warning":
		return LOG_LEVEL_WARN
	case "debug":
		return LOG_LEVEL_DEBUG
	case "info":
		return LOG_LEVEL_INFO
	}
	return LOG_LEVEL_ALL
}

func LogTypeToString(t LogType) string {
	switch t {
	case LOG_FATAL:
		return "fatal"
	case LOG_ERROR:
		return "error"
	case LOG_WARNING:
		return "warning"
	case LOG_DEBUG:
		return "debug"
	case LOG_INFO:
		return "info"
	}
	return "unknown"
}
func New() *Logger {
	return NewLogger(os.Stderr, "", Ldate|Ltime|Lshortfile)
}

func NewLogger(w io.Writer, prefix string, flags int) *Logger {
	return &Logger{_log: log.New(w, prefix, flags), level: LOG_LEVEL_ALL, TimeFormat: FORMAT_TIME_DAY, SuffixName: ".log"}
}
