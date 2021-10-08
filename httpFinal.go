// Author: Samuel Hall
// Date: Oct 1 2021

// HERE BE DRAGONS THAT EAT YOUR DATA
// Editable defaults in the constants field (Line 88)
// I hope you backed up your links.

// TODO:

// Ensure that shorthands don't redirect to other shorthands
// This will have to be deployed on my server
// Enable TLS with autocert
// https://www.geeksforgeeks.org/using-certbot-manually-for-ssl-certificates/
//

// Custom flags that take in default names for autosave and normal save files
// Flags for port, creation limiters, and save refresh rate

// Log system to route to either systemd or syslog-ng

// Done:
// Handles /index -> index.html
// Handles alphanumeric input only
// Handles shutdowns upon SIGTERM
// implement a favicon.ico for web browsers (icon you see in the tab)

// DONE: limit shorthand length:
// source: https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/maxlength

// DONE: Checking for last IP users to prevent spamming
// possible source: https://golangcode.com/get-the-request-ip-addr/

//DONE: Autosave feature
// have a recovery features that checks the lastest saves for autosave
// and shutdown  and picks the latest one
// check io/fs package for ModTime() function for FileInfo

// DONE: IP catching for abuse of shorthand creations
// memory map for recent ip users containing dates that clears after an amount of time
// Restricting creation on an IP basis on a timer (1-5 mintes)

// DONE: Check that targets actually exist and return (part of URL validation
// in createHandler

// SOURCES: BSD CODE
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// From the HTTP demo: https://golang.org/doc/articles/wiki/

// Note that creative commons licensed works (BY-SA) are compatible with GPLv3 (not vice versa)
// Source: https://creativecommons.org/share-your-work/licensing-considerations/compatible-licenses

// Stack Overflow solutions used (CCv4 which is compatible with GPLv3 from what I understand)
// Uses https://stackoverflow.com/a/66607600 for
// shutdown implementation using SIGTERM

// Checking for a valid URL:
// Sources:
// https://golang.cafe/blog/how-to-validate-url-in-go.html pointed me to the url package

// TLS/HTTPS info: https://juliensalinas.com/en/security-golang-website/
// Without Certbot: https://github.com/denji/golang-tls
// Self-signed with go utils: https://youtu.be/ZKlwKg-f__0?t=902
// Package for certbot: https://pkg.go.dev/golang.org/x/crypto/acme/autocert?utm_source=godoc
// go get golang.org/x/crypto/acme/autocert
// autocert demo: https://blog.kowalczyk.info/article/Jl3G/https-for-free-in-go-with-little-help-of-lets-encrypt.html

// And of course any documentation within godoc -http :8888

// check

package main

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"time"
)

const (
	// file to save to upon sigterm (ends in .gob)
	defaultStandardSave = "links1"
	// file to autosave to (ends in .gob)
	defaultAutoSave = "links.auto"
	// File extension name for links files
	defaultExtension = ".gob"
	// autosave rate for every n minutes
	autoSaveRate = 10
	// name of host name according to DNS/IP address
	siteName = "redirect.samhall.xyz:8443"
	// maximum shorthand length
	MAX_SHORTHAND = 32
	// latency between creates for a given IP
	MINUTES_BETWEEN_CREATIONS = 3
)

type Page struct {
	Title string
	Body  []byte
}

// allows for memory to work properly between the final save and autosave
type Links struct {
	Filename        string
	Redirects       map[string]string
	LastSave        time.Time
	LatestInclusion time.Time
}

// NOTE: THIS IMPLEMENTATION IS VOLATILE AND WILL NOT SAVE RESTRICTED IPs BETWEEN SESSIONS!
// user ips is a set that will create a goroutine that waits and then deletes it from map
var UserIPs map[string]bool = make(map[string]bool)

var (
	links    *Links = NewLinks()
	theIndex []byte
	indexErr error
	// zero value for time.Time{}
	zeroTime = time.Time{}
	waitTime = MINUTES_BETWEEN_CREATIONS
)

