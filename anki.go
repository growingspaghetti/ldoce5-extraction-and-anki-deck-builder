package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hyacinthus/mp3join"
)

type entryAsset struct {
	Str  string `xml:",innerxml"`
	Html template.HTML
}

type head struct {
	Str  string `xml:",innerxml"`
	Html template.HTML
	Mp3  string
}

type sense struct {
	Str  string `xml:",innerxml"`
	Html template.HTML
	Mp3  string
}

type entry struct {
	EntryAsset entryAsset `xml:"SE_EntryAssets"`
	Head       head       `xml:"Head"`
	Senses     []sense    `xml:"Sense"`
	Title      string
}

func compileAnkiDeck() {
	pattern := "text/**/*.xml"
	xmls, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	anki, err := os.OpenFile("anki.html", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer anki.Close()

	tmpl := "{{$t := .Title}}{{$h := .Head.Html}}{{$m := .Head.Mp3}}{{range $i,$v := .Senses}}<h2>{{$t}}[{{inc $i}}]\t[sount:media/{{$m}}]\t</h2>\t[sound:{{$v.Mp3}}]\t{{$v.Html}}<hr>{{$h}}\n{{end}}"
	funcMap := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}
	t, err := template.New("sample").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		panic(err)
	}

	for _, x := range xmls {
		fmt.Println(x)
		b, err := ioutil.ReadFile(x)
		if err != nil {
			panic(err)
		}
		et := entry{}
		xml.Unmarshal(b, &et)
		et.Title = titleMatcher.FindStringSubmatch(et.EntryAsset.Str)[1]
		titleAudio := titleAudioMatcher.FindStringSubmatch(et.Head.Str)
		if titleAudio != nil {
			et.Head.Mp3 = titleAudio[1]
		}
		for i := 0; i < len(et.Senses); i++ {
			audioFiles := exampleAudioMatcher.FindAllStringSubmatch(et.Senses[i].Str, -1)
			switch len(audioFiles) {
			case 0:
			case 1:
				et.Senses[i].Mp3 = "media/" + audioFiles[0][1]
			default:
				et.Senses[i].Mp3 = concatenateMp3s(audioFiles)
			}
		}
		//et.EntryAsset.Str = "" //formatEntryAsset(p.EntryAsset.Str)
		et.Head.Str = replacer.Replace(styleApplyer.Replace(et.Head.Str))
		for i := 0; i < len(et.Senses); i++ {
			et.Senses[i].Str = replacer.Replace(styleApplyer.Replace(et.Senses[i].Str))
		}

		// if exampleAudioFiles != nil {
		// 	et.Head.Mp3 = titleAudio[1]
		// }

		mapStrToHtml(&et)
		//fmt.Printf("%#v\n", et)
		out := new(bytes.Buffer)
		t.Execute(out, et)
		if _, err = anki.WriteString(out.String()); err != nil {
			panic(err)
		}
	}
}

func concatenateMp3s(files [][]string) string {
	sb := new(strings.Builder)
	joiner := mp3join.New()
	for _, file := range files {
		f, err := os.Open("media/" + file[1])
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := joiner.Append(f); err != nil {
			panic(err)
		}
		sb.WriteString(strings.TrimSuffix(filepath.Base(file[1]), ".mp3"))
	}
	hash := md5.Sum([]byte(sb.String()))
	dst := "media/" + hex.EncodeToString(hash[:]) + ".mp3"
	concat, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer concat.Close()
	if _, err := io.Copy(concat, joiner.Reader()); err != nil {
		panic(err)
	}
	return dst
}

func mapStrToHtml(et *entry) {
	et.EntryAsset.Html = template.HTML(et.EntryAsset.Str)
	et.Head.Html = template.HTML(et.Head.Str)
	for i := 0; i < len(et.Senses); i++ {
		et.Senses[i].Html = template.HTML(et.Senses[i].Str)
	}
}

var (
	titleMatcher        = regexp.MustCompile(`<hwd>(.*?)</hwd>`)
	titleAudioMatcher   = regexp.MustCompile(`resource="GB_HWD_PRON".*?topic="(.*?\.mp3)"`)
	exampleAudioMatcher = regexp.MustCompile(`resource="EXA_PRON".*?topic="(.*?\.mp3)"`)
	replacer            *strings.Replacer
	styleApplyer        *strings.Replacer
)

func makeClosingTag(from, to string) []string {
	return []string{"<" + from, "<" + to, "</" + from, "</" + to}
}

func makePaddingAttriubute(tag string) []string {
	return []string{"<" + tag, "<" + tag + " style='padding:4px'"}
}

func init() {
	styleMap := make([]string, 0)
	styleMap = append(styleMap, "<HWD", "<HWD style='color:#585800;'")
	styleMap = append(styleMap, "class=\"sensenum\"", "style='padding:5px;font-weight:bold;'")

	styleMap = append(styleMap, "<EXAMPLE", "<EXAMPLE style='color:#0000A0;text-align:left;padding:6px;'")

	styleMap = append(styleMap, makePaddingAttriubute("BASE")...)
	styleApplyer = strings.NewReplacer(styleMap...)

	replaceMap := make([]string, 0)
	replaceMap = append(replaceMap, makeClosingTag("HWD", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("BASE", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("POS", "b")...)
	replaceMap = append(replaceMap, makePaddingAttriubute("HYPHENATION")...)
	replaceMap = append(replaceMap, makeClosingTag("HYPHENATION", "span")...)
	replaceMap = append(replaceMap, makePaddingAttriubute("INFLX")...)
	replaceMap = append(replaceMap, makeClosingTag("INFLX", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GRAM", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PRON", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PronCode", "span")...)
	replaceMap = append(replaceMap, makePaddingAttriubute("SYN")...)
	replaceMap = append(replaceMap, makeClosingTag("SYN", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("REGISTERLAB", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("SE_EntryAssets", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PROPFORMPREP", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GramExa", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("OPP", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("EXPR", "b")...)
	replaceMap = append(replaceMap, "<ACTIV", "<ACTIV style='color:#0000A0;font-weight:bold;padding:6px;'")
	replaceMap = append(replaceMap, makeClosingTag("ACTIV", "div")...)
	replaceMap = append(replaceMap, makeClosingTag("EXAMPLE", "div")...)
	replaceMap = append(replaceMap, makeClosingTag("DEF", "div")...)
	replaceMap = append(replaceMap, "<REFHWD", "<REFHWD style='color:#585800'")
	replaceMap = append(replaceMap, makeClosingTag("REFHWD", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("LEXUNIT", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("COLLOINEXA", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("RELATEDWD", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("Refs", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GLOSS", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("FREQ", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("ORTHVAR", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("HOMNUM", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("Audio", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("Variant", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("BREQUIV", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GEO", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("LINKWORD", "span")...)
	replacer = strings.NewReplacer(replaceMap...)
}
