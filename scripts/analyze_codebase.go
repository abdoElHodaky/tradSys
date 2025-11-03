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

type FileAnalysis struct {
	Path         string
	Lines        int
	Imports      []string
	Functions    int
	IfStmts      int
	SwitchStmts  int
	Complexity   int
	HasCircular  bool
}

type CircularDep struct {
	Package1 string
	Package2 string
	Chain    []string
}

func main() {
	fmt.Println("🔍 TradSys Codebase Analysis")
	fmt.Println("=" * 50)
	
	var files []FileAnalysis
	var circularDeps []CircularDep
	packageImports := make(map[string][]string)
	
	// Walk through internal directory
	err := filepath.Walk("internal", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		
		analysis := analyzeFile(path)
		files = append(files, analysis)
		
		// Track package imports for circular dependency detection
		pkg := filepath.Dir(path)
		packageImports[pkg] = analysis.Imports
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		return
	}
	
	// Sort files by line count
	sort.Slice(files, func(i, j int) bool {
		return files[i].Lines > files[j].Lines
	})
	
	// Print analysis results
	printFileAnalysis(files)
	printCircularDependencies(packageImports)
	printSummary(files)
}

func analyzeFile(path string) FileAnalysis {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return FileAnalysis{Path: path, Lines: 0}
	}
	
	analysis := FileAnalysis{
		Path:    path,
		Imports: make([]string, 0),
	}
	
	// Count lines
	start := fset.Position(node.Pos())
	end := fset.Position(node.End())
	analysis.Lines = end.Line - start.Line + 1
	
	// Analyze AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ImportSpec:
			if x.Path != nil {
				importPath := strings.Trim(x.Path.Value, "\"")
				if strings.Contains(importPath, "github.com/abdoElHodaky/tradSys/internal") {
					analysis.Imports = append(analysis.Imports, importPath)
				}
			}
		case *ast.FuncDecl:
			analysis.Functions++
		case *ast.IfStmt:
			analysis.IfStmts++
			analysis.Complexity++
		case *ast.SwitchStmt:
			analysis.SwitchStmts++
			analysis.Complexity++
		case *ast.TypeSwitchStmt:
			analysis.SwitchStmts++
			analysis.Complexity++
		case *ast.RangeStmt:
			analysis.Complexity++
		case *ast.ForStmt:
			analysis.Complexity++
		}
		return true
	})
	
	return analysis
}

func printFileAnalysis(files []FileAnalysis) {
	fmt.Println("\n📊 FILE SIZE ANALYSIS")
	fmt.Println("Files exceeding 500 lines (need splitting):")
	
	largeFiles := 0
	for _, file := range files {
		if file.Lines > 500 {
			largeFiles++
			fmt.Printf("  %s: %d lines (Functions: %d, If: %d, Switch: %d, Complexity: %d)\n",
				file.Path, file.Lines, file.Functions, file.IfStmts, file.SwitchStmts, file.Complexity)
		}
	}
	
	fmt.Printf("\nTotal files exceeding 500 lines: %d\n", largeFiles)
	
	fmt.Println("\n🔥 TOP 10 LARGEST FILES:")
	for i, file := range files {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s: %d lines\n", i+1, file.Path, file.Lines)
	}
}

func printCircularDependencies(packageImports map[string][]string) {
	fmt.Println("\n🔄 CIRCULAR DEPENDENCY ANALYSIS")
	
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	cycles := make([][]string, 0)
	
	for pkg := range packageImports {
		if !visited[pkg] {
			if cycle := findCycle(pkg, packageImports, visited, recursionStack, []string{}); len(cycle) > 0 {
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

func findCycle(pkg string, imports map[string][]string, visited, recursionStack map[string]bool, path []string) []string {
	visited[pkg] = true
	recursionStack[pkg] = true
	path = append(path, pkg)
	
	for _, imp := range imports[pkg] {
		// Convert import path to package path
		impPkg := strings.Replace(imp, "github.com/abdoElHodaky/tradSys/", "", 1)
		
		if !visited[impPkg] {
			if cycle := findCycle(impPkg, imports, visited, recursionStack, path); len(cycle) > 0 {
				return cycle
			}
		} else if recursionStack[impPkg] {
			// Found cycle
			cycleStart := -1
			for i, p := range path {
				if p == impPkg {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				return append(path[cycleStart:], impPkg)
			}
		}
	}
	
	recursionStack[pkg] = false
	return nil
}

func printSummary(files []FileAnalysis) {
	fmt.Println("\n📈 SUMMARY STATISTICS")
	
	totalLines := 0
	totalFunctions := 0
	totalIfStmts := 0
	totalSwitchStmts := 0
	totalComplexity := 0
	filesOver500 := 0
	filesOver1000 := 0
	
	for _, file := range files {
		totalLines += file.Lines
		totalFunctions += file.Functions
		totalIfStmts += file.IfStmts
		totalSwitchStmts += file.SwitchStmts
		totalComplexity += file.Complexity
		
		if file.Lines > 500 {
			filesOver500++
		}
		if file.Lines > 1000 {
			filesOver1000++
		}
	}
	
	fmt.Printf("Total files analyzed: %d\n", len(files))
	fmt.Printf("Total lines of code: %d\n", totalLines)
	fmt.Printf("Total functions: %d\n", totalFunctions)
	fmt.Printf("Total if statements: %d\n", totalIfStmts)
	fmt.Printf("Total switch statements: %d\n", totalSwitchStmts)
	fmt.Printf("Total complexity score: %d\n", totalComplexity)
	fmt.Printf("Files over 500 lines: %d (%.1f%%)\n", filesOver500, float64(filesOver500)/float64(len(files))*100)
	fmt.Printf("Files over 1000 lines: %d (%.1f%%)\n", filesOver1000, float64(filesOver1000)/float64(len(files))*100)
	
	if len(files) > 0 {
		avgLines := float64(totalLines) / float64(len(files))
		avgComplexity := float64(totalComplexity) / float64(len(files))
		fmt.Printf("Average lines per file: %.1f\n", avgLines)
		fmt.Printf("Average complexity per file: %.1f\n", avgComplexity)
	}
}

