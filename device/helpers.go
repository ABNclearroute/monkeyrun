package device

import (
	"bufio"
	"context"
	"io"
	"os"
)

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func streamLines(ctx context.Context, r io.Reader, logCh chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case logCh <- scanner.Text():
		}
	}
}
