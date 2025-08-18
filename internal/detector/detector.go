// Package detector provides project detection and dependency analysis.
package detector

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectDetector analyzes a project and determines its requirements
type ProjectDetector struct {
	Root         string
	Type         ProjectType
	Languages    []Language
	Framework    string
	Version      string
	Dependencies Dependencies
	Metadata     map[string]interface{}
}

// ProjectType represents the primary project type
type ProjectType string

const (
	TypePHPLaravel    ProjectType = "php-laravel"
	TypePHPSymfony    ProjectType = "php-symfony"
	TypePHPGeneric    ProjectType = "php"
	TypeNodeNext      ProjectType = "node-next"
	TypeNodeNuxt      ProjectType = "node-nuxt"
	TypeNodeGeneric   ProjectType = "node"
	TypePythonDjango  ProjectType = "python-django"
	TypePythonFlask   ProjectType = "python-flask"
	TypePythonGeneric ProjectType = "python"
	TypeGoModule      ProjectType = "go"
	TypeRubyRails     ProjectType = "ruby-rails"
	TypeRubyGeneric   ProjectType = "ruby"
	TypeStatic        ProjectType = "static"
	TypeUnknown       ProjectType = "unknown"
)

// Language represents a programming language with version
type Language struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Primary bool   `json:"primary"`
}

// Dependencies holds all detected dependencies
type Dependencies struct {
	PHP    PHPDependencies    `json:"php,omitempty"`
	Node   NodeDependencies   `json:"node,omitempty"`
	Python PythonDependencies `json:"python,omitempty"`
	System []string           `json:"system,omitempty"` // Alpine packages
}

// PHPDependencies for PHP projects
type PHPDependencies struct {
	Version     string            `json:"version"` // 8.3, 8.2, etc.
	Extensions  []string          `json:"extensions"`
	Composer    bool              `json:"composer"`
	ComposerVer string            `json:"composer_version"`
	IniSettings map[string]string `json:"ini_settings"`
}

// NodeDependencies for Node projects
type NodeDependencies struct {
	Version        string   `json:"version"`
	PackageManager string   `json:"package_manager"` // npm, yarn, pnpm
	GlobalPackages []string `json:"global_packages"`
	BuildTools     bool     `json:"build_tools"`
}

// PythonDependencies for Python projects
type PythonDependencies struct {
	Version    string   `json:"version"`
	UsesPoetry bool     `json:"uses_poetry"`
	UsesPipenv bool     `json:"uses_pipenv"`
	UsesVenv   bool     `json:"uses_venv"`
	SystemDeps []string `json:"system_deps"`
}

// NewProjectDetector creates a detector for the current directory
func NewProjectDetector(root string) *ProjectDetector {
	if root == "" {
		root, _ = os.Getwd()
	}
	return &ProjectDetector{
		Root:     root,
		Type:     TypeUnknown,
		Metadata: make(map[string]interface{}),
	}
}

// Detect analyzes the project and populates all fields
func (pd *ProjectDetector) Detect() error {
	pd.detectProjectType()
	pd.detectLanguages()
	pd.analyzeDependencies()
	pd.detectFrameworkRequirements()
	return nil
}

// DetectPHPExtensions analyzes composer packages to determine required extensions (exported for wrappers)
func (pd *ProjectDetector) DetectPHPExtensions(packages map[string]interface{}) []string {
	return pd.detectPHPExtensions(packages)
}

// ValidatePyProject checks pyproject content for validity
func (pd *ProjectDetector) ValidatePyProject(path string) bool { return pd.validatePyProject(path) }

// CheckFlaskImports checks for flask imports in app.py
func (pd *ProjectDetector) CheckFlaskImports(path string) bool { return pd.checkFlaskImports(path) }

