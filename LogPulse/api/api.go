package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/gophergala2016/Pulse/LogPulse/config"
	"github.com/gophergala2016/Pulse/LogPulse/email"
	"github.com/gophergala2016/Pulse/LogPulse/file"
	"github.com/gophergala2016/Pulse/pulse"
)

// Result : is used for ResponseWriter in handlers
type Result struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

var buffStrings []string
var port int

func init() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			os.Exit(0)
		}
	}()
	val, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("API: %s", err))
	}
	port = val.Port
}

// Start : will start the REST API
func Start() {
	http.HandleFunc("/log/message", StreamLog)
	http.HandleFunc("/log/file", SendFile)

	fmt.Printf("Listening on localhost:%d\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

}

// StreamLog : Post log statement to our API
func StreamLog(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		result, _ := json.Marshal(Result{400, "bad request"})
		io.WriteString(w, string(result))
		return
	}

	decoder := json.NewDecoder(r.Body)
	var body struct {
		Message string `json:"message"`
	}

	err := decoder.Decode(&body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		result, _ := json.Marshal(Result{400, "bad request"})
		io.WriteString(w, string(result))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(Result{200, "success"})
	io.WriteString(w, string(result))

}

// SendFile : Post log files to our API
func SendFile(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		result, _ := json.Marshal(Result{400, "bad request"})
		io.WriteString(w, string(result))
		return
	}
	compressed := false

	f, header, err := r.FormFile("file")
	fmt.Println("Form File")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		result, _ := json.Marshal(Result{400, "bad request"})
		io.WriteString(w, string(result))
		return
	}

	defer f.Close()

	var body struct {
		Email string `json:"email"`
	}
	r.ParseForm()
	body.Email = r.Form["email"][0]
	if !email.IsValid(body.Email) {
		w.Header().Set("Content-Type", "application/json")
		result, _ := json.Marshal(Result{400, "email not valid"})
		io.WriteString(w, string(result))
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		result, _ := json.Marshal(Result{400, "bad request"})
		io.WriteString(w, string(result))
		return
	}
	fmt.Println("Received bodys", body.Email)
	extension := filepath.Ext(header.Filename)
	filename := header.Filename[0 : len(header.Filename)-len(extension)]

	if extension == ".gz" {
		// Load compressed file on disk
		out, err := os.Create(fmt.Sprintf("%s.gz", filename))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			result, _ := json.Marshal(Result{400, "bad request"})
			io.WriteString(w, string(result))
			return
		}

		defer out.Close()

		// write the content from POST to the file
		_, err = io.Copy(out, f)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			result, _ := json.Marshal(Result{400, "gzip copy failed"})
			io.WriteString(w, string(result))
			return
		}

		// Uncompress file
		err = file.UnGZip(fmt.Sprintf("%s.gz", filename))
		if err != nil {
			spew.Dump(err)
			w.Header().Set("Content-Type", "application/json")
			result, _ := json.Marshal(Result{400, "gzip uncompressed failed"})
			io.WriteString(w, string(result))
			return
		}

		compressed = true
	}

	stdIn := make(chan string)
	email.ByPassMail = true // Needs to bypass emails and store in JSON
	email.OutputFile = filename + ".json"
	email.EmailList = []string{body.Email}

	go func() {
		start := time.Now()
		pulse.Run(stdIn, email.SaveToCache)
		line := make(chan string)

		if compressed {
			file.Read(filename, line)
		} else {
			file.StreamRead(f, line)
		}

		for l := range line {
			if l == "EOF" {
				email.ByPassMail = false
				// Once EOF, time to send email from cache JSON storage
				email.SendFromCache(email.OutputFile)
				break
			}
			stdIn <- l
		}
		close(stdIn)

		elapsed := time.Since(start)
		log.Printf("Pulse Algorithm took %s", elapsed)

		// Clean up
		if compressed {
			err := os.Remove(filename)
			if err != nil {
				fmt.Println("Failed to delete uncompressed file, please delete")
			}

			err = os.Remove(fmt.Sprintf("%s.gz", filename))
			if err != nil {
				fmt.Println("Failed to delete uncompressed file, please delete")
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(Result{200, "success"})
	io.WriteString(w, string(result))
}