type IndexConstants struct {
	MaxShorthand int
	SiteName     string
}

var indexConstants IndexConstants = IndexConstants{MAX_SHORTHAND, siteName}

type CreateConstants struct {
	SiteName         string
	CreatedShorthand string
	DesignatedTarget string
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

// holding onto the examples from http demo as reference.
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

// generates links object that is
func NewLinks() *Links {
	links := Links{Filename: defaultStandardSave, Redirects: make(map[string]string)}
	links.Redirects["exampleShorthand"] = "https://www.youtube.com/watch?v=1cfKc8U4iwg"
	links.Redirects["index"] = "/index.html"
	return &links
}

// handler for saving from the original golang http demo
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
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

	renderTemplate(w, "indexTemplate", indexConstants)
	// Fprintf the index
	//fmt.Fprintf(w, string(theIndex))
}

// loads a link given
func shorthandHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("shorthandHandler is dealing with", r.URL.Path)
	if r.FormValue("shorthandExisting") != "" && r.FormValue("shorthandExisting") != "index" {
		http.Redirect(w, r, "/"+r.FormValue("shorthandExisting"), http.StatusSeeOther)
		return
	}
	if r.URL.Path == "/" {
		//thePage, err := ioutil.ReadFile("index.html")
		http.Redirect(w, r, "/index.html", http.StatusFound)
		return
	}
	shorthand := reroutePath.FindStringSubmatch(r.URL.Path)
	if shorthand == nil {
		//fmt.Println("404 found for", r.URL.Path)
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

// Grabs IPs even in an event of a redirect
// source: https://golangcode.com/get-the-request-ip-addr/
func ParseIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-FORWARDED-FOR")
	// if forwarded, return source ip
	if forwardedFor != "" {
		return forwardedFor
	}
	return r.RemoteAddr
}

