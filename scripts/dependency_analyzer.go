package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PackageInfo struct {
	Path         string
	Files        []string
	Imports      []string
	InternalImports []string
	PkgImports   []string
	ServicesImports []string
	TotalLines   int
	Functions    int
	Types        int
	Interfaces   int
}

type CircularDependency struct {
	Chain []string
	Type  string // "internal", "pkg", "services", "mixed"
}

type ViolationInfo struct {
	File        string
	Lines       int
	Violation   string
	Severity    string
	Category    string
}

func main() {
	fmt.Println("🔍 TradSys Deep Dependency & Structure Analysis")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	packages := make(map[string]*PackageInfo)
	violations := []ViolationInfo{}
	
	// Analyze all Go packages
	analyzeDirectory("internal", packages, &violations)
	analyzeDirectory("pkg", packages, &violations)
	analyzeDirectory("services", packages, &violations)
	
	// Print comprehensive analysis
	printStructuralViolations(violations)
	printPackageAnalysis(packages)
	printCircularDependencies(packages)
	printReorganizationPlan(packages, violations)
}

func analyzeDirectory(rootDir string, packages map[string]*PackageInfo, violations *[]ViolationInfo) {
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return
	}
	
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		
		packagePath := filepath.Dir(path)
		if packages[packagePath] == nil {
			packages[packagePath] = &PackageInfo{
				Path:    packagePath,
				Files:   []string{},
				Imports: []string{},
			}
		}
		
		pkg := packages[packagePath]
		pkg.Files = append(pkg.Files, path)
		
		// Analyze file
		analyzeFile(path, pkg, violations)
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Error analyzing %s: %v\n", rootDir, err)
	}
}

func analyzeFile(filePath string, pkg *PackageInfo, violations *[]ViolationInfo) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return
	}
	
	// Count lines
	start := fset.Position(node.Pos())
	end := fset.Position(node.End())
	lines := end.Line - start.Line + 1
	pkg.TotalLines += lines
	
	// Check file size violation
	if lines > 500 {
		*violations = append(*violations, ViolationInfo{
			File:      filePath,
			Lines:     lines,
			Violation: fmt.Sprintf("File exceeds 500 lines (%d lines)", lines),
			Severity:  getSeverity(lines),
			Category:  "file_size",
		})
	}
	
	// Analyze AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ImportSpec:
			if x.Path != nil {
				importPath := strings.Trim(x.Path.Value, "\"")
				pkg.Imports = append(pkg.Imports, importPath)
				
				// Categorize imports
				if strings.Contains(importPath, "github.com/abdoElHodaky/tradSys/internal") {
					pkg.InternalImports = append(pkg.InternalImports, importPath)
				} else if strings.Contains(importPath, "github.com/abdoElHodaky/tradSys/pkg") {
					pkg.PkgImports = append(pkg.PkgImports, importPath)
				} else if strings.Contains(importPath, "github.com/abdoElHodaky/tradSys/services") {
					pkg.ServicesImports = append(pkg.ServicesImports, importPath)
				}
			}
		case *ast.FuncDecl:
			pkg.Functions++
		case *ast.TypeSpec:
			pkg.Types++
			if _, ok := x.Type.(*ast.InterfaceType); ok {
				pkg.Interfaces++
			}
		}
		return true
	})
}

func getSeverity(lines int) string {
	if lines > 1000 {
		return "CRITICAL"
	} else if lines > 750 {
		return "HIGH"
	} else if lines > 500 {
		return "MEDIUM"
	}
	return "LOW"
}

func printStructuralViolations(violations []ViolationInfo) {
	fmt.Println("\n🚨 STRUCTURAL VIOLATIONS ANALYSIS")
	fmt.Println(strings.Repeat("-", 80))
	
	// Group by category and severity
	categories := make(map[string][]ViolationInfo)
	severities := make(map[string]int)
	
	for _, v := range violations {
		categories[v.Category] = append(categories[v.Category], v)
		severities[v.Severity]++
	}
	
	fmt.Printf("Total Violations: %d\n", len(violations))
	fmt.Printf("  CRITICAL: %d | HIGH: %d | MEDIUM: %d | LOW: %d\n\n", 
		severities["CRITICAL"], severities["HIGH"], severities["MEDIUM"], severities["LOW"])
	
	// Print file size violations
	if fileViolations, ok := categories["file_size"]; ok {
		fmt.Printf("📏 FILE SIZE VIOLATIONS (%d files):\n", len(fileViolations))
		
		// Sort by lines descending
		sort.Slice(fileViolations, func(i, j int) bool {
			return fileViolations[i].Lines > fileViolations[j].Lines
		})
		
		for i, v := range fileViolations {
			if i >= 20 { // Show top 20
				fmt.Printf("  ... and %d more files\n", len(fileViolations)-20)
				break
			}
			fmt.Printf("  %s %s (%d lines) - %s\n", 
				getSeverityIcon(v.Severity), v.File, v.Lines, v.Violation)
		}
	}
}

