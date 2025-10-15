package chunked

import (
	"fmt"
	"strconv"
	"strings"
)

func parseContentRange(contentRange string) (int64, error) {
	// expected format: "bytes START-END/TOTAL" or "bytes */TOTAL"
	if !strings.HasPrefix(contentRange, "bytes ") {
		return 0, fmt.Errorf("invalid content-range format: %s", contentRange)
	}

	parts := strings.Split(contentRange, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid content-range format, missing '/': %s", contentRange)
	}

	totalStr := strings.TrimSpace(parts[1])
	total, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse total size from content-range '%s': %w", contentRange, err)
	}

	return total, nil
}
