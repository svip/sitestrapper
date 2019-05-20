package strapper

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"gitlab.com/golang-commonmark/markdown"
	"gopkg.in/yaml.v2"
)

type sitemapPage struct {
	Path      string
	Subpages  []sitemapPage
	ID        string
	Title     string
	LinkTitle string
	Template  string
	Content   string
}

func (p sitemapPage) PublicPath() string {
	s := strings.Replace(p.Path, "site/", "", 1)
	s = strings.Replace(s, filepath.Ext(s), "", 1)
	s = fmt.Sprintf("/%s.html", s)
	return s
}

type sitemapMedia struct {
	Type string
	Path string
	Name string
}

type sitemapTemplate struct {
	Path string
}

type sitemapStructure struct {
	Title        string
	Media        []sitemapMedia
	BasePagePath string
	Pages        []sitemapPage
	Templates    []sitemapTemplate
	Categories   map[string][]string
}

func (s sitemapStructure) MediaMap() map[string]sitemapMedia {
	m := make(map[string]sitemapMedia)
	for _, media := range s.Media {
		m[media.Name] = media
	}
	return m
}

type SiteStrapper struct {
	inputDirectory  string
	outputDirectory string
	sitemap         sitemapStructure
	templates       map[string]*siteTemplate
}

func NewSiteStrapper(inputDirectory string, outputDirectory string) *SiteStrapper {
	return &SiteStrapper{
		inputDirectory:  inputDirectory,
		outputDirectory: outputDirectory,
	}
}

func (ss *SiteStrapper) fillInMissingSitemap() (err error) {
	newSitemap := ss.sitemap
	if len(newSitemap.Media) == 0 {
		// no media files, let's see if there are any
		dir := path.Join(ss.inputDirectory, "media")
		err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}
			if info.IsDir() {
				return nil
			}
			ext := filepath.Ext(filePath)
			var fileType string
			switch ext {
			case ".css":
				fileType = "stylesheet"
			case ".js":
				fileType = "script"
			case ".png", ".jpg", ".jpeg", ".gif":
				fileType = "image"
			}

			if len(fileType) == 0 {
				return nil // just continue
			}
			filePath = strings.Replace(filePath, ss.inputDirectory, "", 1)
			name := path.Base(filePath)
			newSitemap.Media = append(newSitemap.Media, sitemapMedia{Type: fileType, Path: filePath, Name: name})
			return nil
		})
		if err != nil {
			return err
		}
	}
	if len(newSitemap.Pages) == 0 {
		dir := path.Join(ss.inputDirectory, "site")
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".html" {
				return nil
			}
			path = strings.Replace(path, ss.inputDirectory, "", 1)
			newSitemap.Pages = append(newSitemap.Pages, sitemapPage{Path: path})
			return nil
		})
		if err != nil {
			return err
		}
		newSitemap.BasePagePath = "site/"
	}
	if len(newSitemap.Templates) == 0 {
		dir := path.Join(ss.inputDirectory, "templates")
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}
			if info.IsDir() {
				return nil
			}
			path = strings.Replace(path, ss.inputDirectory, "", 1)
			newSitemap.Templates = append(newSitemap.Templates, sitemapTemplate{Path: path})
			return nil
		})
		if err != nil {
			return err
		}
	}
	ss.sitemap = newSitemap
	return nil
}

