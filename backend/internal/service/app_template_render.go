package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

type templateRenderOptions struct {
	ReleaseName string
	Namespace   string
	Revision    int
}

// RenderManifest 返回模板真实渲染后的清单内容，供发布链路复用。
func (s *AppTemplateService) RenderManifest(ctx context.Context, id uint64, values map[string]interface{}) (string, error) {
	detail, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return renderAppTemplateManifest(detail, values, templateRenderOptions{})
}

func renderAppTemplateManifest(detail *AppTemplateDetail, values map[string]interface{}, opts templateRenderOptions) (string, error) {
	if detail == nil {
		return "", ErrWithMessage(ErrInvalidParams, "模板不存在")
	}
	merged := mergeTemplateValues(detail.DefaultValues, values)
	if detail.Engine == "yaml" {
		return strings.TrimSpace(detail.Manifest), nil
	}
	manifest := strings.TrimSpace(detail.Manifest)
	if manifest == "" {
		return "", ErrWithMessage(ErrInvalidParams, "模板内容为空")
	}
	tmpl, err := template.New("app-template").
		Option("missingkey=zero").
		Funcs(template.FuncMap{
			"default":  templateDefault,
			"quote":    templateQuote,
			"toJson":   templateToJSON,
			"indent":   templateIndent,
			"nindent":  templateNindent,
			"required": templateRequired,
			"upper":    strings.ToUpper,
			"lower":    strings.ToLower,
			"trim":     strings.TrimSpace,
		}).
		Parse(manifest)
	if err != nil {
		return "", ErrWithMessage(ErrInvalidParams, fmt.Sprintf("Helm 模板解析失败：%v", err))
	}
	root := buildTemplateRenderContext(detail, merged, opts)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, root); err != nil {
		return "", ErrWithMessage(ErrInvalidParams, fmt.Sprintf("Helm 模板渲染失败：%v", err))
	}
	return strings.TrimSpace(buf.String()), nil
}

func buildTemplateRenderContext(detail *AppTemplateDetail, values map[string]interface{}, opts templateRenderOptions) map[string]interface{} {
	root := make(map[string]interface{}, len(values)+4)
	for key, value := range values {
		root[key] = value
	}
	root["Values"] = values
	root["Chart"] = map[string]interface{}{
		"Name":       firstNonEmpty(detail.ChartName, detail.Name),
		"Version":    detail.Version,
		"AppVersion": detail.AppVersion,
	}
	root["Release"] = map[string]interface{}{
		"Name":      firstNonEmpty(strings.TrimSpace(opts.ReleaseName), detail.Name),
		"Namespace": strings.TrimSpace(opts.Namespace),
		"Revision":  opts.Revision,
		"Service":   "Helm",
	}
	return root
}

func mergeTemplateValues(base, override map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+len(override))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range override {
		out[key] = value
	}
	return out
}

func templateDefault(defaultValue, givenValue interface{}) interface{} {
	if isTemplateEmpty(givenValue) {
		return defaultValue
	}
	return givenValue
}

func templateQuote(value interface{}) string {
	return strconv.Quote(templateString(value))
}

func templateToJSON(value interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(b)
}

func templateIndent(spaces int, value interface{}) string {
	padding := strings.Repeat(" ", maxInt(spaces, 0))
	lines := strings.Split(templateString(value), "\n")
	for i := range lines {
		if lines[i] == "" {
			continue
		}
		lines[i] = padding + lines[i]
	}
	return strings.Join(lines, "\n")
}

func templateNindent(spaces int, value interface{}) string {
	return "\n" + templateIndent(spaces, value)
}

func templateRequired(message string, value interface{}) (interface{}, error) {
	if isTemplateEmpty(value) {
		return nil, fmt.Errorf(strings.TrimSpace(message))
	}
	return value, nil
}

func templateString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func isTemplateEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return rv.IsNil()
	}
	return false
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