func getSeverityIcon(severity string) string {
	switch severity {
	case "CRITICAL":
		return "🔴"
	case "HIGH":
		return "🟠"
	case "MEDIUM":
		return "🟡"
	default:
		return "🟢"
	}
}

func printPackageAnalysis(packages map[string]*PackageInfo) {
	fmt.Println("\n📦 PACKAGE STRUCTURE ANALYSIS")
	fmt.Println(strings.Repeat("-", 80))
	
	internalPkgs := []*PackageInfo{}
	pkgPkgs := []*PackageInfo{}
	servicesPkgs := []*PackageInfo{}
	
	for _, pkg := range packages {
		if strings.HasPrefix(pkg.Path, "internal") {
			internalPkgs = append(internalPkgs, pkg)
		} else if strings.HasPrefix(pkg.Path, "pkg") {
			pkgPkgs = append(pkgPkgs, pkg)
		} else if strings.HasPrefix(pkg.Path, "services") {
			servicesPkgs = append(servicesPkgs, pkg)
		}
	}
	
	fmt.Printf("📁 DIRECTORY DISTRIBUTION:\n")
	fmt.Printf("  internal/: %d packages, %d total lines\n", len(internalPkgs), getTotalLines(internalPkgs))
	fmt.Printf("  pkg/:      %d packages, %d total lines\n", len(pkgPkgs), getTotalLines(pkgPkgs))
	fmt.Printf("  services/: %d packages, %d total lines ⚠️  NON-STANDARD\n", len(servicesPkgs), getTotalLines(servicesPkgs))
	
	// Analyze import patterns
	fmt.Printf("\n🔗 IMPORT PATTERN ANALYSIS:\n")
	
	internalToInternal := 0
	internalToPkg := 0
	internalToServices := 0
	pkgToInternal := 0
	pkgToPkg := 0
	pkgToServices := 0
	
	for _, pkg := range packages {
		if strings.HasPrefix(pkg.Path, "internal") {
			internalToInternal += len(pkg.InternalImports)
			internalToPkg += len(pkg.PkgImports)
			internalToServices += len(pkg.ServicesImports)
		} else if strings.HasPrefix(pkg.Path, "pkg") {
			pkgToInternal += len(pkg.InternalImports)
			pkgToPkg += len(pkg.PkgImports)
			pkgToServices += len(pkg.ServicesImports)
		}
	}
	
	fmt.Printf("  internal → internal: %d imports\n", internalToInternal)
	fmt.Printf("  internal → pkg:      %d imports ✅ Good\n", internalToPkg)
	fmt.Printf("  internal → services: %d imports ⚠️  Should be eliminated\n", internalToServices)
	fmt.Printf("  pkg → internal:      %d imports ❌ VIOLATION - pkg should not import internal\n", pkgToInternal)
	fmt.Printf("  pkg → pkg:           %d imports ✅ Good\n", pkgToPkg)
	fmt.Printf("  pkg → services:      %d imports ⚠️  Should be eliminated\n", pkgToServices)
}

func getTotalLines(packages []*PackageInfo) int {
	total := 0
	for _, pkg := range packages {
		total += pkg.TotalLines
	}
	return total
}

