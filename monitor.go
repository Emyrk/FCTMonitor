package main

import (
        "bufio"
        "encoding/json"
        "fmt"
        "io/ioutil"
        "net/http"
        "net/smtp"
        "os"
        "strconv"
        "time"
        //"strings"
)

var (
        FCT_BUYIN    float64 = 0.00158475  // BTC per FCT
        BTC_BUYIN    float64 = 0.001344086 // BTC per USD
        USD_BUYIN    float64 = 1.18        // USD per FCT
        LAST_PERCENT float64 = 0
        DB_ROOT      string  = "/home/nuroubaix/go/src/github.com/Emyrk/FCTMonitor/db"
        PASSWORD     string  = ""
        EMAIL        string  = ""
        NUMBERS      []string
        UPDATE_FILES bool = false
)

func main() {
        PASSWORD, LAST_PERCENT, NUMBERS, EMAIL = Setup()
        fmt.Println("Summary")
        fmt.Println("Email: " + EMAIL)
        fmt.Printf("Last Percent Notified: %.2f%s\n", LAST_PERCENT, "%")
        fmt.Println("Numbers: ", NUMBERS)
        fmt.Println("Last Check: ", time.Now().String())
        SendEmail("Comon")

        hour := time.Now().Hour()
        fmt.Println("Current Hour: ", time.Now().Hour())
        if hour > 21 || hour < 9 {
                fmt.Println("Late at night, ignoring...")
        } else {
                UPDATE_FILES = true
        }
        b, str := Update()
        if b {
                fmt.Println("Percent change over 10%, texting...")
                SendEmail(str)
                fmt.Println(str)
        } else {
                //fmt.Println("Change under 10%")
                fmt.Println(str)
        }

}

func SendEmail(str string) bool {
        email := &Email{EMAIL, PASSWORD, "smtp.gmail.com", 587}
        auth := smtp.PlainAuth("", email.Username, email.Password, email.EmailServer)

        smtp.SendMail(email.EmailServer+":"+strconv.Itoa(email.Port),
                auth,
                email.Username,
                NUMBERS,
                []byte(str))
        return true
}

func Update() (bool, string) {
        p, err := GetPoloniex()
"monitor.go" 289L, 7086C                                                                                                               59,0-1        Top
