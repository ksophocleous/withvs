package main

import (
	"github.com/Sirupsen/logrus"
	"bytes"
	"strings"
	"fmt"
	"sort"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 36
	gray    = 35
)

type customTextFormatter struct {
	NoColor bool
}

func (f *customTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	b := &bytes.Buffer{}

	levelText := strings.ToUpper(entry.Level.String())[0:4]
	colorStart := "[colorCode]"
	colorEnd := "[colorCode]"
	levelColor := blue

	if f.NoColor == false {
		colorTemplate := "\x1b[%dm"
		colorEnd = "\x1b[0m"
		switch entry.Level {
		case logrus.DebugLevel:
			levelColor = gray
		case logrus.WarnLevel:
			levelColor = yellow
		case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
			levelColor = red
		}
		colorStart = fmt.Sprintf(colorTemplate, levelColor)
	}

	fmt.Fprintf(b, "%s%s%s[%s] %-40s", colorStart, levelText, colorEnd, entry.Time.Format(logrus.DefaultTimestampFormat), entry.Message)
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, "%s%s%s=%+v ", colorStart, k, colorEnd, v)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}