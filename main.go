package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

func form(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
	<html>
	<body background="https://vignette.wikia.nocookie.net/angelbeats/images/4/46/Ab_character_takeyama_image.png/revision/latest?cb=20150222223450">	
	<form action="/convert" method="post" enctype="multipart/form-data">
	  <input type="file" name="file" accept="image/*">
	  <input type="submit">
	</form>
	</body>
	</html>
	`)
}

func convert(w http.ResponseWriter, r *http.Request) {
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
	imgSrc, err := png.Decode(bytes.NewBuffer(data))
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "png decode: %v", err)
		return
	}
	newImg := image.NewRGBA(imgSrc.Bounds())
	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(newImg, newImg.Bounds(), imgSrc, imgSrc.Bounds().Min, draw.Over)
	var opt jpeg.Options
	opt.Quality = 95

	var extension = filepath.Ext(header.Filename)
	var name = header.Filename[0 : len(header.Filename)-len(extension)]
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%v_2x.jpg"`, name))
	err = jpeg.Encode(w, newImg, &opt)
	if err != nil {
		log.Printf("jpeg encode err: %v", err)
	}

}

func main() {
	rand.Seed(time.Now().Unix())
	r := mux.NewRouter()
	r.HandleFunc("/", form)
	r.HandleFunc("/convert", convert)

	err := http.ListenAndServe(":"+os.Getenv("PORT"), r)
	if err != nil {
		log.Fatal("listen: ", err)
	}
}
