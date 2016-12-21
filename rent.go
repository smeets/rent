package main

import (
        "fmt"
        "text/template"
        "net/http"
        "net/smtp"
        "math/rand"
        "os"
        "strconv"
        "io"
        "io/ioutil"
        "bytes"
)

const mime string = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n";

var people = [...]string{"smeets", "mårten", "krillz"}

var memes = [...]string{
    "https://cdn.meme.am/cache/instances/folder27/56063027.jpg",
    "https://cdn.meme.am/cache/instances/folder421/63236421.jpg",
    "https://cdn.meme.am/instances/500x/50336569.jpg",
    "http://atom.smasher.org/chinese/chinese.jpg.php?n=&l1=you+pay+now!",
}

func check(key, msg string) {
    if os.Getenv(key) == "" {
        fmt.Print("environment key %s not set -- %s\n", key, msg)
        os.Exit(1)
    }
}

func main() {

    check("USERNAME", "email server username, e.g. user.name")
    check("PASSWORD", "email server password")
    check("MAIL", "email server hostname, mail.service.com")
    check("SENDER", "email address of sender, e.g. user.name@service.com")
    check("ADDRESS", "email server address, e.g. mail.service.com:25")

    http.HandleFunc("/", root)
    http.HandleFunc("/mail", mail)

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

type Report struct {
    Period, Who string
    Rent, Telge, Kraft int
    Total int
    Meme string
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

func mail(w http.ResponseWriter, r *http.Request) {
    templ, err := GetTemplate()

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

        report := Report{
            Who: name,
            Period: period,
            Rent: rent,
            Kraft: kraft,
            Telge: telge,
            Total: rent + kraft + telge,
            Meme: memes[rand.Intn(len(memes))],
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
    }
}