// Package logger implements a custom logger that prefixes log messages with the database name.
package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// NamespaceFormatter is a logrus formatter that adds the 'db' field to a log prefix
// for nicer formatted text output.
type NamespaceFormatter struct {
	Parent logrus.Formatter
}

// Format implements logrus.Formatter
func (f *NamespaceFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	db, exists := entry.Data["db"]
	if exists {
		ns := db.(string)
		entry.Message = fmt.Sprintf("[%-14s] %s", ns, entry.Message)
	}
	return f.Parent.Format(entry)
}