// detectLanguages populates Languages slice with a best-effort guess
func (pd *ProjectDetector) detectLanguages() {
	langs := []Language{}
	if strings.HasPrefix(string(pd.Type), "php") || fileExists(filepath.Join(pd.Root, "composer.json")) {
		langs = append(langs, Language{Name: "php", Primary: strings.HasPrefix(string(pd.Type), "php")})
	}
	if strings.HasPrefix(string(pd.Type), "node") || fileExists(filepath.Join(pd.Root, "package.json")) {
		langs = append(langs, Language{Name: "node", Primary: strings.HasPrefix(string(pd.Type), "node")})
	}
	if strings.HasPrefix(string(pd.Type), "python") || fileExists(filepath.Join(pd.Root, "requirements.txt")) || fileExists(filepath.Join(pd.Root, "pyproject.toml")) {
		langs = append(langs, Language{Name: "python", Primary: strings.HasPrefix(string(pd.Type), "python")})
	}
	pd.Languages = langs
}

// detectProjectType identifies the primary project type
func (pd *ProjectDetector) detectProjectType() {
	checks := []struct {
		file     string
		projType ProjectType
		validate func(string) bool
	}{
		// PHP
		{"composer.json", TypePHPGeneric, pd.validateComposerJSON},
		{"artisan", TypePHPLaravel, nil},
		{"symfony.lock", TypePHPSymfony, nil},

		// Node
		{"package.json", TypeNodeGeneric, pd.validatePackageJSON},
		{"next.config.js", TypeNodeNext, nil},
		{"nuxt.config.js", TypeNodeNuxt, nil},

		// Python
		{"requirements.txt", TypePythonGeneric, nil},
		{"pyproject.toml", TypePythonGeneric, pd.validatePyProject},
		{"manage.py", TypePythonDjango, nil},
		{"app.py", TypePythonFlask, pd.checkFlaskImports},

		// Go
		{"go.mod", TypeGoModule, nil},

		// Ruby
		{"Gemfile", TypeRubyGeneric, nil},
		{"config.ru", TypeRubyRails, nil},
	}

	for _, check := range checks {
		path := filepath.Join(pd.Root, check.file)
		if _, err := os.Stat(path); err == nil {
			if check.validate != nil {
				prev := pd.Type
				if check.validate(path) {
					// If validator didn't refine the type, fall back to provided type
					if pd.Type == prev || pd.Type == TypeUnknown {
						pd.Type = check.projType
					}
					break
				}
			} else {
				pd.Type = check.projType
				break
			}
		}
	}

	pd.refineProjectType()
}

// validateComposerJSON checks if composer.json is valid
func (pd *ProjectDetector) validateComposerJSON(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var composer map[string]interface{}
	if json.Unmarshal(data, &composer) != nil {
		return false
	}
	pd.Metadata["composer"] = composer

	// Laravel detection and version
	if require, ok := composer["require"].(map[string]interface{}); ok {
		if v, has := require["laravel/framework"]; has {
			pd.Type = TypePHPLaravel
			pd.Framework = "Laravel"
			if s, ok := v.(string); ok {
				pd.Version = extractVersion(s)
			}
		}
	}
	return true
}

func (pd *ProjectDetector) validatePackageJSON(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg map[string]interface{}
	if json.Unmarshal(data, &pkg) != nil {
		return false
	}
	pd.Metadata["package.json"] = pkg
	// Detect Next/Nuxt if deps present
	if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
		if _, ok := deps["next"]; ok {
			pd.Type = TypeNodeNext
		}
		if _, ok := deps["nuxt"]; ok {
			pd.Type = TypeNodeNuxt
		}
	}
	if devDeps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		if _, ok := devDeps["next"]; ok {
			pd.Type = TypeNodeNext
		}
		if _, ok := devDeps["nuxt"]; ok {
			pd.Type = TypeNodeNuxt
		}
	}
	return true
}

func (pd *ProjectDetector) validatePyProject(path string) bool {
	// Light validation: file exists and has [project] or [tool.poetry]
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	s := strings.ToLower(string(b))
	return strings.Contains(s, "[project]") || strings.Contains(s, "[tool.poetry]")
}

func (pd *ProjectDetector) checkFlaskImports(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, "from flask") || strings.Contains(line, "import flask") {
			return true
		}
	}
	return false
}

