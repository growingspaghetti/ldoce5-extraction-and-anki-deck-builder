package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
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

type mp3cater struct {
	sb *strings.Builder
}

func compileAnkiDeck() {
	textBody()
	fmt.Println()
	fmt.Println("Completed")
}

func textBody() {
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
	t := "{{$t := .Title}}{{$h := .Head.Html}}{{$m := .Head.Mp3}}{{$img := .Head.Thumb}}{{$senses := .Senses}}{{range $i,$v := .Senses}}{{$t}}{{inc $i $senses}}\t[sound:media/{{$m}}]\t[sound:{{$v.Mp3}}]\t{{$img}}\t{{$v.Html}}<hr style='width:50%'>{{$h}}<hr>\n{{end}}"
	funcMap := template.FuncMap{
		"inc": func(i int, senses []sense) string {
			if len(senses) > 1 {
				return fmt.Sprintf("(%d)", i+1)
			} else {
				return ""
			}
		},
	}
	tmpl, err := template.New("anki-template").Funcs(funcMap).Parse(t)
	if err != nil {
		panic(err)
	}
	for i, xml := range xmls {
		if i%100 == 0 {
			fmt.Printf("\r adding to your anki file.. %s", xml)
		}
		addXmlToAnki(xml, anki, tmpl)
	}
}

func addXmlToAnki(docFile string, anki *os.File, tmpl *template.Template) {
	doc, err := ioutil.ReadFile(docFile)
	if err != nil {
		panic(err)
	}
	ent := entry{}
	xml.Unmarshal(doc, &ent)
	ent.Title = titleMatcher.FindStringSubmatch(ent.EntryAsset.Str)[1]
	addMp3s(&ent)
	addImages(&ent)
	formatXml(&ent)
	mapStrToHtml(&ent)
	out := new(bytes.Buffer)
	tmpl.Execute(out, ent)
	if _, err = anki.WriteString(out.String()); err != nil {
		panic(err)
	}
}

func formatXml(ent *entry) {
	ent.Head.Str = applyCleaning(ent.Head.Str)
	for i := 0; i < len(ent.Senses); i++ {
		ent.Senses[i].Str = applyCleaning(ent.Senses[i].Str)
	}
}

func addMp3s(ent *entry) {
	titleAudio := titleAudioMatcher.FindStringSubmatch(ent.Head.Str)
	if titleAudio != nil {
		ent.Head.Mp3 = titleAudio[1]
	}
	for i := 0; i < len(ent.Senses); i++ {
		audioFiles := exampleAudioMatcher.FindAllStringSubmatch(ent.Senses[i].Str, -1)
		switch len(audioFiles) {
		case 0:
		case 1:
			ent.Senses[i].Mp3 = "media/" + audioFiles[0][1]
		default:
			ent.Senses[i].Mp3 = mp3cat.concatenateMp3s(audioFiles)
		}
	}
}

func addImages(ent *entry) {
	var imgs string
	thumbFiles := thumbMatcher.FindAllStringSubmatch(ent.Head.Str+ent.Head.Str, -1)
	for _, t := range thumbFiles {
		if !strings.HasSuffix(imgs, t[1]+"\">") {
			imgs += fmt.Sprintf(`<img src="media/%s">`, t[1])
		}
	}
	for i := 0; i < len(ent.Senses); i++ {
		thumbFiles := thumbMatcher.FindAllStringSubmatch(ent.Senses[i].Str, -1)
		for _, t := range thumbFiles {
			if !strings.HasSuffix(imgs, t[1]+"\">") {
				imgs += fmt.Sprintf(`<img src="media/%s">`, t[1])
			}
		}
	}
	ent.Head.Thumb += template.HTML(imgs)
}

func applyCleaning(s string) string {
	s = replacer.Replace(styleApplyer.Replace(s))
	s = selfClosingMatcher.ReplaceAllString(s, "")
	s = asFilterMatcher.ReplaceAllString(s, "")
	return idCleaner.ReplaceAllString(s, "")
}

func (m *mp3cater) concatenateMp3s(files [][]string) string {
	m.sb.Reset()
	joiner := mp3join.New()
	for _, file := range files {
		f, err := os.Open("media/" + file[1])
		if err != nil {
			// panic: open media/p008_001/p008_00155/p008-001559957.mp3: no such file or directory
			if errors.Is(err, os.ErrNotExist) {
				return ""
			}
			panic(err)
		}
		defer f.Close()
		if err := joiner.Append(f); err != nil {
			panic(err)
		}
		m.sb.WriteString(strings.TrimSuffix(filepath.Base(file[1]), ".mp3"))
	}
	hash := md5.Sum([]byte(m.sb.String()))
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
	mp3cat              mp3cater
)

func init() {
	mp3cat = mp3cater{
		sb: new(strings.Builder),
	}
}

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
