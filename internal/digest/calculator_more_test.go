package digest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCalculator_CalculateFiles_Parallel(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(dir, fName(i)), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files := []string{filepath.Join(dir, fName(0)), filepath.Join(dir, fName(1)), filepath.Join(dir, fName(2))}
	c := NewCalculator()
	res, err := c.CalculateFiles(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if res.Digest == "" || len(res.Files) != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func fName(i int) string { return "file" + string(rune('a'+i)) + ".txt" }
