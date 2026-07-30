package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePrefix = "k8s-platform-backend/internal/"

var migratedContexts = []string{"ai", "audit", "change", "fleet", "iam", "incident", "kops", "platform", "provisioning", "workspace"}

func TestDomainPackagesStayFrameworkIndependent(t *testing.T) {
	root := backendRoot(t)
	for _, contextName := range migratedContexts {
		domainDir := filepath.Join(root, "internal", contextName, "domain")
		walkGoFiles(t, domainDir, func(path string, file *ast.File) {
			for _, spec := range file.Imports {
				importPath := unquoteImport(t, spec.Path.Value)
				if strings.HasPrefix(importPath, "k8s-platform-backend/") || isThirdPartyImport(importPath) {
					t.Errorf("%s: domain imports non-standard package %q", path, importPath)
				}
			}
		})
	}
}

func TestApplicationPackagesDoNotDependOnAdaptersOrLegacyLayers(t *testing.T) {
	root := backendRoot(t)
	for _, contextName := range migratedContexts {
		applicationDir := filepath.Join(root, "internal", contextName, "application")
		if _, err := os.Stat(applicationDir); os.IsNotExist(err) {
			continue
		}
		walkGoFiles(t, applicationDir, func(path string, file *ast.File) {
			for _, spec := range file.Imports {
				importPath := unquoteImport(t, spec.Path.Value)
				if importPath == "gorm.io/gorm" {
					t.Errorf("%s: application imports database implementation %q; depend on a context port instead", path, importPath)
				}
				for _, forbidden := range []string{"/adapters/", "/legacy/", "/router"} {
					if strings.Contains(importPath, forbidden) {
						t.Errorf("%s: application imports forbidden layer %q", path, importPath)
					}
				}
			}
		})
	}
}

func TestMigratedContextsDoNotDependOnLegacyBusinessLayers(t *testing.T) {
	root := backendRoot(t)
	for _, contextName := range migratedContexts {
		contextDir := filepath.Join(root, "internal", contextName)
		walkGoFiles(t, contextDir, func(path string, file *ast.File) {
			for _, spec := range file.Imports {
				importPath := unquoteImport(t, spec.Path.Value)
				for _, forbidden := range []string{
					modulePrefix + "legacy/controller",
					modulePrefix + "legacy/model",
					modulePrefix + "legacy/service",
					modulePrefix + "router",
				} {
					if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
						t.Errorf("%s: migrated context imports legacy business layer %q", path, importPath)
					}
				}
			}
		})
	}
}

func TestContextsDoNotImportOtherContextInternals(t *testing.T) {
	root := backendRoot(t)
	contexts := []string{"ai", "audit", "change", "fleet", "iam", "incident", "kops", "platform", "provisioning", "workspace"}
	for _, owner := range contexts {
		contextDir := filepath.Join(root, "internal", owner)
		if _, err := os.Stat(contextDir); os.IsNotExist(err) {
			continue
		}
		walkGoFiles(t, contextDir, func(path string, file *ast.File) {
			for _, spec := range file.Imports {
				importPath := unquoteImport(t, spec.Path.Value)
				for _, other := range contexts {
					if other != owner && strings.HasPrefix(importPath, modulePrefix+other+"/") {
						t.Errorf("%s: context %s imports internal package from %s: %q", path, owner, other, importPath)
					}
				}
			}
		})
	}
}

func TestKopsRuntimeAdaptersStayWithinKopsOrSharedTransport(t *testing.T) {
	runtimeDir := filepath.Join(backendRoot(t), "internal", "kops", "adapters", "runtime")
	allowedPrefixes := []string{
		modulePrefix + "kops/",
		modulePrefix + "transport/",
	}
	walkGoFiles(t, runtimeDir, func(path string, file *ast.File) {
		for _, spec := range file.Imports {
			importPath := unquoteImport(t, spec.Path.Value)
			if !strings.HasPrefix(importPath, modulePrefix) {
				continue
			}
			for _, allowed := range allowedPrefixes {
				if strings.HasPrefix(importPath, allowed) {
					return
				}
			}
			t.Errorf("%s: Kops runtime adapter imports a foreign context %q", path, importPath)
		}
	})
}

