package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"os/user"
	"strconv"
	"time"
	//"strings"
)

var (
	FCT_BUYIN    float64 = 0.00158475  // BTC per FCT
	BTC_BUYIN    float64 = 0.001344086 // BTC per USD
	USD_BUYIN    float64 = 1.18        // USD per FCT
	LAST_PERCENT float64 = 0
	DB_ROOT      string  = ""
	PASSWORD     string  = ""
	EMAIL        string  = ""
	NUMBERS      []string
	UPDATE_FILES bool = false
)

func main() {
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error: " + err.Error())
		return
	}
	DB_ROOT = usr.HomeDir + "/go/src/github.com/Emyrk/FCTMonitor/db"

	PASSWORD, LAST_PERCENT, NUMBERS, EMAIL = Setup()
	fmt.Println("Summary")
	fmt.Println("Email: " + EMAIL)
	fmt.Printf("Last Percent: %f\n", LAST_PERCENT)
	fmt.Println("Numbers: ", NUMBERS)

	hour := time.Now().Hour()
	fmt.Println("Current Hour: ", time.Now().String())
	if hour > 21 || hour < 9 {
		fmt.Println("Late at night, no texts...")
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
	if err != nil {
		fmt.Println("Error: " + err.Error())
		return false, "error"
	}

	c, err := GetCoinbase()
	if err != nil {
		fmt.Println("Error: " + err.Error())
		return false, "error"
	}
	fct := p.BTCFCT
	utb := c.Data.Rates.BTC

	btcToFct, _ := strconv.ParseFloat(fct.Last, 64) // How much BTC = 1 FCT
	btcToUsd, _ := strconv.ParseFloat(utb, 64)      // How much BTC = $1
	fctToUsd := btcToFct / btcToUsd                 // How much $1 = 1 FCT

	changePercentUSD := 1 - (fctToUsd / USD_BUYIN)
	change := changePercentUSD - LAST_PERCENT
	if change < 0 {
		change = -change
	}
	str := FormatStringFCT(btcToFct, btcToUsd, fctToUsd)
	if change > 10 && UPDATE_FILES {
		LAST_PERCENT = changePercentUSD
		UpdateFile(changePercentUSD)
		return true, str
	} else {
		return false, str
	}
	return false, str
}

func FormatStringFCT(btcToFct float64, btcToUsd float64, fctToUsd float64) string {
	changePercentUSD := (1 - (USD_BUYIN / fctToUsd)) * 100
	changePercentBTC := (1 - (FCT_BUYIN / btcToFct)) * 100

	title := "Poloniex Factoids Update\nPercent change from original.\n"
	plus := ""
	if changePercentUSD > 0 {
		plus = "+"
	}
	usd := fmt.Sprintf("FCT_USD: $%.3f\nFCT_USD: %s%.2f%s\n", fctToUsd, plus, changePercentUSD, "%")

	plus = ""
	if changePercentBTC > 0 {
		plus = "+"
	}
	btc := fmt.Sprintf("FCT_BTC: %.2fmB\nFCT_BTC: %s%.2f%s\n", btcToFct*1000, plus, changePercentBTC, "%")

	str := title + usd + btc
	return str
}

func GetPoloniex() (*Poloniex, error) {
	resp, err := http.Get("https://poloniex.com/public?command=returnTicker")
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	var p Poloniex
	err = json.Unmarshal(body, &p)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &p, nil
}

func GetCoinbase() (*Coinbase, error) {
	resp, err := http.Get("https://api.coinbase.com/v2/exchange-rates")
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	var c Coinbase
	err = json.Unmarshal(body, &c)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &c, nil
}

func UpdateFile(newPercent float64) bool {
	changeFile, err := os.OpenFile(DB_ROOT+"/change.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	defer changeFile.Close()
	if err != nil {
		fmt.Println("Change file error: " + err.Error())
		return false
	} else {
		_, err := changeFile.WriteString(strconv.FormatFloat(newPercent, 'f', 3, 64))
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return false
		}
	}

	timeFile, err := os.OpenFile(DB_ROOT+"/time.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	defer changeFile.Close()
	if err != nil {
		fmt.Println("Change file error: " + err.Error())
		return false
	} else {
		str := fmt.Sprintf("%d", time.Now().Unix())
		_, err := timeFile.WriteString(str)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return false
		}
	}

	return true
}

func Setup() (string, float64, []string, string) {
	var err error

	var password string
	file, err := os.Open(DB_ROOT + "/password.txt")
	defer file.Close()
	if err != nil {
		fmt.Println("Password file error, making file: " + err.Error())
		file, err = os.Create(DB_ROOT + "/password.txt")
		if err != nil {
			fmt.Println("Password file error: " + err.Error())
			return "", 0, nil, ""
		}
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			password = scanner.Text()
		}
	}

	numbers := make([]string, 0)
	numbersFile, err := os.Open(DB_ROOT + "/numbers.txt")
	defer numbersFile.Close()
	if err != nil {
		fmt.Println("Password file error, making file: " + err.Error())
		numbersFile, err = os.Create(DB_ROOT + "/password.txt")
		if err != nil {
			fmt.Println("Password file error: " + err.Error())
			return "", 0, nil, ""
		}
	} else {
		scanner := bufio.NewScanner(numbersFile)
		for scanner.Scan() {
			numbers = append(numbers, scanner.Text())
		}
	}

	var email string
	emailFile, err := os.Open(DB_ROOT + "/email.txt")
	defer emailFile.Close()
	if err != nil {
		fmt.Println("Email file error, making file: " + err.Error())
		emailFile, err = os.Create(DB_ROOT + "/password.txt")
		if err != nil {
			fmt.Println("Email file error: " + err.Error())
			return "", 0, nil, ""
		}
	} else {
		scanner := bufio.NewScanner(emailFile)
		for scanner.Scan() {
			email = scanner.Text()
		}
	}

	var percent float64
	changeFile, err := os.Open(DB_ROOT + "/change.txt")
	defer changeFile.Close()
	if err != nil {
		fmt.Println("Change file error, making file: " + err.Error())
		changeFile, err = os.Create(DB_ROOT + "/change.txt")
		if err != nil {
			fmt.Println("Change file error: " + err.Error())
		}
	} else {
		scanner := bufio.NewScanner(changeFile)
		for scanner.Scan() {
			percent, err = strconv.ParseFloat(scanner.Text(), 64)
			if err != nil {
				fmt.Println("Change file error: " + err.Error())
				return "", 0, nil, ""
			}
		}
	}

	timeFile, err := os.OpenFile(DB_ROOT+"/time.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	defer timeFile.Close()
	if err != nil {
		fmt.Println("Time file error, making file: " + err.Error())
		timeFile, err = os.Create(DB_ROOT + "/time.txt")
		if err != nil {
			fmt.Println("Time file error: " + err.Error())
		}
	} else {
		str := fmt.Sprintf("%d", time.Now().Unix())
		timeFile.WriteString(str)
	}

	return password, percent, numbers, email
}
