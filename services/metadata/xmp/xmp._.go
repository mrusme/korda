package xmp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
)

const (
	NSRDF       = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	NSXMP       = "http://ns.adobe.com/xap/1.0/"
	NSDC        = "http://purl.org/dc/elements/1.1/"
	NSEXIF      = "http://ns.adobe.com/exif/1.0/"
	NSEXIFEX    = "http://cipa.jp/exif/1.0/"
	NSTIFF      = "http://ns.adobe.com/tiff/1.0/"
	NSPhotoshop = "http://ns.adobe.com/photoshop/1.0/"
	NSGCamera   = "http://ns.google.com/photos/1.0/camera/"
	NSContainer = "http://ns.google.com/photos/1.0/container/"
	NSItem      = "http://ns.google.com/photos/1.0/container/item/"
	NSGPano     = "http://ns.google.com/photos/1.0/panorama/"
	NSLightroom = "http://ns.adobe.com/lightroom/1.0/"
	NSDigiKam   = "http://www.digikam.org/ns/1.0/"
)

const maxDepth = 64

var ErrNotXMP = errors.New("not an XMP packet")

type Element struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Children []*Element
	Text     string
}

type Packet struct {
	descriptions []*Element
}

type Item struct {
	Mime     string
	Semantic string
	Length   int64
	Padding  int64
}

func Parse(data []byte) (*Packet, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	root := &Element{}
	stack := []*Element{root}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if len(root.Children) == 0 {
				return nil, err
			}
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) > maxDepth {
				return nil, ErrNotXMP
			}
			el := &Element{Name: t.Name, Attrs: t.Attr}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, el)
			stack = append(stack, el)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			stack[len(stack)-1].Text += string(t)
		}
	}

	p := &Packet{}
	var walk func(el *Element)
	walk = func(el *Element) {
		if el.Name.Space == NSRDF && el.Name.Local == "Description" {
			p.descriptions = append(p.descriptions, el)
		}
		for _, child := range el.Children {
			walk(child)
		}
	}
	walk(root)

	if len(p.descriptions) == 0 {
		return nil, ErrNotXMP
	}

	return p, nil
}

func (el *Element) attr(space string, local string) (string, bool) {
	for _, a := range el.Attrs {
		if a.Name.Space == space && a.Name.Local == local {
			return a.Value, true
		}
	}

	return "", false
}

func (el *Element) child(space string, local string) *Element {
	for _, c := range el.Children {
		if c.Name.Space == space && c.Name.Local == local {
			return c
		}
	}

	return nil
}

func listItems(el *Element) ([]*Element, bool) {
	for _, kind := range []string{"Alt", "Seq", "Bag"} {
		if container := el.child(NSRDF, kind); container != nil {
			var items []*Element
			for _, c := range container.Children {
				if c.Name.Space == NSRDF && c.Name.Local == "li" {
					items = append(items, c)
				}
			}
			return items, true
		}
	}

	return nil, false
}

func (p *Packet) find(space string, local string) (*Element, string, bool) {
	for _, d := range p.descriptions {
		if v, ok := d.attr(space, local); ok {
			return nil, v, true
		}
		if c := d.child(space, local); c != nil {
			return c, strings.TrimSpace(c.Text), true
		}
	}

	return nil, "", false
}

func (p *Packet) String(space string, local string) (string, bool) {
	el, value, ok := p.find(space, local)
	if !ok {
		return "", false
	}
	if el == nil {
		return value, true
	}
	if items, isList := listItems(el); isList {
		if len(items) == 0 {
			return "", false
		}
		for _, item := range items {
			if lang, _ := item.attr("http://www.w3.org/XML/1998/namespace", "lang"); lang == "x-default" {
				return strings.TrimSpace(item.Text), true
			}
		}
		return strings.TrimSpace(items[0].Text), true
	}

	return value, true
}

func (p *Packet) List(space string, local string) ([]string, bool) {
	el, value, ok := p.find(space, local)
	if !ok {
		return nil, false
	}
	if el == nil {
		return []string{value}, true
	}
	items, isList := listItems(el)
	if !isList {
		return []string{value}, true
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.TrimSpace(item.Text))
	}

	return out, true
}

func (p *Packet) Int(space string, local string) (int64, bool) {
	s, ok := p.String(space, local)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n, err == nil
}

func (p *Packet) Float(space string, local string) (float64, bool) {
	s, ok := p.String(space, local)
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f, err == nil
}

func (p *Packet) Directory() []Item {
	el, _, ok := p.find(NSContainer, "Directory")
	if !ok || el == nil {
		return nil
	}
	lis, isList := listItems(el)
	if !isList {
		return nil
	}

	var out []Item
	for _, li := range lis {
		holder := li
		if item := li.child(NSContainer, "Item"); item != nil {
			holder = item
		}

		var it Item
		get := func(local string) string {
			if v, ok := holder.attr(NSItem, local); ok {
				return v
			}
			if c := holder.child(NSItem, local); c != nil {
				return strings.TrimSpace(c.Text)
			}
			return ""
		}
		it.Mime = get("Mime")
		it.Semantic = get("Semantic")
		it.Length, _ = strconv.ParseInt(get("Length"), 10, 64)
		it.Padding, _ = strconv.ParseInt(get("Padding"), 10, 64)
		out = append(out, it)
	}

	return out
}
