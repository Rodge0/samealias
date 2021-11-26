package samealias

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var imports = map[string]string{}
var p = fmt.Println

var Analyzer = &analysis.Analyzer{
	Name: "samealias",
	Doc:  "check different aliases for same package",
	Run:  run,

	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {

	// _, file, _, _ := runtime.Caller(1) // 得到当前执行的文件
	// p("current file:   ", file)
	// _, file, _, _ = runtime.Caller(0) // 得到调用者的文件路径
	// p("current file:   ", file)
	// // p("run: <pass.Files> - ", pass.Files)               // 0x ，看起来是地址，不明所以
	// // p("run: <pass.OtherFiles> - ", pass.OtherFiles)     // 空，不明所以
	// // p("run: <pass.IgnoredFiles> - ", pass.IgnoredFiles) // 空，不明所以
	// p("run: <pass.Files[0].Name> - ", pass.Files[0].Name)                 // main，不明所以
	// p("run: <pass.Files[0].Name.Name> - ", pass.Files[0].Name.Name)       // main，不明所以
	// p("run: <pass.Files[0].Name.NamePos> - ", pass.Files[0].Name.NamePos) // 数字，不明所以
	// p("run: <pass.Files[0].Imports> - ", pass.Files[0].Imports)           // 很多个0x，不明所以
	// p("run: <pass.Files[0].Package> - ", pass.Files[0].Package)           // 数字，不明所以
	// p("run: <pass.Files[0].Name> - ", pass.Files[0].Name)                 //

	filename := pass.Fset.Position(pass.Files[0].Pos())
	// p("filename --------- ", filename.Filename)

	//if handlePath(filename.String()) {
	if res, err := isAutogenFile(filename.Filename); res && err == nil {
		p(filename.Filename, "test file, no need handle")
	} else {
		p(filename.Filename, "not test file, need handle")
		// p("run: <pass.Fset> - ", pass.Fset) //&{{{0 0} 0 0 0 0} 4326435 [0xc000467c20 0xc0004e88a0 0xc0004e8a20 0xc0004e9380 0xc0004e9e60 0xc000288420 0xc0002884e0 0xc000288660 0xc00028878 .....

		inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		inspect.Preorder([]ast.Node{(*ast.ImportSpec)(nil)}, func(n ast.Node) {
			visitImportSpecNode(n.(*ast.ImportSpec), pass)
		})
	}
	return nil, nil
}

func isAutogenFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines := strings.ToUpper(scanner.Text())
		// p("lines: ", lines)
		if strings.Contains(lines, "PACKAGE") {
			return false, scanner.Err()
		}
		if strings.Contains(lines, "DO NOT EDIT") {
			return true, scanner.Err()
		}
	}
	return false, scanner.Err()
}

/*
// 也可以用filepath.Join 在跨OS下合成
// path := strings.Split(filepath.Join("tmp", "gen", "tmp"), "tmp")[1]

var fileAllowList = []string{
	"gen/",
	"gen\\",
	"mocks/",
	"mocks\\",
	"mock/",
	"mock\\",
	"_mock.go",
	".pb.go",
	"_gen.go",
	"gen.copier.go",
	"models_gen_easyjson.go",
}

// ignore autogen or test files
func handlePath(path string) bool {
	strs := strings.Split(path, "backendv2/")
	if len(strs) < 2 {
		return false
	}
	path = strs[1]
	p(path)
	for _, str := range fileAllowList {
		if strings.Contains(path, str) {
			return true
		}
	}
	return false
}
*/
func visitImportSpecNode(node *ast.ImportSpec, pass *analysis.Pass) {
	// 如果没有别名就结束
	if node.Name == nil {
		return
	}

	// pos := pass.Fset.Position(node.Pos()) // 获取被处理文件的绝对路径
	// p("visitImportSpecNode: <pass.Fset.Position(node.Pos())>", pos)

	// p("visitImportSpecNode: <node.Name> - ", node.Name) // node.Name = 别名

	alias := ""
	if node.Name != nil {
		alias = node.Name.String()
	}

	// p("visitImportSpecNode: <node.Name.String()> - ", node.Name.String()) // 还是别名

	// 忽略了 . 和 _ 别名的情况，因为大多用于测试和自动包含
	if alias == "." {
		return // Dot aliases are generally used in tests, so ignore.
	}

	if strings.HasPrefix(alias, "_") {
		return // Used by go test and for auto-includes, not a conflict.
	}

	// 去掉别名的双引号，因为node.Path.Value返回的是带引号的package路径
	path, err := strconv.Unquote(node.Path.Value)
	if err != nil {
		pass.Reportf(node.Pos(), "import not quoted")
	}

	// p("visitImportSpecNode: <strconv.Unquote(node.Path.Value)> - ", path) //package路径
	// p("visitImportSpecNode: <node.Path.Value> - ", node.Path.Value)       //  带引号的package路径
	// p("visitImportSpecNode: <node.Pos()> - ", node.Pos())           // 一串数字，不明所以
	// p("visitImportSpecNode: <node.Path.Pos()> - ", node.Path.Pos())       // 一串数字，不明所以
	// p("visitImportSpecNode: <node.Path.ValuePos> - ", node.Path.ValuePos) // 一串数字，不明所以
	// p("visitImportSpecNode: <node.Path.Kind> - ", node.Path.Kind)         // pakcage类型，string
	// p("visitImportSpecNode: <node.Path.End()> - ", node.Path.End())       // 一串数字，不明所以

	if alias != "" {
		val, ok := imports[path]
		if ok {
			if val != alias {
				message := fmt.Sprintf("package %q have different alias, %q, %q", path, alias, val)

				pass.Report(analysis.Diagnostic{
					Pos:     node.Pos(),
					End:     node.End(),
					Message: message,
					SuggestedFixes: []analysis.SuggestedFix{{
						Message:   "Use same alias or do not use alias",
						TextEdits: findEdits(node, pass.TypesInfo.Uses, path, alias, val),
					}},
				})
			}
		} else {
			imports[path] = alias
		}
	}
}

func findEdits(node ast.Node, uses map[*ast.Ident]types.Object, importPath, original, required string) []analysis.TextEdit {
	// Edit the actual import line.
	importLine := strconv.Quote(importPath)
	if required != "" {
		importLine = required + " " + importLine
	}
	result := []analysis.TextEdit{{
		Pos:     node.Pos(),
		End:     node.End(),
		NewText: []byte(importLine),
	}}

	packageReplacement := required
	if required == "" {
		packageParts := strings.Split(importPath, "/")
		if len(packageParts) != 0 {
			packageReplacement = packageParts[len(packageParts)-1]
		} else {
			// fall back to original
			packageReplacement = original
		}
	}

	// Edit all the uses of the alias in the code.
	for use, pkg := range uses {
		pkgName, ok := pkg.(*types.PkgName)
		if !ok {
			// skip identifiers that aren't pointing at a PkgName.
			continue
		}

		if pkgName.Pos() != node.Pos() {
			// skip identifiers pointing to a different import statement.
			continue
		}

		result = append(result, analysis.TextEdit{
			Pos:     use.Pos(),
			End:     use.End(),
			NewText: []byte(packageReplacement),
		})
	}

	return result
}
