package nanocms_results

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"

	wzlib_logger "github.com/infra-whizz/wzlib/logger"

	nanocms_runners "github.com/infra-whizz/wzcmslib/nanorunners"
	"github.com/sirupsen/logrus"
)

type ResultLogEntry struct {
	Level   logrus.Level
	Message string
	wzlib_logger.WzLogger
}

func (rle *ResultLogEntry) Log() {
	switch rle.Level {
	case logrus.WarnLevel:
		rle.GetLogger().Warn(rle.Message)
	case logrus.ErrorLevel:
		rle.GetLogger().Error(rle.Message)
	case logrus.PanicLevel:
		rle.GetLogger().Panic(rle.Message)
	case logrus.DebugLevel:
		rle.GetLogger().Debug(rle.Message)
	case logrus.TraceLevel:
		rle.GetLogger().Trace(rle.Message)
	default:
		rle.GetLogger().Info(rle.Message)
	}
}

type ResultsToLog struct {
	results *nanocms_runners.RunnerResponse
}

func NewResultsToLog() *ResultsToLog {
	rtl := new(ResultsToLog)
	return rtl
}

func (rtl *ResultsToLog) LoadResults(results *nanocms_runners.RunnerResponse) *ResultsToLog {
	rtl.results = results
	return rtl
}

// ToLog parses the whole thing and greates a series of log entries with the log levels in it.
func (rtl *ResultsToLog) ToLog() []*ResultLogEntry {
	messages := make([]*ResultLogEntry, 0)
	if rtl.results != nil {
		messages = append(messages, &ResultLogEntry{
			Level:   logrus.InfoLevel,
			Message: fmt.Sprintf("Summary of the state: %s", rtl.results.Description),
		})
		for _, group := range rtl.results.Groups {
			var blockLogEntry *ResultLogEntry
			if group.Errcode > nanocms_runners.ERR_OK {
				blockLogEntry = &ResultLogEntry{
					Level:   logrus.ErrorLevel,
					Message: fmt.Sprintf("Error processing block '%s' (code: %d): %s", group.GroupId, group.Errcode, group.Errmsg),
				}
			} else {
				blockLogEntry = &ResultLogEntry{
					Level:   logrus.DebugLevel,
					Message: fmt.Sprintf("Block '%s' executed successfully", group.GroupId),
				}
			}
			messages = append(messages, blockLogEntry)

			// Get tasks
			for _, resp := range group.Response {
				var respLogEntry *ResultLogEntry
				if resp.Errcode > nanocms_runners.ERR_OK {
					respLogEntry = &ResultLogEntry{
						Level:   logrus.ErrorLevel,
						Message: fmt.Sprintf("Error calling module '%s' (code: %d): %s", resp.Module, resp.Errcode, resp.Errmsg),
					}
				} else {
					respLogEntry = &ResultLogEntry{
						Level:   logrus.DebugLevel,
						Message: fmt.Sprintf("Module '%s' call finished successfully", resp.Module),
					}
				}
				messages = append(messages, respLogEntry)

				// Module results
				for _, mres := range resp.Response {
					if mres.Host != "localhost" {
						messages = append(messages, &ResultLogEntry{
							Level:   logrus.DebugLevel,
							Message: fmt.Sprintf("Host: %s", mres.Host),
						})
					}
					// module responses
					for moduleId, moduleCallResult := range mres.Response {
						if moduleCallResult.Errcode > 0 {
							messages = append(messages, &ResultLogEntry{
								Level:   logrus.ErrorLevel,
								Message: fmt.Sprintf("Module '%s' failed (%d): %s", moduleId, moduleCallResult.Errcode, moduleCallResult.Errmsg),
							})
							if moduleCallResult.Stderr != "" {
								messages = append(messages, &ResultLogEntry{
									Level:   logrus.DebugLevel,
									Message: fmt.Sprintf("%s - STDERR: %s", moduleId, moduleCallResult.Stderr),
								})
							}
							if moduleCallResult.Stdout != "" {
								messages = append(messages, &ResultLogEntry{
									Level:   logrus.DebugLevel,
									Message: fmt.Sprintf("%s - STDOUT: %s", moduleId, moduleCallResult.Stdout),
								})
							}
						} else {
							messages = append(messages, &ResultLogEntry{
								Level:   logrus.InfoLevel,
								Message: fmt.Sprintf("Module '%s' succeed", moduleId),
							})
							if moduleCallResult.Stdout != "" {
								messages = append(messages, &ResultLogEntry{
									Level:   logrus.DebugLevel,
									Message: fmt.Sprintf("Module '%s' output:\n---\n%s\n---", moduleId, moduleCallResult.Stdout),
								})
							}
							// Add module results
							var level logrus.Level

							failed, fex := moduleCallResult.Json["failed"]
							changed, cex := moduleCallResult.Json["changed"]
							if fex && failed.(bool) {
								level = logrus.ErrorLevel
							} else if cex && !changed.(bool) {
								level = logrus.WarnLevel
							} else {
								level = logrus.InfoLevel
							}
							messages = append(messages, &ResultLogEntry{
								Level: level,
								Message: fmt.Sprintf("%s - changed: %v, failed: %v",
									moduleId, moduleCallResult.Json["changed"],
									moduleCallResult.Json["failed"]),
							})

							// Add module JSON output introspection
							messages = append(messages, &ResultLogEntry{
								Level: logrus.DebugLevel,
								Message: fmt.Sprintf("%s introspected direct output:\n%s",
									moduleId, spew.Sdump(moduleCallResult.Json)),
							})
						}
					}
				}
			}
		}
	}
	return messages
}
