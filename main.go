package main

import (
	"bytes"
	"flag"
	"github.com/peterbourgon/mergemap"
	"github.com/russross/blackfriday"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

var (
	FrontSeparator = []byte("---\n")
)

var (
	sourceDir = flag.String("source", "src", "path to site source (input)")
	targetDir = flag.String("target", "tgt", "path to site target (output)")
	globalKey = flag.String("global.key", "files", "template node name for global info")
)

func init() {
	log.SetFlags(0)
	flag.Parse()

	var err error
	for _, s := range []*string{sourceDir, targetDir} {
		if *s, err = filepath.Abs(*s); err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}
}

func main() {
	m := map[string]interface{}{}
	s := NewStack()
	filepath.Walk(*sourceDir, gather(s, m))
	s.Add("", map[string]interface{}{*globalKey: m})

	filepath.Walk(*sourceDir, transform(s))
}

// splitMetadata splits the input buffer on FrontSeparator. It returns a byte-
// slice suitable for unmarshaling into metadata, if it exists, and the
// remainder of the input buffer.
func splitMetadata(buf []byte) ([]byte, []byte) {
	split := bytes.SplitN(buf, FrontSeparator, 2)
	if len(split) == 2 {
		return split[0], split[1]
	}
	return []byte{}, buf
}

func gather(s StackReadWriter, m map[string]interface{}) filepath.WalkFunc {
	return func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil // descend
		}
		switch filepath.Ext(path) {
		case ".json":
			metadata := mustJSON(mustRead(path))
			s.Add(filepath.Dir(path), metadata)
			log.Printf("%s gathered (%d element(s))", path, len(metadata))

		case ".html":
			fullMetadata := map[string]interface{}{
				"source": diffPath(*sourceDir, path),
				"target": diffPath(*targetDir, targetFor(path, filepath.Ext(path))),
				"url":    "/" + diffPath(*targetDir, targetFor(path, filepath.Ext(path))),
			}
			metadataBuf, _ := splitMetadata(mustRead(path))
			if len(metadataBuf) > 0 {
				fileMetadata := mustJSON(metadataBuf)
				s.Add(path, fileMetadata)
			}
			fullMetadata = mergemap.Merge(s.Get(path), fullMetadata)
			splatInto(m, diffPath(*sourceDir, path), fullMetadata)
			log.Printf("%s gathered (%d element(s))", path, len(fullMetadata))

		case ".md":
			fullMetadata := map[string]interface{}{
				"source": diffPath(*sourceDir, path),
				"target": diffPath(*targetDir, targetFor(path, ".html")),
				"url":    "/" + diffPath(*targetDir, targetFor(path, ".html")),
			}
			metadataBuf, _ := splitMetadata(mustRead(path))
			if len(metadataBuf) > 0 {
				fileMetadata := mustJSON(metadataBuf)
				s.Add(path, fileMetadata)
			}
			fullMetadata = mergemap.Merge(s.Get(path), fullMetadata)
			splatInto(m, diffPath(*sourceDir, path), fullMetadata)
			log.Printf("%s gathered (%d element(s))", path, len(fullMetadata))

		default:
			log.Printf("%s ignored for gathering", path)

		}
		return nil
	}
}

func transform(s StackReader) filepath.WalkFunc {
	return func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil // descend
		}
		switch filepath.Ext(path) {
		case ".json":
			log.Printf("%s ignored for transformation", path)

		case ".html":
			_, contentBuf := splitMetadata(mustRead(path))
			metadata := s.Get(path)
			outputBuf := renderTemplate(path, contentBuf, metadata)
			dst := targetFor(path, filepath.Ext(path))
			mustWrite(dst, outputBuf)
			log.Printf("%s transformed to %s", path, dst)

		case ".md":
			_, contentBuf := splitMetadata(mustRead(path))

			// render the markdown, and put it into the 'content' key of an
			// interstitial metadata, to be fed to the template renderer
			metadata := mergemap.Merge(s.Get(path), map[string]interface{}{
				"content": renderMarkdown(contentBuf),
			})

			// render the complete html output according to the template
			outputBuf := renderTemplate(path, mustTemplate(s, path), metadata)

			// write
			dst := targetFor(path, ".html")
			mustWrite(dst, outputBuf)
			log.Printf("%s transformed to %s", path, dst)

		case ".source", ".template":
			log.Printf("%s ignored for transformation", path)

		default:
			dst := filepath.Join(*targetDir, diffPath(*sourceDir, path))
			mustCopy(targetFor(path, filepath.Ext(path)), path)
			log.Printf("%s transformed to %s verbatim", path, dst)
		}
		return nil
	}
}

func renderTemplate(path string, input []byte, metadata map[string]interface{}) []byte {
	funcMap := template.FuncMap{
		"importcss": func(filename string) template.CSS {
			return template.CSS(mustRead(filepath.Join(filepath.Dir(path), filename)))
		},
		"importjs": func(filename string) template.JS {
			return template.JS(mustRead(filepath.Join(filepath.Dir(path), filename)))
		},
		"importhtml": func(filename string) template.HTML {
			return template.HTML(mustRead(filepath.Join(filepath.Dir(path), filename)))
		},
	}
	tmpl, err := template.New("x").Funcs(funcMap).Parse(string(input))
	if err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
	output := bytes.Buffer{}
	if err := tmpl.Execute(&output, metadata); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
	return output.Bytes()
}

func renderMarkdown(input []byte) []byte {
	htmlOptions := 0
	htmlOptions = htmlOptions | blackfriday.HTML_GITHUB_BLOCKCODE
	htmlOptions = htmlOptions | blackfriday.HTML_USE_SMARTYPANTS
	title, css := "", ""
	htmlRenderer := blackfriday.HtmlRenderer(htmlOptions, title, css)

	mdOptions := 0
	mdOptions = mdOptions | blackfriday.EXTENSION_FENCED_CODE

	return blackfriday.Markdown(input, htmlRenderer, mdOptions)
}
