package global

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
)

var HTTP_CLIENT = http.Client{}

var JSON = jsoniter.ConfigCompatibleWithStandardLibrary

var Base64 = base64.StdEncoding

var verboseLogging atomic.Bool

func SetVerboseLogging(enabled bool) {
	verboseLogging.Store(enabled)
}

func VerboseLogging() bool {
	return verboseLogging.Load()
}

func Warningf(format string, args ...interface{}) {
	if !VerboseLogging() {
		return
	}
	fmt.Printf(format, args...)
}

func Warningln(args ...interface{}) {
	if !VerboseLogging() {
		return
	}
	fmt.Println(args...)
}
