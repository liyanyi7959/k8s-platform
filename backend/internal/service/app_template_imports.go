package service

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	yamlv3 "gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

type importedTemplateContent struct {
	Name               string
	Version            string
	AppVersion         string
	Summary            string
	Manifest           string
	Readme             string
	EnvExample         string
	ProjectNameDefault string
	InstallDirDefault  string
	ValuesSchema       map[string]interface{}
	DefaultValues      map[string]interface{}
	ExtraFiles         []string
	SourceRef          map[string]interface{}
}

type helmChartMeta struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	AppVersion  string `yaml:"appVersion"`
	Description string `yaml:"description"`
}

func fetchRemoteTemplatePackage(ctx context.Context, sourceURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", ErrWithMessage(ErrInvalidParams, "远程地址不合法")
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", ErrWithMessage(ErrInvalidParams, "无法访问远程地址，请检查网络或地址配置")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, "", ErrWithMessage(ErrInvalidParams, fmt.Sprintf("远程地址返回异常状态：%d", resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, "", ErrWithMessage(ErrInvalidParams, "读取远程模板内容失败")
	}
	filename := path.Base(req.URL.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = "remote-package"
	}
	return body, filename, nil
}

func parseImportedTemplate(engine, filename string, content []byte) (*importedTemplateContent, error) {
	if strings.EqualFold(engine, "helm") {
		return parseHelmTemplatePackage(filename, content)
	}
	return parseComposeTemplatePackage(filename, content)
}

func parseHelmTemplatePackage(filename string, content []byte) (*importedTemplateContent, error) {
	files, err := extractPackageFiles(filename, content)
	if err != nil {
		return nil, err
	}
	root := detectHelmChartRoot(files)
	if root == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "未在导入包中找到 Chart.yaml")
	}
	chartText := files[path.Join(root, "Chart.yaml")]
	var meta helmChartMeta
	if err := yamlv3.Unmarshal([]byte(chartText), &meta); err != nil {
		return nil, ErrWithMessage(ErrInvalidParams, "Chart.yaml 解析失败")
	}
	defaultValues := map[string]interface{}{}
	if raw, ok := files[path.Join(root, "values.yaml")]; ok && strings.TrimSpace(raw) != "" {
		if err := k8syaml.Unmarshal([]byte(raw), &defaultValues); err != nil {
			defaultValues = map[string]interface{}{}
		}
	}
	valuesSchema := map[string]interface{}{}
	if raw, ok := files[path.Join(root, "values.schema.json")]; ok && strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &valuesSchema)
	}
	readme := files[path.Join(root, "README.md")]
	manifestFiles := make([]string, 0)
	for fileName := range files {
		if strings.HasPrefix(fileName, path.Join(root, "templates")+"/") && (strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") || strings.HasSuffix(fileName, ".tpl")) {
			manifestFiles = append(manifestFiles, fileName)
		}
	}
	sort.Strings(manifestFiles)
	manifestParts := make([]string, 0, len(manifestFiles))
	for _, fileName := range manifestFiles {
		manifestParts = append(manifestParts, fmt.Sprintf("# file: %s\n%s", strings.TrimPrefix(fileName, root+"/"), files[fileName]))
	}
	manifest := strings.Join(manifestParts, "\n\n---\n\n")
	if strings.TrimSpace(manifest) == "" {
		manifest = "# templates 目录为空，请补充 Chart 模板文件"
	}
	return &importedTemplateContent{
		Name:               firstNonEmpty(strings.TrimSpace(meta.Name), strings.TrimSuffix(filename, filepath.Ext(filename))),
		Version:            firstNonEmpty(strings.TrimSpace(meta.Version), "1.0.0"),
		AppVersion:         strings.TrimSpace(meta.AppVersion),
		Summary:            strings.TrimSpace(meta.Description),
		Manifest:           manifest,
		Readme:             readme,
		ValuesSchema:       valuesSchema,
		DefaultValues:      defaultValues,
		ExtraFiles:         listFileNames(files),
		SourceRef:          map[string]interface{}{"root": root, "file_count": len(files)},
		ProjectNameDefault: strings.TrimSpace(meta.Name),
	}, nil
}

