package serviceutil

import (
	"encoding/json"
	"fmt"
	"github.com/Clever/wag/logging/wagclientlogger"
)

// A lightweight logger which prints the wag client logs to standard out.
type FmtPrinlnLogger struct{}

func (FmtPrinlnLogger) Log(level wagclientlogger.LogLevel, title string, data map[string]interface{}) {
	bs, _ := json.Marshal(data)
	fmt.Printf("%s - %s %s\n", levelString(level), title, string(bs))
}

func levelString(l wagclientlogger.LogLevel) string {
	switch l {
	case 0:
		return "TRACE"
	case 1:
		return "DEBUG"
	case 2:
		return "INFO"
	case 3:
		return "WARNING"
	case 4:
		return "ERROR"
	case 5:
		return "CRITICAL"
	case 6:
		return "FROM_ENV"
	default:
		return "INFO"
	}
}
