package gemini

import "fmt"

type GeminiBody struct {
	buf string
}

func NewBody() *GeminiBody {
	return &GeminiBody{}
}

func (body *GeminiBody) AddHeader(str string) {
	body.buf += fmt.Sprintf("# %s\n\n", str)
}

func (body *GeminiBody) AddTextLine(str string) {
	body.buf += str + "\n\n"
}

func (body *GeminiBody) AddLinkLine(url, text string) {
	body.buf += fmt.Sprintf("=> %s %s\n\n", url, text)
}

func (body *GeminiBody) AddRaw(data string) {
	body.buf += data
}