// refineProjectType refines generic types to specific frameworks if signals found
func (pd *ProjectDetector) refineProjectType() {
	// Python: prefer framework-specific when signals found
	if pd.Type == TypePythonGeneric {
		if fileExists(filepath.Join(pd.Root, "manage.py")) {
			pd.Type = TypePythonDjango
		} else if fileExists(filepath.Join(pd.Root, "app.py")) {
			// If app.py references Flask, validator sets; otherwise leave generic
		}
	}
	// Node: refine generic to framework by config files
	if pd.Type == TypeNodeGeneric {
		if fileExists(filepath.Join(pd.Root, "next.config.js")) || fileExists(filepath.Join(pd.Root, "next.config.ts")) {
			pd.Type = TypeNodeNext
		} else if fileExists(filepath.Join(pd.Root, "nuxt.config.js")) || fileExists(filepath.Join(pd.Root, "nuxt.config.ts")) {
			pd.Type = TypeNodeNuxt
		}
	}
}

// analyzeDependencies performs deep analysis of project dependencies
func (pd *ProjectDetector) analyzeDependencies() {
	switch {
	case strings.HasPrefix(string(pd.Type), "php"):
		pd.analyzePHPDependencies()
	case strings.HasPrefix(string(pd.Type), "node"):
		pd.analyzeNodeDependencies()
	case strings.HasPrefix(string(pd.Type), "python"):
		pd.analyzePythonDependencies()
	}

	// Mixed stacks: analyze secondary language deps when present
	if fileExists(filepath.Join(pd.Root, "package.json")) && pd.Dependencies.Node.Version == "" {
		pd.analyzeNodeDependencies()
	}
	if fileExists(filepath.Join(pd.Root, "composer.json")) && pd.Dependencies.PHP.Version == "" {
		pd.analyzePHPDependencies()
	}

	pd.detectSecondaryLanguages()
}

// analyzePHPDependencies extracts PHP requirements
func (pd *ProjectDetector) analyzePHPDependencies() {
	deps := PHPDependencies{
		Version:    "8.3",
		Composer:   true,
		Extensions: []string{},
		IniSettings: map[string]string{
			"memory_limit":        "256M",
			"max_execution_time":  "300",
			"post_max_size":       "100M",
			"upload_max_filesize": "100M",
		},
	}
	if composer, ok := pd.Metadata["composer"].(map[string]interface{}); ok {
		if require, ok := composer["require"].(map[string]interface{}); ok {
			if phpReq, ok := require["php"].(string); ok {
				deps.Version = ExtractPHPVersion(phpReq)
			}
			deps.Extensions = pd.detectPHPExtensions(require)
		}
	}
	// Add Laravel defaults
	if pd.Type == TypePHPLaravel {
		laravelExts := []string{"bcmath", "ctype", "curl", "dom", "fileinfo", "json", "mbstring", "openssl", "pdo", "pdo_mysql", "tokenizer", "xml", "zip"}
		deps.Extensions = UniqueStrings(append(deps.Extensions, laravelExts...))
	}
	// Scan source for usages (best-effort)
	deps.Extensions = UniqueStrings(append(deps.Extensions, pd.scanPHPFiles()...))
	pd.Dependencies.PHP = deps
}

// analyzeNodeDependencies extracts Node requirements
func (pd *ProjectDetector) analyzeNodeDependencies() {
	nd := NodeDependencies{Version: "20", PackageManager: "npm"}
	if pkg, ok := pd.Metadata["package.json"].(map[string]interface{}); ok {
		if engines, ok := pkg["engines"].(map[string]interface{}); ok {
			if v, ok := engines["node"].(string); ok {
				nd.Version = ExtractNodeVersion(v)
			}
		}
		// PM detection via lockfiles
		if fileExists(filepath.Join(pd.Root, "pnpm-lock.yaml")) {
			nd.PackageManager = "pnpm"
		} else if fileExists(filepath.Join(pd.Root, "yarn.lock")) {
			nd.PackageManager = "yarn"
		} else {
			nd.PackageManager = "npm"
		}
		// Build script present?
		if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
			_, nd.BuildTools = scripts["build"]
		}
	}
	pd.Dependencies.Node = nd
}

// analyzePythonDependencies extracts Python requirements
func (pd *ProjectDetector) analyzePythonDependencies() {
	pd.Dependencies.Python = PythonDependencies{Version: "3.11"}
}

