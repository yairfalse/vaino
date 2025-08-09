package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Level int

const (
	LevelCmd Level = iota + 1
	LevelHighInternal
	LevelMidInternal
	LevelLowInternal
	LevelPkg
)

var packageLevels = map[string]Level{
	"cmd":                 LevelCmd,
	"internal/watchers":   LevelHighInternal,
	"internal/workers":    LevelHighInternal,
	"internal/scanner":    LevelHighInternal,
	"internal/collectors": LevelMidInternal,
	"internal/storage":    LevelMidInternal,
	"internal/analysis":   LevelMidInternal,
	"internal/errors":     LevelLowInternal,
	"internal/utils":      LevelLowInternal,
	"pkg":                 LevelPkg,
}

type Violation struct {
	FromFile    string
	FromPackage string
	FromLevel   Level
	ToPackage   string
	ToLevel     Level
}

func getPackageLevel(pkgPath string) Level {
	for prefix, level := range packageLevels {
		if strings.HasPrefix(pkgPath, prefix) {
			return level
		}
	}
	return 0
}

func getPackageFromPath(filePath string) string {
	dir := filepath.Dir(filePath)
	dir = strings.TrimPrefix(dir, "./")
	dir = strings.TrimPrefix(dir, ".")
	return dir
}

func checkFile(filePath string) ([]Violation, error) {
	var violations []Violation

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	fromPackage := getPackageFromPath(filePath)
	fromLevel := getPackageLevel(fromPackage)

	if fromLevel == 0 {
		return violations, nil
	}

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Skip standard library and external packages
		if !strings.Contains(importPath, "/") || strings.Contains(importPath, ".") {
			continue
		}

		// Check if it's an internal import
		if strings.HasPrefix(importPath, "github.com/yairfalse/vaino/") {
			importPath = strings.TrimPrefix(importPath, "github.com/yairfalse/vaino/")
		}

		toLevel := getPackageLevel(importPath)
		if toLevel == 0 {
			continue
		}

		// Check for violations: importing from a higher level
		if toLevel < fromLevel {
			violations = append(violations, Violation{
				FromFile:    filePath,
				FromPackage: fromPackage,
				FromLevel:   fromLevel,
				ToPackage:   importPath,
				ToLevel:     toLevel,
			})
		}
	}

	return violations, nil
}

func walkGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "vendor/") && !strings.Contains(path, ".git/") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func levelName(l Level) string {
	switch l {
	case LevelCmd:
		return "CMD (Level 1)"
	case LevelHighInternal:
		return "HIGH_INTERNAL (Level 2)"
	case LevelMidInternal:
		return "MID_INTERNAL (Level 3)"
	case LevelLowInternal:
		return "LOW_INTERNAL (Level 4)"
	case LevelPkg:
		return "PKG (Level 5)"
	default:
		return "UNKNOWN"
	}
}

func main() {
	fmt.Println("🔍 VAINO Architecture Level Checker")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("Architectural Levels:")
	fmt.Println("  Level 1 (CMD):          cmd/")
	fmt.Println("  Level 2 (HIGH_INTERNAL): internal/watchers, workers, scanner")
	fmt.Println("  Level 3 (MID_INTERNAL):  internal/collectors, storage, analysis")
	fmt.Println("  Level 4 (LOW_INTERNAL):  internal/errors, utils")
	fmt.Println("  Level 5 (PKG):          pkg/")
	fmt.Println()
	fmt.Println("Rule: Each level can only import from same level or lower (higher number)")
	fmt.Println()

	files, err := walkGoFiles(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking files: %v\n", err)
		os.Exit(1)
	}

	var allViolations []Violation
	checkedFiles := 0

	for _, file := range files {
		violations, err := checkFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking %s: %v\n", file, err)
			continue
		}
		allViolations = append(allViolations, violations...)
		checkedFiles++
	}

	fmt.Printf("✅ Checked %d Go files\n", checkedFiles)
	fmt.Println()

	if len(allViolations) == 0 {
		fmt.Println("🎉 No architectural level violations found!")
		os.Exit(0)
	}

	fmt.Printf("❌ Found %d architectural level violations:\n", len(allViolations))
	fmt.Println()

	// Group violations by type
	violationMap := make(map[string][]Violation)
	for _, v := range allViolations {
		key := fmt.Sprintf("%s → %s", levelName(v.FromLevel), levelName(v.ToLevel))
		violationMap[key] = append(violationMap[key], v)
	}

	for violationType, violations := range violationMap {
		fmt.Printf("\n⚠️  %s (%d violations):\n", violationType, len(violations))
		for i, v := range violations {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(violations)-5)
				break
			}
			fmt.Printf("   %s imports %s\n", v.FromPackage, v.ToPackage)
		}
	}

	fmt.Println()
	fmt.Println("📝 To fix these violations:")
	fmt.Println("   1. Move shared code to lower levels (higher numbers)")
	fmt.Println("   2. Use dependency injection to avoid upward dependencies")
	fmt.Println("   3. Consider if the code is in the right package")

	os.Exit(1)
}