func TestCompositionRootCatalogsEveryCurrentModule(t *testing.T) {
	type moduleCatalog struct {
		audit        struct{}
		iam          struct{}
		platform     struct{}
		workspace    struct{}
		fleet        struct{}
		kops         struct{}
		ai           struct{}
		change       struct{}
		provisioning struct{}
		incident     struct{}
	}
	expected := fieldNames(reflect.TypeOf(moduleCatalog{}))
	actual := applicationModuleFields(t, filepath.Join(backendRoot(t), "internal", "router", "modules.go"))
	if strings.Join(actual, ",") != strings.Join(expected, ",") {
		t.Fatalf("applicationModules fields = %v, want %v", actual, expected)
	}
}

func TestRouterCompositionRootOnlyComposesRoutes(t *testing.T) {
	path := filepath.Join(backendRoot(t), "internal", "router", "router.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var functions []string
	for _, declaration := range parsed.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			functions = append(functions, function.Name.Name)
		}
	}
	sort.Strings(functions)
	want := []string{"New", "registerRoutes"}
	if strings.Join(functions, ",") != strings.Join(want, ",") {
		t.Fatalf("router.go functions = %v, want composition functions %v", functions, want)
	}
}

func TestLegacyCompatibilityTreeIsEmpty(t *testing.T) {
	legacyRoot := filepath.Join(backendRoot(t), "internal", "legacy")
	if _, err := os.Stat(legacyRoot); os.IsNotExist(err) {
		return
	}
	count := 0
	err := filepath.WalkDir(legacyRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk legacy business layer: %v", err)
	}
	if count != 0 {
		t.Fatalf("legacy Go files = %d; compatibility tree must remain empty", count)
	}
}

func TestLegacyModelCompatibilityPackageIsEmpty(t *testing.T) {
	modelDir := filepath.Join(backendRoot(t), "internal", "legacy", "model")
	entries, err := os.ReadDir(modelDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read legacy model directory: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			t.Errorf("legacy model compatibility source must not be reintroduced: %s", entry.Name())
		}
	}
}

func TestMigrationNumbersDoNotIntroduceNewDuplicates(t *testing.T) {
	migrationRoot := filepath.Join(backendRoot(t), "internal", "db", "migrations")
	entries, err := os.ReadDir(migrationRoot)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	knownLegacyDuplicates := map[string]bool{"006": true, "016": true, "026": true}
	counts := map[string]int{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		prefix, _, found := strings.Cut(name, "_")
		if !found {
			t.Errorf("migration has no numeric prefix: %s", name)
			continue
		}
		counts[prefix]++
	}
	for prefix, count := range counts {
		if count > 1 && !knownLegacyDuplicates[prefix] {
			t.Errorf("migration number %s is used by %d files", prefix, count)
		}
	}
}

func applicationModuleFields(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var fields []string
	for _, declaration := range parsed.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "applicationModules" {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatal("applicationModules must be a struct")
			}
			for _, field := range structure.Fields.List {
				for _, name := range field.Names {
					fields = append(fields, name.Name)
				}
			}
		}
	}
	sort.Strings(fields)
	return fields
}

func fieldNames(value reflect.Type) []string {
	fields := make([]string, 0, value.NumField())
	for index := 0; index < value.NumField(); index++ {
		fields = append(fields, value.Field(index).Name)
	}
	sort.Strings(fields)
	return fields
}

func walkGoFiles(t *testing.T, root string, visit func(string, *ast.File)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		visit(path, parsed)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

func backendRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func unquoteImport(t *testing.T, value string) string {
	t.Helper()
	importPath, err := strconv.Unquote(value)
	if err != nil {
		t.Fatalf("unquote import %s: %v", value, err)
	}
	return importPath
}

func isThirdPartyImport(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return strings.Contains(first, ".")
}
