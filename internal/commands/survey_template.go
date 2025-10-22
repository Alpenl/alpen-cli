package commands

import (
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
)

var (
	menuSelectTemplate = `
{{- define "option"}}
    {{- $desc := $.GetDescription .CurrentOpt }}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color "cyan"}}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}{{color "reset"}}
    {{- if ne $desc ""}} {{color "240"}}- {{$desc}}{{color "reset"}}{{end}}
{{end}}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "cyan"}}▸ {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "reset"}}{{"\n"}}
{{- else}}
{{- if .FilterMessage}}
{{"\n"}}{{color "cyan"}}{{ .FilterMessage }}{{color "reset"}}
{{- end}}
{{"\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- if gt $ix 0}}{{"\n"}}{{- end}}
    {{- template "option" $.IterateOption $ix $option}}
  {{- end}}
{{- end}}`

	envSelectTemplate = `
{{- define "envOption"}}
    {{- $meta := optionMeta .CurrentOpt.Index -}}
    {{- if eq $meta.Name "" }}
        {{- template "defaultOption" . }}
    {{- else }}
        {{- if $meta.First }}
{{- if $meta.GapBefore }}{{else}}{{end}}  {{color "cyan"}}{{ $meta.Group }}{{color "reset"}}
        {{- end }}
        {{- if eq .SelectedIndex .CurrentIndex }}
   {{color "cyan"}}{{ .Config.Icons.SelectFocus.Text }}{{color "reset"}}  {{if $meta.Active}}{{color "green"}}{{end}}{{$meta.Name}}{{color "reset"}}
        {{- else if $meta.Active }}
      {{color "green"}}{{$meta.Name}}{{color "reset"}}
        {{- else }}
      {{$meta.Name}}
        {{- end }}
    {{- end }}
{{- end }}
{{- define "defaultOption"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color "cyan"}}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}{{ if ne ($.GetDescription .CurrentOpt) "" }} - {{color "240"}}{{ $.GetDescription .CurrentOpt }}{{end}}
    {{- color "reset"}}
{{- end }}
{{- define "option"}}
    {{- if metaEnabled }}{{template "envOption" .}}{{else}}{{template "defaultOption" .}}{{end}}
{{- end }}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{end}}
{{- color "cyan"}}▸ {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{color "reset"}}{{"\n"}}
{{- if .ShowAnswer}}{{color "reset"}}{{"\n"}}
{{- else}}
{{- if .FilterMessage }}{{"\n"}}{{color "cyan"}}{{ .FilterMessage }}{{color "reset"}}{{end}}
{{- range $ix, $option := .PageEntries}}
{{- if eq $ix 0 }}  {{template "option" $.IterateOption $ix $option}}
{{- else }}{{"\n"}}  {{template "option" $.IterateOption $ix $option}}
{{- end}}
{{- end}}
{{- end}}`
)

var (
	envTemplateFuncsOnce sync.Once
	selectMetaMu         sync.RWMutex
	selectMetaStore      []selectOptionTemplateMeta
	selectMetaEnabled    bool
)

type selectOptionTemplateMeta struct {
	Group     string
	Name      string
	Path      string
	Active    bool
	First     bool
	GapBefore bool
}

func ensureMenuSelectTemplate() {
	survey.SelectQuestionTemplate = menuSelectTemplate

	// 注册模板函数
	core.TemplateFuncsWithColor["sub"] = func(a, b int) int { return a - b }
	core.TemplateFuncsNoColor["sub"] = func(a, b int) int { return a - b }
	core.TemplateFuncsWithColor["len"] = func(v interface{}) int {
		if slice, ok := v.([]interface{}); ok {
			return len(slice)
		}
		return 0
	}
	core.TemplateFuncsNoColor["len"] = func(v interface{}) int {
		if slice, ok := v.([]interface{}); ok {
			return len(slice)
		}
		return 0
	}

	selectMetaMu.Lock()
	selectMetaStore = nil
	selectMetaEnabled = false
	selectMetaMu.Unlock()
}

func ensureEnvSelectTemplate() {
	ensureEnvTemplateFuncs()
	survey.SelectQuestionTemplate = envSelectTemplate
}

func ensureEnvTemplateFuncs() {
	envTemplateFuncsOnce.Do(func() {
		core.TemplateFuncsWithColor["optionMeta"] = lookupSelectOptionMeta
		core.TemplateFuncsNoColor["optionMeta"] = lookupSelectOptionMeta
		core.TemplateFuncsWithColor["metaEnabled"] = isOptionMetaEnabled
		core.TemplateFuncsNoColor["metaEnabled"] = isOptionMetaEnabled
	})
}

func setSelectOptionMeta(meta []selectOptionTemplateMeta) func() {
	selectMetaMu.Lock()
	selectMetaStore = append([]selectOptionTemplateMeta(nil), meta...)
	selectMetaEnabled = len(meta) > 0
	selectMetaMu.Unlock()
	return func() {
		selectMetaMu.Lock()
		selectMetaStore = nil
		selectMetaEnabled = false
		selectMetaMu.Unlock()
	}
}

func lookupSelectOptionMeta(index int) selectOptionTemplateMeta {
	selectMetaMu.RLock()
	defer selectMetaMu.RUnlock()
	if index < 0 || index >= len(selectMetaStore) {
		return selectOptionTemplateMeta{}
	}
	return selectMetaStore[index]
}

func isOptionMetaEnabled() bool {
	selectMetaMu.RLock()
	defer selectMetaMu.RUnlock()
	return selectMetaEnabled
}
