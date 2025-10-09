package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

func WriteFile(name string, content any) {
	if L.Level() != zap.DebugLevel {
		return
	}
	baseName := strings.TrimSuffix(name, filepath.Ext(name))
	rawData, err := extractRawData(content)
	if err != nil {
		L.Errorf("failed to extract data: %v", err)
		return
	}
	if len(rawData) == 0 {
		L.Warn("no data to write for file: " + name)
		return
	}
	filePath := determineFilePath(baseName, rawData)
	if err := writeToFile(filePath, rawData); err != nil {
		L.Errorf("failed to write file %s: %v", filePath, err)
		return
	}
	L.Debug("saved file " + filePath)
}

func extractRawData(content any) ([]byte, error) {
	switch v := content.(type) {
	case *http.Response:
		return extractFromHTTPResponse(v)
	case json.RawMessage:
		return []byte(v), nil
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	case fmt.Stringer:
		return []byte(v.String()), nil
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}
}

func extractFromHTTPResponse(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return nil, nil
	}
	if seeker, ok := resp.Body.(io.Seeker); ok {
		currentPos, _ := seeker.Seek(0, io.SeekCurrent)
		data, err := io.ReadAll(resp.Body)
		seeker.Seek(currentPos, io.SeekStart)
		return data, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}

func isJSONData(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	firstChar := data[0]
	if firstChar != '{' && firstChar != '[' {
		return false
	}

	var dummy any
	return json.Unmarshal(data, &dummy) == nil
}

func formatJSON(data []byte) []byte {
	var jsonObj any
	if err := sonic.ConfigFastest.Unmarshal(data, &jsonObj); err == nil {
		if prettyJSON, err := sonic.ConfigFastest.MarshalIndent(jsonObj, "", "  "); err == nil {
			return prettyJSON
		}
	}
	var indented bytes.Buffer
	if json.Indent(&indented, data, "", "  ") == nil {
		return indented.Bytes()
	}
	return data
}

func determineFilePath(baseName string, data []byte) string {
	var filename string
	if isJSONData(data) {
		filename = baseName + ".json"
	} else {
		filename = baseName + ".txt"
	}
	return filepath.Join("logs", filename)
}

func writeToFile(filePath string, data []byte) error {
	if strings.HasSuffix(filePath, ".json") {
		data = formatJSON(data)
	}
	return os.WriteFile(filePath, data, 0644)
}