func parseComposeTemplatePackage(filename string, content []byte) (*importedTemplateContent, error) {
	trimmedName := strings.ToLower(filename)
	if strings.HasSuffix(trimmedName, ".yml") || strings.HasSuffix(trimmedName, ".yaml") {
		base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), ".compose")
		return &importedTemplateContent{
			Name:               sanitizeTemplateName(base),
			Version:            "1.0.0",
			Summary:            "从 Compose 文件导入",
			Manifest:           string(content),
			ProjectNameDefault: sanitizeTemplateName(base),
			InstallDirDefault:  "/opt/apps/" + sanitizeTemplateName(base),
			ValuesSchema:       map[string]interface{}{},
			DefaultValues:      map[string]interface{}{},
			ExtraFiles:         []string{filename},
			SourceRef:          map[string]interface{}{"import_file": filename},
		}, nil
	}
	files, err := extractPackageFiles(filename, content)
	if err != nil {
		return nil, err
	}
	composeFile := detectComposeMainFile(files)
	if composeFile == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "未在导入包中找到 docker-compose.yml 或 compose.yml")
	}
	projectName := sanitizeTemplateName(strings.TrimSuffix(filepath.Base(composeFile), filepath.Ext(composeFile)))
	if projectName == "docker-compose" || projectName == "compose" {
		projectName = sanitizeTemplateName(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	}
	if projectName == "" {
		projectName = "compose-app"
	}
	envExample := firstNonEmpty(files[path.Join(path.Dir(composeFile), ".env.example")], files[path.Join(path.Dir(composeFile), ".env")])
	readme := firstNonEmpty(files[path.Join(path.Dir(composeFile), "README.md")], files["README.md"])
	return &importedTemplateContent{
		Name:               projectName,
		Version:            "1.0.0",
		Summary:            "从 Docker Compose 模板导入",
		Manifest:           files[composeFile],
		Readme:             readme,
		EnvExample:         envExample,
		ProjectNameDefault: projectName,
		InstallDirDefault:  "/opt/apps/" + projectName,
		ValuesSchema:       map[string]interface{}{},
		DefaultValues:      map[string]interface{}{},
		ExtraFiles:         listFileNames(files),
		SourceRef:          map[string]interface{}{"main_file": composeFile, "file_count": len(files)},
	}, nil
}

func extractPackageFiles(filename string, content []byte) (map[string]string, error) {
	lowerName := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lowerName, ".zip"):
		return extractZipFiles(content)
	case strings.HasSuffix(lowerName, ".tgz"), strings.HasSuffix(lowerName, ".tar.gz"):
		return extractTarGzFiles(content)
	case strings.HasSuffix(lowerName, ".yaml"), strings.HasSuffix(lowerName, ".yml"):
		return map[string]string{filename: string(content)}, nil
	default:
		return nil, ErrWithMessage(ErrInvalidParams, "暂不支持该文件类型，请上传 .tgz、.tar.gz、.zip、.yaml 或 .yml")
	}
}

func extractZipFiles(content []byte) (map[string]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, ErrWithMessage(ErrInvalidParams, "压缩包不是有效的 zip 文件")
	}
	files := make(map[string]string)
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !isTextTemplateFile(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "读取压缩包文件失败")
		}
		data, readErr := io.ReadAll(io.LimitReader(rc, 2<<20))
		_ = rc.Close()
		if readErr != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "读取压缩包内容失败")
		}
		files[normalizeArchivePath(f.Name)] = string(data)
	}
	return files, nil
}

func extractTarGzFiles(content []byte) (map[string]string, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, ErrWithMessage(ErrInvalidParams, "压缩包不是有效的 tar.gz 文件")
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	files := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "解压 tar.gz 内容失败")
		}
		if hdr.FileInfo().IsDir() || !isTextTemplateFile(hdr.Name) {
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(tr, 2<<20))
		if readErr != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "读取 tar.gz 内容失败")
		}
		files[normalizeArchivePath(hdr.Name)] = string(data)
	}
	return files, nil
}

func detectHelmChartRoot(files map[string]string) string {
	roots := make([]string, 0)
	for name := range files {
		if path.Base(name) == "Chart.yaml" {
			roots = append(roots, path.Dir(name))
		}
	}
	sort.Strings(roots)
	if len(roots) == 0 {
		return ""
	}
	return roots[0]
}

func detectComposeMainFile(files map[string]string) string {
	preferred := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, name := range preferred {
		for fileName := range files {
			if path.Base(fileName) == name {
				return fileName
			}
		}
	}
	return ""
}

func listFileNames(files map[string]string) []string {
	out := make([]string, 0, len(files))
	for name := range files {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func normalizeArchivePath(name string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(name, "\\", "/")), "./")
}

func isTextTemplateFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".tpl") || strings.HasSuffix(lower, ".env") || strings.HasSuffix(lower, ".example")
}

func sanitizeTemplateName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "app-template"
	}
	return name
}
