package coverage

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func Write(w io.Writer, profile *Profile) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "mode: %s\n", profile.Mode)
	for _, e := range profile.Entries {
		fmt.Fprintln(bw, e.Raw)
	}
	return bw.Flush()
}

func WriteFile(path string, profile *Profile) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := Write(f, profile)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
