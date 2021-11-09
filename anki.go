package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
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

	tmpl := "{{$t := .Title}}{{$h := .Head.Html}}{{$m := .Head.Mp3}}{{range $i,$v := .Senses}}<h2>{{$t}}\t[sount:media/{{$m}}]\t</h2>{{$v.Html}}<hr>{{$h}}\n{{end}}"
	t, err := template.New("sample").Parse(tmpl)
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
			if audioFiles != nil {
				et.Senses[i].Mp3 = concatenateMp3s(audioFiles)
			}
		}
		//et.EntryAsset.Str = "" //formatEntryAsset(p.EntryAsset.Str)
		et.Head.Str = replacer.Replace(et.Head.Str)
		for i := 0; i < len(et.Senses); i++ {
			et.Senses[i].Str = replacer.Replace(et.Senses[i].Str)
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
	}
	dest := joiner.Reader()
	return ""
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
)

func makeClosingTag(from, to string) []string {
	return []string{"<" + from, "<" + to, "</" + from, "</" + to}
}

func init() {
	replaceMap := make([]string, 0)
	replaceMap = append(replaceMap, makeClosingTag("HWD", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("BASE", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("POS", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("HYPHENATION", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("INFLX", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GRAM", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PRON", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PronCode", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("REGISTERLAB", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("SE_EntryAssets", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PROPFORMPREP", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GramExa", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("OPP", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("EXPR", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("ACTIV", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("EXAMPLE", "div")...)
	replaceMap = append(replaceMap, makeClosingTag("DEF", "p")...)
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
	replacer = strings.NewReplacer(replaceMap...)
}
