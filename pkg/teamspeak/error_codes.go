package teamspeak

import (
	_ "embed"
	"encoding/csv"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

//go:embed docs/Errors.csv
var errorsCSV string

type ErrorCodeInfo struct {
	Name        string
	Description string
}

var (
	errorCodesOnce sync.Once
	errorCodeMap   map[int]ErrorCodeInfo
)

func LookupErrorCode(code int) (ErrorCodeInfo, bool) {
	errorCodesOnce.Do(loadErrorCodes)
	info, ok := errorCodeMap[code]
	return info, ok
}

func loadErrorCodes() {
	errorCodeMap = make(map[int]ErrorCodeInfo)

	r := csv.NewReader(strings.NewReader(errorsCSV))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		slog.Debug("failed to parse teamspeak error catalog", "error", err)
		return
	}

	for _, rec := range records {
		if len(rec) < 3 {
			continue
		}
		name := strings.TrimSpace(rec[0])
		description := strings.TrimSpace(rec[1])
		hexCode := strings.TrimSpace(rec[2])
		if name == "" || hexCode == "" {
			continue
		}

		code64, err := strconv.ParseInt(strings.TrimPrefix(hexCode, "0x"), 16, 32)
		if err != nil {
			continue
		}

		errorCodeMap[int(code64)] = ErrorCodeInfo{
			Name:        name,
			Description: description,
		}
	}
}
