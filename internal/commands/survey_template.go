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
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}
    {{- if ne $desc ""}}
    {{color "cyan"}}- {{$desc}}{{color "reset"}}
    {{- else}}{{color "reset"}}
    {{- end}}
{{end}}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}
{{- else}}
{{- if .FilterMessage}}
{{"\n"}}{{color "yellow"}}当前过滤:{{ .FilterMessage }}{{color "reset"}}
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
{{- if $meta.GapBefore }}{{"\n\n"}}{{else}}{{"\n\n"}}{{end}}  {{color "cyan"}}{{ $meta.Group }}{{color "reset"}}
        {{- end }}
        {{- $display := printf "%s (%s)" $meta.Name $meta.Path -}}
        {{- if eq .SelectedIndex .CurrentIndex }}
   {{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }}{{color "reset"}}  {{$display}}
        {{- else if $meta.Active }}
    {{color "red"}}+{{color "reset"}} {{$display}}
        {{- else }}
      {{$display}}
        {{- end }}
    {{- end }}
{{- end }}
{{- define "defaultOption"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}{{ if ne ($.GetDescription .CurrentOpt) "" }} - {{color "cyan"}}{{ $.GetDescription .CurrentOpt }}{{end}}
    {{- color "reset"}}
{{- end }}
{{- define "option"}}
    {{- if metaEnabled }}{{template "envOption" .}}{{else}}{{template "defaultOption" .}}{{end}}
{{- end }}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else}}
{{- if .FilterMessage }}{{"\n"}}{{"\n"}}{{color "cyan"}}{{ .FilterMessage }}{{color "reset"}}{{"\n"}}{{end}}
  {{- range $ix, $option := .PageEntries}}
    {{- template "option" $.IterateOption $ix $option}}
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
