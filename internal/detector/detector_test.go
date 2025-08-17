package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectDetection(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected ProjectType
	}{
		{
			name: "Laravel project",
			files: map[string]string{
				"composer.json": `{"require":{"laravel/framework":"^10.0"}}`,
				"artisan":       "",
			},
			expected: TypePHPLaravel,
		},
		{
			name: "Next.js project",
			files: map[string]string{
				"package.json":   `{"dependencies":{"next":"13.0.0"}}`,
				"next.config.js": "",
			},
			expected: TypeNodeNext,
		},
		{
			name: "Django project",
			files: map[string]string{
				"requirements.txt": "django==4.2.0\n",
				"manage.py":        "",
			},
			expected: TypePythonDjango,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for file, content := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				os.WriteFile(path, []byte(content), 0644)
			}
			detector := NewProjectDetector(tmpDir)
			detector.Detect()
			if detector.Type != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, detector.Type)
			}
		})
	}
}

func TestPHPExtensionDetection(t *testing.T) {
	tests := []struct {
		name     string
		packages map[string]interface{}
		expected []string
	}{
		{
			name: "Laravel with database",
			packages: map[string]interface{}{
				"laravel/framework": "^10.0",
				"predis/predis":     "^2.0",
				"doctrine/dbal":     "^3.0",
			},
			expected: []string{"pdo_mysql", "redis", "pdo"},
		},
		{
			name: "Image manipulation",
			packages: map[string]interface{}{
				"intervention/image": "^2.7",
			},
			expected: []string{"gd", "imagick"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewProjectDetector("")
			exts := detector.DetectPHPExtensions(tt.packages)
			for _, expected := range tt.expected {
				found := false
				for _, got := range exts {
					if got == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected extension %s not found", expected)
				}
			}
		})
	}
}

func BenchmarkProjectDetection(b *testing.B) {
	tmpDir := b.TempDir()
	// Create minimal Laravel signals
	os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(`{"require":{"laravel/framework":"^10.0"}}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "artisan"), []byte(""), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector := NewProjectDetector(tmpDir)
		_ = detector.Detect()
	}
}
