package restapi

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SplitPart struct {
	Lines     []string      // All lines: commands (renumbered) and comments/markers (as-is)
	WaitAfter time.Duration // Duration to wait after this part, if any
}

func parseWaitDuration(line string) (time.Duration, error) {
	re := regexp.MustCompile(`wait\s+(\d{2}):(\d{2}):(\d{2}):(\d{3})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 5 {
		return 0, fmt.Errorf("invalid wait format: %s", line)
	}
	h, _ := strconv.Atoi(matches[1])
	m, _ := strconv.Atoi(matches[2])
	s, _ := strconv.Atoi(matches[3])
	ms, _ := strconv.Atoi(matches[4])
	totalMs := (((h*60+m)*60 + s) * 1000) + ms
	return time.Duration(totalMs) * time.Millisecond, nil
}

func splitAndRenumberFile(content string) ([]SplitPart, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var parts [][]string
	var waits []time.Duration

	currentPart := []string{}
	waitPattern := regexp.MustCompile(`^(\d{3}\s+)?wait\s+\d{2}:\d{2}:\d{2}:\d{3}`)
	commandPattern := regexp.MustCompile(`^(\d{3})\s+(.*)$`)
	minWait := time.Minute

	for scanner.Scan() {
		line := scanner.Text()
		// Split on wait lines > 1min
		if waitPattern.MatchString(line) {
			d, err := parseWaitDuration(line)
			if err != nil {
				return nil, err
			}
			if d >= minWait {
				parts = append(parts, currentPart)
				waits = append(waits, d)
				currentPart = []string{}
				continue // do not add this wait line to part
			}
			// If wait < 1min, keep line (it might be a comment or allowed)
		}
		currentPart = append(currentPart, line)
	}
	// Add final part
	parts = append(parts, currentPart)
	waits = append(waits, 0)

	var result []SplitPart
	for _, origLines := range parts {
		var outLines []string
		newCmdNum := 1
		for _, line := range origLines {
			if m := commandPattern.FindStringSubmatch(line); m != nil {
				// Renumber command
				outLines = append(outLines, fmt.Sprintf("%03d %s", newCmdNum, m[2]))
				newCmdNum++
			} else {
				// Keep as-is (comments, markers, etc.)
				outLines = append(outLines, line)
			}
		}
		// Append the end line
		outLines = append(outLines, fmt.Sprintf("%03d end", newCmdNum))
		result = append(result, SplitPart{
			Lines: outLines,
			// WaitAfter set below
		})
	}
	// Attach waits to result (except after last)
	for i := 0; i < len(result); i++ {
		if i < len(waits) {
			result[i].WaitAfter = waits[i]
		}
	}
	return result, nil
}
