package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	rand.Seed(time.Now().Unix())
	r := mux.NewRouter()
	r.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength <= 0 {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Only fixed-size files are allowed")
			return
		}
		if r.ContentLength > 16*1024*1024 {
			w.WriteHeader(400)
			fmt.Fprintf(w, "filesize is > 16 MB")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024*1024)
		err := r.ParseMultipartForm(16 * 1024 * 1024)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "multipart read: %v", err)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "can't find multipart file with name 'file': %v", err)
			return
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "read: %v", err)
			return
		}
		inFile := fmt.Sprintf("/data/%v_%v", rand.Int63(), header.Filename) // TODO: fix security
		outFile := fmt.Sprintf("/data/%v_out.png", rand.Int63())
		err = ioutil.WriteFile(inFile, data, 7777)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "write file: %v", err)
			return
		}
		cmd := exec.Command("/opt/waifu2x-cpp/waifu2x-converter-cpp", fmt.Sprintf("-i %v", inFile), fmt.Sprintf("-o %v", outFile))
		out, err := cmd.CombinedOutput()
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "waifu2x: %v %v", err, string(out))
			return
		}
		log.Print("waifu: ", string(out))
		data, err = ioutil.ReadFile(outFile)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "read file: %v", err)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="2x_%v"`, header.Filename))
		w.Write(data)
	})

	err := http.ListenAndServe(":"+os.Getenv("PORT"), r)
	if err != nil {
		log.Fatal("listen: ", err)
	}
}
