package segmented

import (
	"fmt"
	"io"
	"os"
)

func writeSegments(
	writer io.Writer,
	initSegment string,
	segments []string,
) error {
	if len(segments) == 0 {
		return fmt.Errorf("no segments to merge")
	}
	if initSegment != "" {
		return mergeSegmentsWithInit(writer, initSegment, segments)
	}
	return mergeSegments(writer, segments)
}

func mergeSegmentsWithInit(writer io.Writer, initSegment string, segments []string) error {
	data, err := os.ReadFile(initSegment)
	if err != nil {
		return fmt.Errorf("failed to read init segment %s: %w", initSegment, err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write init segment %s: %w", initSegment, err)
	}
	return mergeSegments(writer, segments)
}

func mergeSegments(writer io.Writer, segments []string) error {
	for _, segment := range segments {
		data, err := os.ReadFile(segment)
		if err != nil {
			return fmt.Errorf("failed to read segment %s: %w", segment, err)
		}
		_, err = writer.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write segment %s: %w", segment, err)
		}
	}
	return nil
}