func UserTimeOut(address string) {
	// if accidentally called on an ip that is in the map, log the error
	if UserIPs[address] {
		log.Println("Unexpected IP recorded that already was on timeout list:", address)
		return
	}
	UserIPs[address] = true
	time.Sleep(MINUTES_BETWEEN_CREATIONS * time.Minute)

	delete(UserIPs, address)
	return
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	// check if this is from the same ip to prevent mass attacks
	requesterIP := ParseIP(r)
	// if true then
	if UserIPs[requesterIP] {
		fmt.Fprintf(w, "<h1>Error 429: Too Many Requests!</h1>Please wait %v minutes between shorthand assignments!\n",
			MINUTES_BETWEEN_CREATIONS)
		http.Error(w, "Bad Request", http.StatusTooManyRequests)
		return
	}
	// grab the shorthand!
	tmpShorthand := r.FormValue("shorthand")

	target := r.FormValue("target")
	// handler or target is null
	if tmpShorthand == "" || target == "" {
		fmt.Fprintln(w, "<h1>400: Bad Request</h1>Please define a target and handler.")
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	// check if links is alphanumeric
	if !(shorthandCheck.MatchString(tmpShorthand)) {
		fmt.Fprintln(w, "<h1>400: Bad Request</h1>Please define a target with alphanumeric characters!")
		http.Error(w, "", http.StatusBadRequest)
	}
	// check if links already is taken
	if links.Redirects[tmpShorthand] != "" {
		http.Error(w, "Shorthand already taken, please try another one.",
			http.StatusConflict)
		return
	}

	// assign shorthand to target
	// use template to send back a confirm message
	// MAKE SURE THAT TARGET HAS https:// or http:// at the front or it will be treated internally
	// if http continue, else request that target starts with "http or https"
	//fmt.Println(httpCheck.MatchString(target))
	// checks if it is a valid url and then modifies the links object's latest update
	if httpCheck.MatchString(target) {
		_, getErr := http.Get(target)
		if getErr != nil {
			fmt.Fprintln(w,
				"<h1>400: Bad Request</h1>Target did not give back an ok response! Ensure your link works and has a valid cert if under https!")
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		// Successfully made a proper link! Do two things: check if URL gives a response

		links.Redirects[tmpShorthand] = target
		links.LatestInclusion = time.Now()
		if len(tmpShorthand) > MAX_SHORTHAND {
			fmt.Fprintf(w, "<h1>400 Bad Request</h1>Please ensure that your shorthand is at most %v characters long!",
				MAX_SHORTHAND)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		// create a thread that blacklists user for a time!
		go UserTimeOut(requesterIP)

		createInfo := CreateConstants{SiteName: siteName, CreatedShorthand: tmpShorthand,
			DesignatedTarget: target}
		// render template with success message
		renderTemplate(w, "create", createInfo)
		// old without template
		//fmt.Fprintf(w, "<h1>Success!</h1><br><p>You have created a link @ %s:8443/%s pointing to %s</p>",
		//	siteName, tmpShorthand, target)
	} else {
		fmt.Fprintf(w, "<h1>Failure!</h1><p>Please check that your URL contains http or https!")
	}
	return

}
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicons/favicon.ico")
}

//var templates = template.Must(template.ParseFiles("edit.html", "view.html"))
var templates = template.Must(template.ParseFiles("indexTemplate.html", "create.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, info interface{}) {
	err := templates.ExecuteTemplate(w, tmpl+".html", info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var indexCatcher = regexp.MustCompile("^/index[/]+$")
var reroutePath = regexp.MustCompile("^/([a-zA-Z0-9]+)$")
var shorthandCheck = regexp.MustCompile("^[a-zA-Z0-9+]$")
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
	theLinks.Filename = filename
	file, _ := os.Create(filename + defaultExtension)
	theLinks.LastSave = time.Now()
	// Write to disk
	defer file.Close()

	encoder := gob.NewEncoder(file)
	encoder.Encode(theLinks)

}

func (theLinks *Links) autosaveLinks(filename string) {
	// Note that time.Time{} is the zero value for the time.Time type
	// compare last new link's timestamp with
	// the last write timestamp, if the last write timestamp is
	// before the last new link timestamp, autosave to disk
	for {
		if theLinks.LatestInclusion.IsZero() {
			theLinks.LatestInclusion = time.Now()
		}
		// first wait for server to spin up
		time.Sleep(autoSaveRate * time.Minute)
		// check if last save has been recorded
		// if not, just save just to be sure.
		// if last save was before the latest shorthand, save
		if theLinks.LastSave.Before(theLinks.LatestInclusion) {
			theLinks.saveLinks(filename)
		}
		// was checking if lazy saving for autosave was working
		/*else {
			fmt.Println("skipping autosave since no new links created")
		} */
	}
}

func (theLinks *Links) loadLinks(normalFilename string, autoFilename string) {
	file, errDefault := os.Open(normalFilename + defaultExtension)
	fileAuto, errAuto := os.Open(autoFilename + defaultExtension)
	// if both autosave and normal save exist
	defer file.Close()
	defer fileAuto.Close()
	if errDefault == nil && errAuto == nil {
		// check which file is most recent
		// using the os packages' Stat method and
		infoNormal, _ := file.Stat()
		infoAuto, _ := fileAuto.Stat()
		// io/fs FileInfo methods
		modNormal := infoNormal.ModTime()
		modAuto := infoAuto.ModTime()
		// if autosave is after normal save, load auto, and vice versa
		if modAuto.After(modNormal) {
			log.Println("Autosave was created after normal save\nLoading from",
				(autoFilename + defaultExtension))
			// decode file interface, create the decoder
			gobDecoder := gob.NewDecoder(fileAuto)
			// use decoder to decode "file" then assign result to pointer "theLinks"
			gobDecoder.Decode(theLinks)
		} else {
			log.Println("Loading normal save from", (normalFilename + defaultExtension))
			// decode file interface, create the decoder
			gobDecoder := gob.NewDecoder(file)
			// use decoder to decode "file" then assign result to pointer "theLinks"
			gobDecoder.Decode(theLinks)
		}
	} else if errDefault == nil {
		log.Println("Loading normal save from", (normalFilename + defaultExtension))
		// decode file interface, create the decoder
		gobDecoder := gob.NewDecoder(file)
		// use decoder to decode "file" then assign result to pointer "theLinks"
		gobDecoder.Decode(theLinks)

	} else if errAuto == nil {
		log.Println("Loading autosave from", (autoFilename + defaultExtension))
		// decode file interface, create the decoder
		gobDecoder := gob.NewDecoder(file)
		// use decoder to decode "file" then assign result to pointer "theLinks"
		gobDecoder.Decode(theLinks)
	} else {
		log.Printf("Could not find either default file %s%s nor autosave %s%s\n",
			normalFilename, defaultExtension, autoFilename, defaultExtension)
	}
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
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// Handle unexpected errors
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return srv
}

func main() {
	// first load index file
	theIndex, indexErr = ioutil.ReadFile("index.html")

	// load links from the most recent file (final save/autosave)

	links.loadLinks(defaultStandardSave, defaultAutoSave)
	if indexErr != nil {
		panic("index.html could not be found! Please check your root directory!")
	}

	// print metadata on links object
	fmt.Println("Last save for links:", links.LastSave)
	fmt.Println("Last added link:", links.LatestInclusion)

	// TLS configs
	config := &tls.Config{MinVersion: tls.VersionTLS10}

	// define server with handlers
	http.HandleFunc("/index.html", indexHandler)
	http.HandleFunc("/shorthand/", createHandler)
	http.HandleFunc("/", shorthandHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)

	// CERT/KEY locations
	// generated in shell:
	// go run $GOROOT/src/crypto/tls/generate_cert.go --host=localhost
	// where GOROOT=$(go env GOROOT)
	tlsCertPath := "keys/cert.pem"
	tlsKeyPath := "keys/key.pem"
	//http version:
	//srv := &http.Server{Addr: ":8080"}
	//https version
	srv := &http.Server{
		Addr:      ":8443",
		TLSConfig: config,
	}
	srv2 := &http.Server{Addr: ":8080"}

	// create wait group
	httpServerExitDone := &sync.WaitGroup{}
	// adds 1 to the delta, when delta goes to zero it will release
	// all suspended goroutines in the waitgroup, panics if delta drops
	// below zero.
	httpServerExitDone.Add(1)

	// Create an anonymous function that after receiving SIGTERM will tell the
	// server to Shutdown().
	go links.autosaveLinks(defaultAutoSave)
	go func() {
		killSignal := make(chan os.Signal, 1)
		signal.Notify(killSignal, os.Interrupt)
		<-killSignal
		// initiates shutdown
		// comment out main server (srv) to develop main features
		shutdownErr := srv.Shutdown(context.Background())
		shutdownErr2 := srv2.Shutdown(context.Background())
		if shutdownErr != nil {
			log.Printf("Error during shutdown\n Port %v: %v\nPort %v:",
				srv.Addr, shutdownErr, srv2.Addr, shutdownErr2)
		}
		// flush waitgroup
		httpServerExitDone.Done()
	}()

	log.Println("Listening on port 8433, redirecting port 8080 to 8433...")
	//Execution flow freezes until SIGTERM is given which signals srv to Shutdown
	go srv2.ListenAndServe()
	httpErr := srv.ListenAndServeTLS(tlsCertPath, tlsKeyPath)
	//httpErr := srv.ListenAndServe()
	// handling a proper shutdown
	if httpErr == http.ErrServerClosed {
		log.Println("Starting shutdown sequence for server on port :8080")
		httpServerExitDone.Wait()
		log.Println("Server successfully shut down!")
		links.saveLinks(defaultStandardSave)
	} else if httpErr != nil {
		log.Printf("Unexpected error received upon server shutdown: %v\n", httpErr)
	}

}
