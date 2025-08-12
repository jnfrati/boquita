package logger

import (
	"flag"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var Global = zerolog.New(os.Stderr).With().Timestamp().Logger()

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	debug := flag.Bool("debug", false, "sets log level to debug")
	logOutput := flag.String("log-output", "stderr", "sets log output (stderr, stdout, file)")
	logFile := flag.String("log-file", "", "sets log file path (if log-output is file)")
	flag.Parse()

	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	zerolog.ErrorStackMarshaler = func(err error) interface{} {
		if err == nil {
			return nil
		}
		type stackTracer interface {
			StackTrace() errors.StackTrace
		}
		if st, ok := err.(stackTracer); ok {
			return st.StackTrace()
		}
		return nil
	}

	if logOutput == nil || *logOutput == "" {
		v := "stdout"
		logOutput = &v
	}

	switch *logOutput {
	case "stderr":
		Global = Global.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	case "stdout":
		Global = Global.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	case "file":
		if *logFile == "" {
			Global = Global.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		} else {
			file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				Global = Global.Output(zerolog.ConsoleWriter{Out: os.Stderr})
			} else {
				Global = Global.Output(file)
			}
		}
	default:
		Global = Global.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		Global.Error().Msgf("Unknown log output: %s, defaulting to stderr", *logOutput)
	}
}
