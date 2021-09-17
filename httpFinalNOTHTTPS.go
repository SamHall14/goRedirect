// Author: Samuel Hall
// Date: Sep 14 2021


// TODO:
// Check that targets actually exist and return
// Ensure that shorthands don't redirect to other shorthands


// Done:
// Handles /index -> index.html
// Handles alphanumeric input only
// Handles shutdowns upon SIGTERM

// SOURCES: BSD CODE
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// From the HTTP demo: https://golang.org/doc/articles/wiki/

// Stack Overflow solutions used
// Uses https://stackoverflow.com/a/66607600 for
// shutdown implementation using SIGTERM

// And of course any documentation within godoc -http :8888

package main

import (
	"encoding/gob"
	"os"
	"fmt"
	_ "html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"os/signal"
	"context"
	"sync"
	_ "errors"
)

type Page struct {
	Title string
	Body []byte
}

type Links struct {
	Filename string
	Redirects map[string]string
}


func NewLinks() *Links {
	links := Links{Filename: "links1", Redirects: make(map[string]string) }
	links.Redirects["exampleShorthand"] = "https://www.youtube.com/watch?v=1cfKc8U4iwg"
	links.Redirects["index"] = "/index.html"
	return &links
}

var (
	links *Links = NewLinks()
	theIndex []byte
	indexErr error
)

func (p *Page) save() error {
	filename := p.Title+".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title+".txt"
	body, err :=  ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

//func viewHandler(w http.ResponseWriter, r *http.Request, title string){
//	p, err := loadPage(title)
//	if err != nil {
//		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
//		return
//	}
//	renderTemplate(w, "view", p)
//}
//
//func editHandler(w http.ResponseWriter, r *http.Request, title string) {
//	p, err := loadPage(title)
//	if err != nil {
//		p = &Page{Title: title}
//	}
//	renderTemplate(w, "edit", p)
//}


func saveHandler(w http.ResponseWriter, r *http.Request, title string){
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	//indexPage, err := ioutil.ReadFile("index.html")
	//if err != nil {
	//	return
	//}
	fmt.Fprintf(w, string(theIndex))
}

// loads a iink given
func shorthandHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("shorthandHandler is dealing with", r.URL.Path)
	//if(r.URL.Path == "/shorthand/load/") {
	if r.FormValue("shorthandExisting") != "" && r.FormValue("shorthandExisting") != "index" {
		http.Redirect(w, r, "/"+r.FormValue("shorthandExisting"), http.StatusSeeOther)
		return
	}
	//}
	if r.URL.Path == "/" {
		//thePage, err := ioutil.ReadFile("index.html")
		http.Redirect(w, r, "/index.html", http.StatusFound)
		return
	}
	shorthand := reroutePath.FindStringSubmatch(r.URL.Path)
	if shorthand == nil {
		fmt.Println("404 found for", r.URL.Path)
		http.Error(w, "404: File not found!", http.StatusNotFound)
		return
	}
	if shorthand[1] == "" {
		http.NotFound(w, r)
		return
	}
	if links.Redirects[shorthand[1]] == "" {
		http.Error(w, "File not found", http.StatusNotFound)
	}
	http.Redirect(w, r, links.Redirects[shorthand[1]], http.StatusSeeOther)


}

func createHandler(w http.ResponseWriter, r *http.Request) {
	tmpShorthand := r.FormValue("shorthand")
	target := r.FormValue("target")
	// handler
	if tmpShorthand == "" || target == "" {
		http.Error(w, "A TARGET AND HANDLER MUST BE DEFINED", http.StatusBadRequest)
		return
	}
	// check if links already is taken
	if links.Redirects[tmpShorthand] != ""  {
		http.Error(w, "Shorthand already taken, please try another one.",
			http.StatusConflict)
		return
	}
	// assign shorthand to target
	// use template to send back a confirm message
	// MAKE SURE THAT TARGET HAS https:// or http:// at the front or it will be treated internally
	// if http continue, else request that target starts with "http or https"
	//fmt.Println(httpCheck.MatchString(target))
	if httpCheck.MatchString(target) {
		links.Redirects[tmpShorthand] = target
		fmt.Fprintf(w, "<h1>Success!</h1><br><p>You have created a link @ localhost:8080/%s pointing to %s</p>",
			tmpShorthand, target)
	} else {
		fmt.Fprintf(w, "<h1>Failure!</h1><p>Please check that your URL contains http or https!")
	}
	return

}



// var templates = template.Must(template.ParseFiles("edit.html", "view.html"))
//var templates2 = template.Must(template.ParseFiles("index.html", "create.html"))

//func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
//	err := templates.ExecuteTemplate(w, tmpl+".html", p)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//	}
//}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var indexCatcher = regexp.MustCompile("^/index[/]+$")
var reroutePath = regexp.MustCompile("^/([a-zA-Z0-9]+)$")
var httpCheck = regexp.MustCompile("^https?://.*")

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

// saves links to disk
func (theLinks *Links) saveLinks(filename string) {
	// WARNING: os.Create truncates file (empties all contents) when writing!
	file, _ := os.Create(filename+".gob")
	// Write to disk
	defer file.Close()

	encoder :=  gob.NewEncoder(file)
	encoder.Encode(theLinks)

}


func (theLinks *Links) loadLinks(filename string) {
	file, err := os.Open(filename+".gob")
	if err == nil {
		// decode file interface, create the decoder
		gobDecoder := gob.NewDecoder(file)
		// use decoder to decode "file" then assign result to pointer "theLinks"
		gobDecoder.Decode(theLinks)
	}
	file.Close()
	return
}

func startHttpServer(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/index.html", indexHandler)
	http.HandleFunc("/shorthand/", createHandler)
	http.HandleFunc("/", shorthandHandler)

	go func() {
		defer wg.Done()
		// always return an error!, ErrServerClosed on a graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed{
			// Handle unexpected errors
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return srv
}

func main() {
	// first load index file
	theIndex, indexErr = ioutil.ReadFile("index.html")
	// then load links
	links.loadLinks("links1")
	if indexErr != nil {
		panic("index.html could not be found! Please check your root directory!")
	}

	// define server with handlers
	http.HandleFunc("/index.html", indexHandler)
	http.HandleFunc("/shorthand/", createHandler)
	http.HandleFunc("/", shorthandHandler)
	srv := &http.Server{Addr: ":8080"}
	// create wait group
	httpServerExitDone := &sync.WaitGroup{}
	// adds 1 to the delta, when delta goes to zero it will release
	// all suspended goroutines in the waitgroup, panics if delta drops
	// below zero.
	httpServerExitDone.Add(1)


	// Create an anonymous function that after receiving SIGTERM will tell the
	// server to Shutdown().
	go func() {
		killSignal := make(chan os.Signal, 1)
		signal.Notify(killSignal, os.Interrupt)
		<-killSignal
		// initiates shutdown
		shutdownErr := srv.Shutdown(context.Background())
		if shutdownErr != nil {
			log.Printf("Error during shutdown: %v\n", shutdownErr)
		}
		// flush waitgroup
		httpServerExitDone.Done()
	}()

	log.Println("Listening on port 8080...")
	//Execution flow freezes until SIGTERM is given which signals srv to Shutdown
	httpErr := srv.ListenAndServe()
	// handling a proper shutdown
	if httpErr == http.ErrServerClosed {
		log.Println("Starting shutdown sequence for server on port :8080")
		httpServerExitDone.Wait()
		log.Println("Server successfully shut down!")
		links.saveLinks("links1")
	} else if httpErr != nil {
		log.Printf("Unexpected error received upon server shutdown: %v\n", httpErr)
	}

}