func (ss *SiteStrapper) makeSitemap() (err error) {
	sitemapFilename := path.Join(ss.inputDirectory, "sitemap.yaml")

	f, err := os.Open(sitemapFilename)
	if err != nil {
		// if no sitemap, guess from file structure
		return ss.fillInMissingSitemap()
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := yaml.Unmarshal(b, &ss.sitemap); err != nil {
		return errors.WithStack(err)
	}

	return ss.fillInMissingSitemap()
}

type templateHeader struct {
	Name string
}

type siteTemplate struct {
	Header templateHeader
	T      *template.Template
}

func (ss *SiteStrapper) parseLink(original string, text string, link string) string {
	s := strings.SplitN(link, ":", 2)
	if len(s) != 2 {
		return original
	}
	prefix := s[0]
	id := s[1]
	switch prefix {
	case "id":
		var page sitemapPage
		found := false
		for _, p := range ss.sitemap.Pages {
			if p.ID == id {
				page = p
				found = true
				break
			}
		}
		if !found {
			return original
		}
		return fmt.Sprintf("[%s](%s)", text, page.PublicPath())
	case "image":
		return fmt.Sprintf("[%s](%s/%s)", text, "/media/images/", id)
	default:
		return original
	}
}

func (ss *SiteStrapper) parseContent(content string) (htmlContent string, err error) {
	reLinks := regexp.MustCompile(`\[(.+)]\((.+)\)`)
	content = reLinks.ReplaceAllStringFunc(content, func(found string) string {
		g := reLinks.FindAllStringSubmatch(found, -1)
		if len(g) != 1 {
			return found
		}
		return ss.parseLink(found, g[0][1], g[0][2])
	})

	md := markdown.New(markdown.HTML(true))
	htmlContent = md.RenderToString([]byte(content))

	return htmlContent, nil
}

type page struct {
	Link  string
	Title string
}

func (ss *SiteStrapper) getPage(ref string) (p page, err error) {
	for _, tmp := range ss.sitemap.Pages {
		if tmp.ID == ref {
			return page{
				Link:  tmp.PublicPath(),
				Title: tmp.LinkTitle,
			}, nil
		}
	}
	return p, errors.Errorf("could not find page %s", ref)
}

func (ss *SiteStrapper) makeSingleTemplate(name string, content string) (tmpl *template.Template, err error) {
	tmpl, err = template.New(name).
		Funcs(template.FuncMap{
			"media": func(mediaType string) template.HTML {
				var c template.HTML
				for _, m := range ss.sitemap.Media {
					if m.Type == mediaType {
						switch m.Type {
						case "stylesheet":
							c += template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="/%s" />`, m.Path))
						case "script":
							c += template.HTML(fmt.Sprintf(`<script src="/%s"></script>`, m.Path))
						}
					}
				}
				return c
			},
			"tmpl": func(name string, params ...string) template.HTML {
				t, ok := ss.templates[name]
				if !ok {
					return template.HTML(fmt.Sprintf("<!-- could not find template %s -->", name))
				}

				out := bytes.NewBufferString("")
				data := make(map[string]string)
				for i, p := range params {
					data[fmt.Sprintf("Param%d", i+1)] = p
				}
				err := t.T.Execute(out, data)
				if err != nil {
					log.Printf("cannot execute template %s (%v)\n", name, err)
					return template.HTML("")
				}

				return template.HTML(out.String())
			},
			"page": func(id string) (page page) {
				page, _ = ss.getPage(id)
				return page
			},
			"pageLink": func(id string) string {
				p, err := ss.getPage(id)
				if err != nil {
					return id
				}
				return p.Link
			},
			"imageLink": func(imageName string) string {
				return fmt.Sprintf("/media/images/%s", imageName)
			},
			"pages": func(category string) (pages []page) {
				c, ok := ss.sitemap.Categories[category]
				if !ok {
					return pages
				}
				for _, ref := range c {
					p, err := ss.getPage(ref)
					if err != nil {
						continue
					}
					pages = append(pages, p)
				}
				return pages
			},
		}).
		Parse(content)
	if err != nil {
		return tmpl, errors.WithStack(err)
	}
	return tmpl, nil
}

func (ss *SiteStrapper) makeTemplate(smT sitemapTemplate) (siteT *siteTemplate, err error) {
	filename := path.Join(ss.inputDirectory, smT.Path)
	f, err := os.Open(filename)
	if err != nil {
		return siteT, errors.WithStack(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return siteT, errors.WithStack(err)
	}

	s := string(b)

	parts := strings.SplitN(s, "====", 2)
	if len(parts) != 2 {
		return siteT, errors.Errorf("header missing in template %s", smT.Path)
	}
	var header templateHeader
	if err := yaml.Unmarshal([]byte(parts[0]), &header); err != nil {
		return siteT, errors.WithStack(err)
	}

	content := strings.Trim(parts[1], " \n")

	t, err := ss.makeSingleTemplate(header.Name, content)
	if err != nil {
		return siteT, err
	}

	siteT = &siteTemplate{
		Header: header,
		T:      t,
	}
	return siteT, nil
}

type pageHeader struct {
	Template  string
	ID        string
	Title     string
	LinkTitle string `yaml:"linkTitle"`
}

func (ss *SiteStrapper) makeTemplates() (err error) {
	ss.templates = make(map[string]*siteTemplate)
	for _, tmpl := range ss.sitemap.Templates {
		t, err := ss.makeTemplate(tmpl)
		if err != nil {
			return err
		}
		ss.templates[t.Header.Name] = t
	}
	return nil
}

type pageContents struct {
	Content   template.HTML
	Title     string
	SiteTitle string
}

func (ss *SiteStrapper) gatherPageInfo(page *sitemapPage) (err error) {
	filename := path.Join(ss.inputDirectory, page.Path)
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.WithStack(err)
	}

	s := string(b)

	parts := strings.SplitN(s, "====", 2)
	if len(parts) != 2 {
		return errors.Errorf("header missing in page %s", page.Path)
	}
	var header pageHeader
	if err := yaml.Unmarshal([]byte(parts[0]), &header); err != nil {
		return errors.WithStack(err)
	}

	if len(header.Template) == 0 {
		header.Template = "Main"
	}

	if len(header.ID) > 0 {
		page.ID = header.ID
	} else {
		page.ID = strings.ToLower(strings.Replace(strings.Replace(page.Path, "site/", "", 1), filepath.Ext(page.Path), "", 1))
	}

	if len(header.Title) > 0 {
		page.Title = header.Title
	} else {
		tmp := strings.Split(page.ID, "/")
		last := tmp[len(tmp)-1]
		page.Title = fmt.Sprintf("%s%s", strings.ToUpper(last[0:1]), last[1:])
	}

	if len(header.LinkTitle) > 0 {
		page.LinkTitle = header.LinkTitle
	} else {
		page.LinkTitle = page.Title
	}

	page.Template = header.Template
	page.Content = strings.Trim(parts[1], " \n")

	return nil
}

func (ss *SiteStrapper) generatePage(page sitemapPage) (err error) {
	t, ok := ss.templates[page.Template]
	if !ok {
		return errors.Errorf("no such template %s in page %s", page.Template, page.Path)
	}

	content, err := ss.parseContent(page.Content)
	if err != nil {
		return err
	}

	ct, err := ss.makeSingleTemplate(page.ID, content)
	if err != nil {
		return err
	}

	outContent := bytes.NewBufferString("")
	err = ct.Execute(outContent, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	out := bytes.NewBufferString("")
	err = t.T.Execute(out, pageContents{
		Content:   template.HTML(outContent.String()),
		Title:     page.Title,
		SiteTitle: ss.sitemap.Title,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	outFilenamePath := path.Join(ss.outputDirectory, page.PublicPath()[1:])
	dir := filepath.Dir(outFilenamePath)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return errors.WithStack(err)
			}
		} else {
			return errors.WithStack(err)
		}
	}
	of, err := os.Create(outFilenamePath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer of.Close()

	_, err = of.Write(out.Bytes())
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func ensureDirectory(file string) (err error) {
	dir := filepath.Dir(file)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (ss *SiteStrapper) writeMedia(media sitemapMedia) (err error) {
	inPath := path.Join(ss.inputDirectory, media.Path)
	mediaDir := path.Join(ss.outputDirectory, "media")
	p := strings.Replace(media.Path, "media/", "", 1)
	outPath := path.Join(mediaDir, p)
	ff, err := os.Open(inPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer ff.Close()

	if err := ensureDirectory(mediaDir); err != nil {
		return err
	}
	if err := ensureDirectory(outPath); err != nil {
		return err
	}

	tf, err := os.Create(outPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		cerr := tf.Close()
		if cerr != nil {
			err = errors.WithStack(cerr)
		}
	}()

	fi, err := ff.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	b := make([]byte, fi.Size(), fi.Size())
	if _, err := ff.Read(b); err != nil {
		return errors.WithStack(err)
	}

	if _, err := tf.Write(b); err != nil {
		return errors.WithStack(err)
	}

	err = tf.Sync()
	return errors.WithStack(err)
}

func (ss *SiteStrapper) GenerateSite() (err error) {
	err = ss.makeSitemap()
	if err != nil {
		return err
	}

	err = ss.makeTemplates()
	if err != nil {
		return err
	}

	for i, page := range ss.sitemap.Pages {
		if err := ss.gatherPageInfo(&page); err != nil {
			return err
		}
		ss.sitemap.Pages[i] = page
	}

	for _, page := range ss.sitemap.Pages {
		if err := ss.generatePage(page); err != nil {
			return err
		}
	}

	for _, media := range ss.sitemap.Media {
		if err := ss.writeMedia(media); err != nil {
			return err
		}
	}

	return nil
}
