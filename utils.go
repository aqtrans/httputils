package utils

import (
    "log"
    "os"
    "io"
    "time"
    "strings"
    "html/template"
    "net/http"
    "fmt"
    "encoding/json"
    "crypto/rand"
    "bytes"
)

const timestamp = "2006-01-02 at 03:04:05PM"

var Debug bool

//JSON Response
type jsonresponse struct {
	Name    string `json:"name,omitempty"`
	Success bool   `json:"success"`
}

func Debugln(v ...interface{}) {
	if Debug {
		d := log.New(os.Stdout, "DEBUG: ", log.Ldate)
		d.Println(v)
	}
}

func PrettyDate(date int64) string {
	if date == 0 {
		return "N/A"
	}
	t := time.Unix(date, 0)
	return t.Format(timestamp)
}

func ImgClass(s string) string {
	if strings.HasSuffix(s, ".gif") {
		return "gifs"
	}
	return "imgs"
}

func SafeHTML(s string) template.HTML {
	return template.HTML(s)
}

//Hack to allow me to make full URLs due to absence of http:// from URL.Scheme in dev situations
//When behind Nginx, use X-Forwarded-Proto header to retrieve this, then just tack on "://"
//getScheme(r) should return http:// or https://
func GetScheme(r *http.Request) (scheme string) {
	scheme = r.Header.Get("X-Forwarded-Proto") + "://"
	/*
		scheme = "http://"
		if r.TLS != nil {
			scheme = "https://"
		}
	*/
	if scheme == "://" {
		scheme = "http://"
	}
	return scheme
}

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("[timer] %s took %s", name, elapsed)
}

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Status() int {
	return w.status
}

func (w *statusWriter) Size() int {
	return w.size
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	written, err := w.ResponseWriter.Write(b)
	w.size += written
	return written, err
}

//Custom Logging Middleware
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer

		start := time.Now()
		writer := statusWriter{w, 0, 0}

		buf.WriteString("Started ")
		fmt.Fprintf(&buf, "%s ", r.Method)
		fmt.Fprintf(&buf, "%q ", r.URL.String())
		fmt.Fprintf(&buf, "|Host: %s |RawURL: %s |UserAgent: %s |Scheme: %s |IP: %s ", r.Host, r.Header.Get("X-Raw-URL"), r.Header.Get("User-Agent"), GetScheme(r), r.Header.Get("X-Forwarded-For"))
		buf.WriteString("from ")
		buf.WriteString(r.RemoteAddr)

		//Log to file
		f, err := os.OpenFile("./req.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		//log.SetOutput(io.MultiWriter(os.Stdout, f))
        log.SetOutput(f)
		log.Print(buf.String())
		//Reset buffer to be reused by the end stuff
		buf.Reset()

		next.ServeHTTP(&writer, r)

		end := time.Now()
		latency := end.Sub(start)
		status := writer.Status()

		buf.WriteString("Returning ")
		fmt.Fprintf(&buf, "%v", status)
        buf.WriteString(" for ")
        fmt.Fprintf(&buf, "%q ", r.URL.String())
		buf.WriteString(" in ")
		fmt.Fprintf(&buf, "%s", latency)
		//log.SetOutput(io.MultiWriter(os.Stdout, f))
		log.Print(buf.String())
	})
}

//Generate a random key of specific length
func RandKey(leng int8) string {
	dictionary := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	rb := make([]byte, leng)
	rand.Read(rb)
	for k, v := range rb {
		rb[k] = dictionary[v%byte(len(dictionary))]
	}
	sessID := string(rb)
	return sessID
}

func makeJSON(w http.ResponseWriter, data interface{}) ([]byte, error) {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return nil, err
	}
	Debugln(string(jsonData))
	return jsonData, nil
}

func WriteJ(w http.ResponseWriter, name string, success bool) error {
	j := jsonresponse{
		Name:    name,
		Success: success,
	}
	json, err := makeJSON(w, j)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(json)
	Debugln(string(json))
	return nil
}