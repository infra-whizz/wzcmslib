package nanoutils

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"os"
)

func GetTextLogger(level logrus.Level, out *os.File) *logrus.Logger {
	formatter := new(nested.Formatter)
	formatter.HideKeys = true
	formatter.FieldsOrder = []string{"component", "category"}
	formatter.ShowFullLevel = true

	logger := logrus.New()

	if out == nil {
		logger.Out = os.Stderr
		formatter.NoColors = false
		formatter.NoFieldsColors = false
	} else {
		logger.Out = out
		formatter.NoColors = true
		formatter.NoFieldsColors = true
	}

	logger.Level = level
	logger.SetFormatter(formatter)

	return logger
}
