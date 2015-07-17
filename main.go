package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"syscall"

	"github.com/robertkrimen/otto"
)

// requestCookie: Get a session cookie from the BMC
func requestCookie(client *http.Client, base url.URL) string {
	form := url.Values{}
	form.Set("WEBVAR_USERNAME", username)
	form.Add("WEBVAR_PASSWORD", password)
	uri := base
	uri.Path = "/rpc/WEBSES/create.asp"
	response, err := client.PostForm(uri.String(), form)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	} else {
		defer response.Body.Close()
		content, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}
		return string(content)
	}
	return ""
}

// requestJNLP: fetch the jnlp for launching jViewer
func requestJNLP(client *http.Client, base url.URL, address string) string {
	uri := base
	uri.Path = "/Java/jviewer.jnlp"
	uri.RawQuery = fmt.Sprintf("EXTRNIP=%s&JNLPSTR=JViewer", address)
	response, err := client.Get(uri.String())
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	} else {
		defer response.Body.Close()
		content, err := ioutil.ReadAll(response.Body)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				log.Fatalf("Error: %s\n", err)
			}
		}
		return string(content)
	}
	return ""
}

// writeJNLPFile: Can't pipe to javaws, so we drop a temporary file
func writeJNLPFile(jnlp string) *os.File {
	tempdir := os.Getenv("TMPDIR")
	if tempdir == "" {
		tempdir = "/tmp"
	}
	file, err := ioutil.TempFile(tempdir, "console")
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	w := bufio.NewWriter(file)
	defer w.Flush()
	_, err = w.WriteString(jnlp)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	return file
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s host\nConnect to an AMIBios console.\n", filepath.Base(os.Args[0]))
		os.Exit(0)
	}
	hostAddresses, err := net.LookupHost(os.Args[1])
	address := hostAddresses[0]
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	cookies, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookies,
	}
	fmt.Fprintf(os.Stderr, "Trying host %s\n", address)
	base, _ := url.Parse(fmt.Sprintf("http://%s/", address))
	vm := otto.New()
	js := requestCookie(client, *base)
	result, err := vm.Run(js + "\nWEBVAR_JSONVAR_WEB_SESSION.WEBVAR_STRUCTNAME_WEB_SESSION[0].SESSION_COOKIE")
	cookie, _ := result.ToString()
	cookies.SetCookies(base, []*http.Cookie{{Name: "SessionCookie", Value: cookie}})
	jnlp := requestJNLP(client, *base, address)
	file := writeJNLPFile(jnlp)
	//	syscall.Exec("/bin/bash", []string{"bash", "-c", fmt.Sprintf("javaws %s", file.Name())}, os.Environ())
	//	syscall.Exec("/System/Library/Frameworks/JavaVM.framework/Versions/1.6/Commands/javaws", []string{"javaws", file.Name()}, []string{"PATH=/System/Library/Frameworks/JavaVM.framework/Versions/1.6/Commands","JAVA_HOME=/System/Library/Frameworks/JavaVM.framework/Versions/1.6/Home"})
	syscall.Exec("/System/Library/Frameworks/JavaVM.framework/Commands/javaws", []string{"/System/Library/Frameworks/JavaVM.framework/Commands/javaws", file.Name()}, os.Environ())
}
