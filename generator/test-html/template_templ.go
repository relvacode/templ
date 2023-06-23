// Code generated by templ@(devel) DO NOT EDIT.

package testhtml

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func render(p person) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_1 := templ.GetChildren(ctx)
		if var_1 == nil {
			var_1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<div><h1>")
		if err != nil {
			return err
		}
		var var_2 string = p.name
		_, err = templBuffer.WriteString(templ.EscapeString(var_2))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</h1><div style=\"font-family: &#39;sans-serif&#39;\" id=\"test\" data-contents=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(`something with "quotes" and a <tag>`))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\"><div>")
		if err != nil {
			return err
		}
		var_3 := `email:`
		_, err = templBuffer.WriteString(var_3)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("<a href=\"")
		if err != nil {
			return err
		}
		var var_4 templ.SafeURL = templ.URL("mailto: " + p.email)
		_, err = templBuffer.WriteString(templ.EscapeString(string(var_4)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\">")
		if err != nil {
			return err
		}
		var var_5 string = p.email
		_, err = templBuffer.WriteString(templ.EscapeString(var_5))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a></div></div></div><hr")
		if err != nil {
			return err
		}
		if true {
			_, err = templBuffer.WriteString(" noshade")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("><hr optionA")
		if err != nil {
			return err
		}
		if true {
			_, err = templBuffer.WriteString(" optionB")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString(" optionC=\"other\"")
		if err != nil {
			return err
		}
		if false {
			_, err = templBuffer.WriteString(" optionD")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("><hr noshade>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = io.Copy(w, templBuffer)
		}
		return err
	})
}
