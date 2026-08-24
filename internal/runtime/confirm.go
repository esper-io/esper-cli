package runtime

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func Confirm(reader io.Reader, writer io.Writer, target string, count int, yes bool) (bool, error) {
	if yes {
		return true, nil
	}
	if count < 1 {
		count = 1
	}
	if _, err := fmt.Fprintf(writer, "Confirm operation on %s (%d target(s))? [y/N] ", target, count); err != nil {
		return false, fmt.Errorf("write confirmation prompt: %w", err)
	}
	answer, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && len(answer) == 0 {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
