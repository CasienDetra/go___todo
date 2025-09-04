package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Page struct {
	Title string
	Body  []byte
}
type ListPageInfo struct {
	PageTitle string
	Pages     []Page
}

const (
	port     = ":8080"
	htmlpath = "./html/"
	NotePath = "./My-Notes/"
)

var (
	templates *template.Template
	validPath = regexp.MustCompile(`^/(edit|view)/([a-zA-Z0-9]+)$`)
)

func (p *Page) save() error {
	if err := os.MkdirAll(NotePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", NotePath, err)
	}
	filepath := filepath.Join(NotePath, p.Title+".txt")
	return os.WriteFile(filepath, p.Body, 0644)
}

func initTemplates() error {
	var err error
	templates, err = template.ParseGlob(filepath.Join(htmlpath, "*.html"))
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}
	return nil
}
func addHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		renderTemplate(w, "add", nil)
	case http.MethodPost:
		title := strings.TrimSpace(r.FormValue("title"))
		body := r.FormValue("body")
		if title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}
		page := &Page{Title: title, Body: []byte(body)}
		if err := page.save(); err != nil {
			log.Printf("Failed to save page %s: %v", title, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/view/"+title, http.StatusFound)
	}
}
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		matches := validPath.FindStringSubmatch(r.URL.Path)
		if matches == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, matches[2])
	}
}
func loadPage(title string) (*Page, error) {
	filename := filepath.Join(NotePath, title+".txt")
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get page %s: %w", title, err)
	}
	return &Page{Title: title, Body: body}, nil

}
func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, name+".html", data)
	if err != nil {
		log.Printf("failed to load html file %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	page, err := loadPage(title)
	if err != nil {
		log.Printf("Failed to load page %s: %v", title, err)
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", page)
}
func listHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(NotePath)
	if err != nil {
		log.Printf("Failed to fetch the data: %v ", err)
		files = []os.DirEntry{}
	}
	var pages []Page
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		if !strings.HasSuffix(filename, ".txt") {
			continue
		}
		title := strings.TrimSuffix(filename, filepath.Ext(filename))
		pages = append(pages, Page{Title: title})
	}
	info := ListPageInfo{PageTitle: "List  Pages", Pages: pages}
	renderTemplate(w, "list", info)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	page := &Page{Title: title, Body: []byte(body)}

	if err := page.save(); err != nil {
		log.Printf("Failed to save page %s: %v", title, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	page, err := loadPage(title)
	if err != nil {
		page = &Page{Title: title}
	}
	renderTemplate(w, "edit", page)
}

func main() {
	if err := initTemplates(); err != nil {
		log.Fatalf("Failed to init: %v", err)
	}
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/", listHandler)

	log.Printf("Server is running at port 8080")
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
