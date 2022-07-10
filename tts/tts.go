package tts

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Lang string

const (
	ZhCN Lang = "zh-CN"
	EnUS      = "en-US"
)

var voiceMap = map[Lang]string{
	ZhCN: "zh-CN-XiaoxiaoNeural",
	EnUS: "en-US-JennyNeural",
}

const (
	endpoint = "https://%s.tts.speech.microsoft.com/cognitiveservices/v1"
)

type TTSer interface {
	TTS(text string, lang Lang) ([]byte, error)
}

type ttserImpl struct {
	key    string
	region string
}

func New(key string, region string) TTSer {
	return &ttserImpl{key: key, region: region}
}

func (t *ttserImpl) TTS(text string, lang Lang) ([]byte, error) {
	voice, ok := voiceMap[lang]
	if !ok {
		return nil, errors.New("invalid language value")
	}
	ssml := `<speak version="1.0" xml:lang="` + string(lang) + `">
<voice xml:lang="` + string(lang) + `" xml:gender="Female" name="` + voice + `">` + text + `</voice>
</speak>`

	var buf bytes.Buffer
	buf.WriteString(ssml)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(endpoint, t.region), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Ocp-Apim-Subscription-Key", t.key)
	req.Header.Add("Content-Type", "application/ssml+xml")
	req.Header.Add("X-Microsoft-OutputFormat", "audio-16khz-128kbitrate-mono-mp3")
	req.Header.Add("User-Agent", "curl")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("TTS service error with code: %v", res.StatusCode)
	}

	return io.ReadAll(res.Body)
}
