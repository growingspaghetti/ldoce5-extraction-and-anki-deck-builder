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
	Str   string `xml:",innerxml"`
	Html  template.HTML
	Mp3   string
	Thumb template.HTML
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

	tmpl := "{{$t := .Title}}{{$h := .Head.Html}}{{$m := .Head.Mp3}}{{$img := .Head.Thumb}}{{$senses := .Senses}}{{range $i,$v := .Senses}}{{$t}}{{inc $i $senses}}\t[sound:media/{{$m}}]\t[sound:{{$v.Mp3}}]\t{{$img}}\t{{$v.Html}}<hr style='width:50%'>{{$h}}<hr>\n{{end}}"
	funcMap := template.FuncMap{
		"inc": func(i int, senses []sense) string {
			if len(senses) > 1 {
				return fmt.Sprintf("(%d)", i+1)
			} else {
				return ""
			}
		},
	}
	t, err := template.New("sample").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		panic(err)
	}

	for _, x := range xmls {
		fmt.Printf("\r adding to your anki file.. %s", x)
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
		for i := 0; i < len(et.Senses); i++ {
			thumbFiles := thumbMatcher.FindAllStringSubmatch(et.Head.Str+et.Senses[i].Str, -1)
			for _, t := range thumbFiles {
				et.Head.Thumb += template.HTML(fmt.Sprintf(`<img src="media/%s">`, t[1]))
			}
		}
		et.Head.Str = applyCleaning(et.Head.Str)
		for i := 0; i < len(et.Senses); i++ {
			et.Senses[i].Str = applyCleaning(et.Senses[i].Str)
		}
		mapStrToHtml(&et)
		out := new(bytes.Buffer)
		t.Execute(out, et)
		if _, err = anki.WriteString(out.String()); err != nil {
			panic(err)
		}
	}
}

func applyCleaning(s string) string {
	s = replacer.Replace(styleApplyer.Replace(s))
	s = selfClosingMatcher.ReplaceAllString(s, "")
	s = asFilterMatcher.ReplaceAllString(s, "")
	return idCleaner.ReplaceAllString(s, "")
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
	selfClosingMatcher  = regexp.MustCompile(`<span [^>]+?/>"`)
	asFilterMatcher     = regexp.MustCompile(`as_filter="[^"]+"`)
	thumbMatcher        = regexp.MustCompile(`thumb="(thumbnail/[^"]+?)"`)
	idCleaner           = regexp.MustCompile(`id="[^"]+"`)
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
	styleMap = append(styleMap, "<INFLX", "<INFLX style='display:none'")
	styleMap = append(styleMap, "<Refs", "<Refs style='display:none'")
	styleMap = append(styleMap, "<ACTIV", "<ACTIV style='color:white;background-color:#839a5c;font-weight:bold;padding:6px;'")
	styleMap = append(styleMap, makePaddingAttriubute("HYPHENATION")...)
	styleMap = append(styleMap, "<EXAMPLE", "<EXAMPLE style='color:#0000A0;text-align:left;padding:6px;'")
	styleMap = append(styleMap, makePaddingAttriubute("SYN")...)
	styleMap = append(styleMap, "<REFHWD", "<REFHWD style='color:#585800'")
	styleMap = append(styleMap, makePaddingAttriubute("BASE")...)
	styleApplyer = strings.NewReplacer(styleMap...)

	replaceMap := make([]string, 0)
	//replaceMap = append(replaceMap, makeClosingTag("HWD", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("BASE", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("POS", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("HYPHENATION", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("INFLX", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("GRAM", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PRON", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PronCode", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("SYN", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("REGISTERLAB", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("SE_EntryAssets", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("LEXVAR", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("PROPFORMPREP", "p")...)
	replaceMap = append(replaceMap, makeClosingTag("GramExa", "p")...)
	replaceMap = append(replaceMap, makeClosingTag("PROPFORM", "p")...)
	replaceMap = append(replaceMap, makeClosingTag("OPP", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("EXPR", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("ACTIV", "div")...)
	replaceMap = append(replaceMap, makeClosingTag("EXAMPLE", "p")...)
	//replaceMap = append(replaceMap, makeClosingTag("DEF", "div")...)
	replaceMap = append(replaceMap, makeClosingTag("REFHWD", "b")...)
	replaceMap = append(replaceMap, makeClosingTag("LEXUNIT", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("COLLOINEXA", "span")...)
	replaceMap = append(replaceMap, makeClosingTag("RELATEDWD", "span")...)
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
