package global

import (
	"html/template"
	funcs "xiaohuAdmin/function"
)

func GetViewFuncMap() template.FuncMap {
	return template.FuncMap{
		"safe": func(str string) template.HTML {
			return template.HTML(str)
		},
		"p": func(data any) template.HTML {
			return template.HTML("<script>var json = " + funcs.JsonEncodeStr(data) + ";json = JSON.stringify(json, null, 4);json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');var pre = document.createElement('pre');pre.style.backgroundColor = '#282c34';pre.style.color = '#ffffff';pre.style.padding = '20px';pre.style.borderRadius = '4px';pre.style.fontSize = '14px';pre.style.overflowX = 'auto';pre.style.whiteSpace = 'pre-wrap';pre.style.wordWrap = 'break-word';pre.textContent = json;document.body.appendChild(pre);</script>")
		},
	}
}
