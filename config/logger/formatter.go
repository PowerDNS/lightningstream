// Package logger implements a custom logger that prefixes log messages with the database name.
package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// NamespaceFormatter is a logrus formatter that moves the 'namespace' field to a log prefix
// for nicer formatted text output.
type NamespaceFormatter struct {
	Parent logrus.Formatter
}

// Format implements logrus.Formatter
func (f *NamespaceFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	db, exists := entry.Data["db"]
	if exists {
		ns := db.(string)
		newFields := logrus.Fields{}
		for k, v := range entry.Data {
			if k != "db" {
				newFields[k] = v
			}
		}
		entry.Data = newFields
		entry.Message = fmt.Sprintf("[%-14s] %s", ns, entry.Message)
	}
	return f.Parent.Format(entry)
}
