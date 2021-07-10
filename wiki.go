package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Page struct {
	Title string
	Body  []byte
	Route string
}

type Archive struct {
	Wikis []string
}

var templates = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/edit.html", "tmpl/home.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9-]+)$|^/(create)$|^/$")

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile("data/"+filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile("data/" + filename)
	if err != nil {
		return nil, err
	}

	return &Page{Title: strings.Title(title), Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func joinHyphenatedTitles(title string) string {
	containsHyphen := strings.Contains(title, "-")
	if containsHyphen {
		splittedTitle := strings.Split(title, "-")
		return strings.Join(splittedTitle, " ")
	}
	return title
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	title = joinHyphenatedTitles(title)
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+strings.ToLower(title), http.StatusFound)
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	urlTitle := title
	title = joinHyphenatedTitles(title)
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}

	p.Route = urlTitle

	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	urlTitle := title
	title = joinHyphenatedTitles(title)
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(w, r, "/view/"+strings.ToLower(urlTitle), http.StatusFound)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	m := validPath.FindStringSubmatch((r.URL.Path))

	if m == nil {
		http.NotFound(w, r)
		return
	}

	files, _ := ioutil.ReadDir("data")

	var fileNames []string

	for _, file := range files {
		extIndex := strings.Index(file.Name(), ".txt")
		fileName := file.Name()[:extIndex]
		fileNames = append(fileNames, strings.Title(fileName))
	}

	renderTemplate(w, "home", &Archive{Wikis: fileNames[:6]})
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	p := &Page{Title: strings.ToLower(title), Body: []byte("")}
	err := p.save()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	splittedTitle := strings.Split(title, " ")

	if len(splittedTitle) > 1 {
		// fmt.Println(strings.Join(splittedTitle, "-"))
		http.Redirect(w, r, "/edit/"+strings.ToLower(strings.Join(splittedTitle, "-")), http.StatusFound)
	}

	http.Redirect(w, r, "/edit/"+strings.ToLower(title), http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)

		if m == nil {
			http.NotFound(w, r)
			return
		}

		fn(w, r, m[2])
	}
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/create/", createHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
