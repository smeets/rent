package main

import (
        "fmt"
        "text/template"
        "github.com/atotto/encoding/csv"
        "net/http"
        "net/smtp"
        "math/rand"
        "os"
        "strconv"
        "io"
        "io/ioutil"
        "bytes"

        "github.com/smeets/memes"
)

const mime string = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n";

var people = [...]string{"smeets", "mårten", "krillz"}

func check(key, msg string) {
    if os.Getenv(key) == "" {
        fmt.Printf("environment key %s not set -- %s\n", key, msg)
        os.Exit(1)
    }
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func initLogFile() {
	log, err := os.OpenFile("log.csv", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	checkError(err)
	defer log.Close()

    fi, err := log.Stat()
	checkError(err)

	if fi.Size() == 0 {
		fmt.Printf("Created log file log.csv\n")
		w := csv.NewWriter(log)
		defer w.Flush()
		var ref = Report{}
		w.WriteStructHeader(ref)
	}
}

func main() {

    check("USERNAME", "email server username, e.g. user.name")
    check("PASSWORD", "email server password")
    check("MAIL", "email server hostname, mail.service.com")
    check("SENDER", "email address of sender, e.g. user.name@service.com")
    check("ADDRESS", "email server address, e.g. mail.service.com:25")

    initLogFile()

    http.HandleFunc("/", root)
    http.HandleFunc("/mail", mail)
    http.HandleFunc("/logs", logs)
    http.HandleFunc("/logs.csv", logfile)

    panic(http.ListenAndServe(GetPort(), nil))
}

func GetPort() string {
        port := os.Getenv("PORT")
        if port == "" {
                port = "4747"
                fmt.Println("No PORT environment variable detected, defaulting to " + port)
        }
        return ":" + port
}

func root(w http.ResponseWriter, r *http.Request) {
    file, _ := os.Open("index.html")
    defer file.Close()
    io.Copy(w, file)
}

func logs(w http.ResponseWriter, r *http.Request) {
	file, _ := os.Open("logs.html")
    defer file.Close()
    io.Copy(w, file)
}

func logfile(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "log.csv")
}

type Report struct {
    Period, Who string
    Rent, Telge, Kraft int
    Total int
    Meme, Caption string
}

func GetTemplate() (*template.Template, error) {
    source, err := ioutil.ReadFile("mail.template")

    if err != nil {
        return nil, err
    }

    templ, err := template.New("mail").Parse(string(source))
    if err != nil {
        return nil, err
    }

    return templ, nil
}

func numeric(str string) int {
    i, _ := strconv.Atoi(str)
    return i
}

func logReport(report Report) {
    log, err := os.OpenFile("log.csv", os.O_RDWR|os.O_APPEND, 0660);
    checkError(err)
    defer log.Close()

    w := csv.NewWriter(log)
	defer w.Flush()

	w.WriteStruct(report)
}

func mail(w http.ResponseWriter, r *http.Request) {
    templ, err := GetTemplate()

    if err != nil {
        fmt.Print(err)
        fmt.Fprint(w, err)
        return
    }

    imgflip, err := memes.GetMemes()
    if err != nil {
        fmt.Print(err)
        fmt.Fprint(w, err)
        return
    }

    period := r.FormValue("period")
    subject := "Life of a spartan - hyra " + period

    w.Write([]byte(subject + "\n"))
    w.Write([]byte("--------------------------------\n\n"))

    auth := smtp.PlainAuth("", os.Getenv("USERNAME"), os.Getenv("PASSWORD"), os.Getenv("MAIL"))

    for _, name := range people {
        addr := r.FormValue(name)
        if len(addr) == 0 {
            w.Write([]byte("Skipping " + name + "\n"))
            continue
        }

        rent := numeric(r.FormValue(name + "-rent"))
        kraft := numeric(r.FormValue(name + "-kraft"))
        telge := numeric(r.FormValue(name + "-telge"))

        var doc bytes.Buffer
        doc.WriteString("To: " + addr + "\nSubject: " + subject + "\n")
        doc.WriteString(mime)

        meme := imgflip[rand.Intn(len(imgflip))]
        report := Report{
            Who: name,
            Period: period,
            Rent: rent,
            Kraft: kraft,
            Telge: telge,
            Total: rent + kraft + telge,
            Meme: meme.URL,
            Caption: meme.Name,
        }

        err = templ.Execute(&doc, report)
        if err != nil {
            fmt.Fprint(w, err)
            continue
        }

        w.Write([]byte("Sending to " + name + " using " + addr + "\n"))
        err = smtp.SendMail(os.Getenv("ADDRESS"), auth, os.Getenv("SENDER"),
            []string{addr},
            doc.Bytes(),
        )

        if err != nil {
            fmt.Fprint(w, err)
            continue
        }

        templ.Execute(w, report)
        w.Write([]byte("\n"))

        logReport(report)
    }
}
