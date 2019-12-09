// Copyright 2019 The Nym Mixnet Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
/*
	Package logger provides the functionalities for log actions of Nym entities.
	Code was adapted from the nymtech/nym/logger package.
	However backend log library was replaced from gopkg.in/op/go-log.v1 to github.com/sirupsen/logrus
	due to being more stable and supported.
*/

package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	ColorBlack = iota + 30
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

type customFormatter struct {
	log.TextFormatter
	module string
}

func (f *customFormatter) Format(entry *log.Entry) ([]byte, error) {
	// this whole mess of dealing with ansi colour codes is required if you
	// want the coloured output otherwise you will lose colours in the log levels
	var levelColor int
	switch entry.Level {
	case log.TraceLevel:
		levelColor = ColorCyan
	case log.DebugLevel:
		levelColor = ColorBlue
	case log.InfoLevel:
		levelColor = ColorGreen
	case log.WarnLevel:
		levelColor = ColorYellow
	case log.ErrorLevel:
		levelColor = ColorRed
	case log.FatalLevel:
		levelColor = ColorMagenta
	case log.PanicLevel:
		levelColor = ColorMagenta
	default:
		levelColor = ColorGreen
	}

	levelText := strings.ToUpper(entry.Level.String())
	levelText = levelText[0:4]
	callerString := f.module
	if entry.HasCaller() {
		fullFuncName := entry.Caller.Func.Name()
		i := strings.LastIndex(fullFuncName, ".")

		callerString += fmt.Sprintf("/%s", fullFuncName[i+1:])
	}

	formattedTime := fmt.Sprintf("\x1b[%dm[%s]\x1b[0m",
		ColorWhite,
		entry.Time.Format(f.TimestampFormat),
	)
	return []byte(fmt.Sprintf("%s\x1b[%dm %s ▶ %s \x1b[0m- %s\n",
		formattedTime,
		levelColor,
		callerString,
		levelText,
		entry.Message,
	)), nil
}

// Hold all necessary data to create module-specific loggers
type Logger struct {
	logOut io.Writer
	level  log.Level
}

// GetLogger returns a per-module logger that writes to the backend.
func (l *Logger) GetLogger(module string) *log.Logger {
	// we need to create a new formatter to include module name in the output
	// we could get rid of it, but personally I found it useful
	formatter := &customFormatter{
		TextFormatter: log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05.000",
		},
		module: module,
	}

	baseLogger := log.New()
	baseLogger.Formatter = formatter
	baseLogger.Out = l.logOut
	baseLogger.Level = l.level
	baseLogger.ReportCaller = true

	return baseLogger
}

// New returns new instance of logger
func New(f string, level string, disable bool) (*Logger, error) {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		return nil, err
	}

	var logOut io.Writer
	if disable {
		logOut = ioutil.Discard
	} else if f == "" {
		logOut = os.Stdout
	} else {
		const fileMode = 0600

		var err error
		flags := os.O_CREATE | os.O_APPEND | os.O_WRONLY
		logOut, err = os.OpenFile(f, flags, fileMode)
		if err != nil {
			return nil, fmt.Errorf("logger: failed to create log file: %v", err)
		}
	}

	return &Logger{
		logOut: logOut,
		level:  lvl,
	}, nil
}