// detectSecondaryLanguages detects mixed stacks (e.g., PHP + Node)
func (pd *ProjectDetector) detectSecondaryLanguages() {
	// Simple heuristic: if package.json present, note Node
	if fileExists(filepath.Join(pd.Root, "package.json")) {
		pd.Languages = append(pd.Languages, Language{Name: "node", Primary: strings.HasPrefix(string(pd.Type), "node")})
	}
	if fileExists(filepath.Join(pd.Root, "composer.json")) {
		pd.Languages = append(pd.Languages, Language{Name: "php", Primary: strings.HasPrefix(string(pd.Type), "php")})
	}
}

// detectFrameworkRequirements placeholder for future
func (pd *ProjectDetector) detectFrameworkRequirements() {}

// detectPHPExtensions analyzes composer packages to determine required extensions
func (pd *ProjectDetector) detectPHPExtensions(packages map[string]interface{}) []string {
	extensions := []string{}
	packageExtMap := map[string][]string{
		"laravel/framework":  {"pdo_mysql", "pdo"},
		"guzzlehttp":         {"curl"},
		"intervention/image": {"gd", "imagick"},
		"predis":             {"redis"},
		"mongodb":            {"mongodb"},
		"postgresql":         {"pdo_pgsql", "pgsql"},
		"mysql":              {"pdo_mysql", "mysqli"},
		"sqlite":             {"pdo_sqlite", "sqlite3"},
		"gd":                 {"gd"},
		"imagick":            {"imagick"},
		"ldap":               {"ldap"},
		"soap":               {"soap"},
		"xlsx":               {"zip", "xml"},
		"pdf":                {"gd"},
		"maatwebsite/excel":  {"zip", "xml", "gd"},
		"doctrine/dbal":      {"pdo"},
	}
	for pkg := range packages {
		pkgLower := strings.ToLower(pkg)
		for pattern, exts := range packageExtMap {
			if strings.Contains(pkgLower, pattern) {
				extensions = append(extensions, exts...)
			}
		}
	}
	return UniqueStrings(extensions)
}

// scanPHPFiles looks for function calls that indicate extension usage
func (pd *ProjectDetector) scanPHPFiles() []string {
	extensions := []string{}
	functionExtMap := map[string]string{
		`imagecreate`:    "gd",
		`curl_init`:      "curl",
		`redis\s*\(`:     "redis",
		`mysqli_connect`: "mysqli",
		`pg_connect`:     "pgsql",
		`ldap_connect`:   "ldap",
		`soap.*client`:   "soap",
		`simplexml_load`: "xml",
		`json_encode`:    "json",
		`mb_strlen`:      "mbstring",
		`iconv`:          "iconv",
		`openssl_`:       "openssl",
		`sodium_`:        "sodium",
		`bcadd`:          "bcmath",
		`intl`:           "intl",
	}
	_ = filepath.Walk(pd.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, "vendor/") {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".php") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			contentStr := strings.ToLower(string(content))
			for pattern, ext := range functionExtMap {
				matched, _ := regexp.MatchString(pattern, contentStr)
				if matched {
					extensions = append(extensions, ext)
				}
			}
		}
		return nil
	})
	return UniqueStrings(extensions)
}

// Helpers
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func UniqueStrings(in []string) []string {
	m := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func ContainsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func extractVersion(req string) string {
	// Extract numeric prefix (e.g., ^10.0 -> 10.0)
	for i := 0; i < len(req); i++ {
		if req[i] >= '0' && req[i] <= '9' {
			return strings.TrimLeft(req[i:], "v")
		}
	}
	return req
}

func ExtractPHPVersion(req string) string {
	// Simplify to major.minor if present
	v := extractVersion(req)
	// trim constraints like >=, <=, ^, ~
	v = strings.TrimLeft(v, "=<>^~ ")
	// Default reasonable fallback
	if v == "" {
		return "8.3"
	}
	return v
}

func ExtractNodeVersion(req string) string {
	v := extractVersion(req)
	v = strings.TrimSpace(v)
	if v == "" {
		return "20"
	}
	// keep only major if form is like 20.x
	if i := strings.IndexByte(v, '.'); i > 0 {
		return v[:i]
	}
	return v
}
