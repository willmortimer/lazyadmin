package tasks

import (
	"bytes"
	"text/template"
)

func executeTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("summary").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
