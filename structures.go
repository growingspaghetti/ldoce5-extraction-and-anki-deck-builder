package main

import (
	"html/template"
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

type phrasalverb struct {
	Head   head    `xml:"Head"`
	Senses []sense `xml:"Sense"`
	Title  template.HTML
}

type entry struct {
	EntryAsset   entryAsset    `xml:"SE_EntryAssets"`
	Head         head          `xml:"Head"`
	Senses       []sense       `xml:"Sense"`
	PhrasalVerbs []phrasalverb `xml:"PhrVbEntry"`
	Title        string
}

type doc interface {
	heading() string
	setStrHeading(string)
	setHtmlHeading(template.HTML)
	useHeading() bool
	setTitleAudio(string)
	senses() []sense
}

func (p *phrasalverb) useHeading() bool {
	return false
}

func (p *phrasalverb) heading() string {
	return p.Head.Str
}

func (p *phrasalverb) setHtmlHeading(h template.HTML) {
	p.Head.Html = h
}

func (p *phrasalverb) setStrHeading(s string) {
	p.Head.Str = s
}

func (p *phrasalverb) setTitleAudio(a string) {
	p.Head.Mp3 = a
}

func (p *phrasalverb) senses() []sense {
	return p.Senses
}

func (e *entry) useHeading() bool {
	return true
}

func (e *entry) heading() string {
	return e.Head.Str
}

func (e *entry) setHtmlHeading(h template.HTML) {
	e.Head.Html = h
}

func (e *entry) setStrHeading(s string) {
	e.Head.Str = s
}

func (e *entry) setTitleAudio(a string) {
	e.Head.Mp3 = a
}

func (e *entry) senses() []sense {
	return e.Senses
}