func printCircularDependencies(packages map[string]*PackageInfo) {
	fmt.Println("\n🔄 CIRCULAR DEPENDENCY ANALYSIS")
	fmt.Println(strings.Repeat("-", 80))
	
	// Build dependency graph
	graph := make(map[string][]string)
	for _, pkg := range packages {
		deps := []string{}
		for _, imp := range pkg.InternalImports {
			depPkg := strings.Replace(imp, "github.com/abdoElHodaky/tradSys/", "", 1)
			deps = append(deps, depPkg)
		}
		for _, imp := range pkg.PkgImports {
			depPkg := strings.Replace(imp, "github.com/abdoElHodaky/tradSys/", "", 1)
			deps = append(deps, depPkg)
		}
		for _, imp := range pkg.ServicesImports {
			depPkg := strings.Replace(imp, "github.com/abdoElHodaky/tradSys/", "", 1)
			deps = append(deps, depPkg)
		}
		graph[pkg.Path] = deps
	}
	
	// Find cycles using DFS
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	cycles := [][]string{}
	
	for pkg := range graph {
		if !visited[pkg] {
			if cycle := findCycleDFS(pkg, graph, visited, recursionStack, []string{}); len(cycle) > 0 {
				cycles = append(cycles, cycle)
			}
		}
	}
	
	if len(cycles) == 0 {
		fmt.Println("✅ No circular dependencies detected")
	} else {
		fmt.Printf("❌ Found %d circular dependencies:\n", len(cycles))
		for i, cycle := range cycles {
			fmt.Printf("  %d. %s\n", i+1, strings.Join(cycle, " → "))
		}
	}
}

func findCycleDFS(pkg string, graph map[string][]string, visited, recursionStack map[string]bool, path []string) []string {
	visited[pkg] = true
	recursionStack[pkg] = true
	path = append(path, pkg)
	
	for _, dep := range graph[pkg] {
		if !visited[dep] {
			if cycle := findCycleDFS(dep, graph, visited, recursionStack, path); len(cycle) > 0 {
				return cycle
			}
		} else if recursionStack[dep] {
			// Found cycle
			cycleStart := -1
			for i, p := range path {
				if p == dep {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				return append(path[cycleStart:], dep)
			}
		}
	}
	
	recursionStack[pkg] = false
	return nil
}

func printReorganizationPlan(packages map[string]*PackageInfo, violations []ViolationInfo) {
	fmt.Println("\n🏗️  REORGANIZATION PLAN")
	fmt.Println(strings.Repeat("-", 80))
	
	// Count services packages that need relocation
	servicesToPkg := 0
	servicesToInternal := 0
	servicesLines := 0
	
	for _, pkg := range packages {
		if strings.HasPrefix(pkg.Path, "services") {
			servicesLines += pkg.TotalLines
			// Heuristic: if package has many types/interfaces, move to pkg
			if pkg.Types > pkg.Functions || pkg.Interfaces > 0 {
				servicesToPkg++
			} else {
				servicesToInternal++
			}
		}
	}
	
	fmt.Printf("📋 SERVICES DIRECTORY ELIMINATION:\n")
	fmt.Printf("  Total services packages: %d (%d lines)\n", servicesToPkg+servicesToInternal, servicesLines)
	fmt.Printf("  Move to pkg/: %d packages (types/interfaces)\n", servicesToPkg)
	fmt.Printf("  Move to internal/: %d packages (implementations)\n", servicesToInternal)
	
	// Count file splitting needed
	criticalFiles := 0
	highFiles := 0
	mediumFiles := 0
	
	for _, v := range violations {
		if v.Category == "file_size" {
			switch v.Severity {
			case "CRITICAL":
				criticalFiles++
			case "HIGH":
				highFiles++
			case "MEDIUM":
				mediumFiles++
			}
		}
	}
	
	fmt.Printf("\n📏 FILE SPLITTING REQUIREMENTS:\n")
	fmt.Printf("  Critical (>1000 lines): %d files - IMMEDIATE ACTION\n", criticalFiles)
	fmt.Printf("  High (750-1000 lines): %d files - HIGH PRIORITY\n", highFiles)
	fmt.Printf("  Medium (500-750 lines): %d files - MEDIUM PRIORITY\n", mediumFiles)
	
	fmt.Printf("\n⏱️  ESTIMATED EFFORT:\n")
	fmt.Printf("  Services elimination: %d hours\n", (servicesToPkg+servicesToInternal)*2)
	fmt.Printf("  File splitting: %d hours\n", criticalFiles*3+highFiles*2+mediumFiles*1)
	fmt.Printf("  Import path updates: %d hours\n", 8)
	fmt.Printf("  Testing & validation: %d hours\n", 12)
	fmt.Printf("  TOTAL: %d hours\n", (servicesToPkg+servicesToInternal)*2+criticalFiles*3+highFiles*2+mediumFiles*1+20)
}

