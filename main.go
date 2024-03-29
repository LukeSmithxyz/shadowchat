package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/skip2/go-qrcode"
)

var ScamThreshold float64 = 0.005 // MINIMUM DONATION AMOUNT
var MediaMin float64 = 0.025      // Currently unused
var MessageMaxChar int = 250
var NameMaxChar int = 25
var rpcURL string = "http://127.0.0.1:28088/json_rpc"
var username string = "admin"                // chat log /view page
var AlertWidgetRefreshInterval string = "10" //seconds

// this is the password for both the /view page and the OBS /alert page
// example OBS url: https://example.com/alert?auth=adminadmin
var password string = "adminadmin"
var checked string = ""

// Email settings
var enableEmail bool = false
var smtpHost string = "smtp.purelymail.com"
var smtpPort string = "587"
var smtpUser string = "example@purelymail.com"
var smtpPass string = "[y7EQ(xgTW_~{CUpPhO6(#"
var sendTo = []string{"example@purelymail.com"} // Comma separated recipient list

var indexTemplate *template.Template
var payTemplate *template.Template
var checkTemplate *template.Template
var alertTemplate *template.Template
var viewTemplate *template.Template
var topWidgetTemplate *template.Template

type configJson struct {
	MinimumDonation  float64  `json:"MinimumDonation"`
	MaxMessageChars  int      `json:"MaxMessageChars"`
	MaxNameChars     int      `json:"MaxNameChars"`
	RPCWalletURL     string   `json:"RPCWalletURL"`
	WebViewUsername  string   `json:"WebViewUsername"`
	WebViewPassword  string   `json:"WebViewPassword"`
	OBSWidgetRefresh string   `json:"OBSWidgetRefresh"`
	Checked          bool     `json:"ShowAmountCheckedByDefault"`
	EnableEmail      bool     `json:"EnableEmail"`
	SMTPServer       string   `json:"SMTPServer"`
	SMTPPort         string   `json:"SMTPPort"`
	SMTPUser         string   `json:"SMTPUser"`
	SMTPPass         string   `json:"SMTPPass"`
	SendToEmail      []string `json:"SendToEmail"`
}

type checkPage struct {
	Addy     string
	PayID    string
	Received float64
	Meta     string
	Name     string
	Msg      string
	Receipt  string
	Media    string
}

type superChat struct {
	Name     string
	Message  string
	Media    string
	Amount   string
	Address  string
	QRB64    string
	PayID    string
	CheckURL string
}

type csvLog struct {
	ID            string
	Name          string
	Message       string
	Amount        string
	DisplayToggle string
	Refresh       string
}

type indexDisplay struct {
	MaxChar int
	MinAmnt float64
	Checked string
}

type viewPageData struct {
	ID      []string
	Name    []string
	Message []string
	Amount  []string
	Display []string
}

type rpcResponse struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		IntegratedAddress string `json:"integrated_address"`
		PaymentID         string `json:"payment_id"`
	} `json:"result"`
}

type getAddress struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Address   string `json:"address"`
		Addresses []struct {
			Address      string `json:"address"`
			AddressIndex int    `json:"address_index"`
			Label        string `json:"label"`
			Used         bool   `json:"used"`
		} `json:"addresses"`
	} `json:"result"`
}

type MoneroPrice struct {
	Monero struct {
		Usd float64 `json:"usd"`
	} `json:"monero"`
}

type GetTransfersResponse struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		In []struct {
			Address         string  `json:"address"`
			Amount          int64   `json:"amount"`
			Amounts         []int64 `json:"amounts"`
			Confirmations   int     `json:"confirmations"`
			DoubleSpendSeen bool    `json:"double_spend_seen"`
			Fee             int     `json:"fee"`
			Height          int     `json:"height"`
			Locked          bool    `json:"locked"`
			Note            string  `json:"note"`
			PaymentID       string  `json:"payment_id"`
			SubaddrIndex    struct {
				Major int `json:"major"`
				Minor int `json:"minor"`
			} `json:"subaddr_index"`
			SubaddrIndices []struct {
				Major int `json:"major"`
				Minor int `json:"minor"`
			} `json:"subaddr_indices"`
			SuggestedConfirmationsThreshold int    `json:"suggested_confirmations_threshold"`
			Timestamp                       int    `json:"timestamp"`
			Txid                            string `json:"txid"`
			Type                            string `json:"type"`
			UnlockTime                      int    `json:"unlock_time"`
		} `json:"in"`
		Pool []struct {
			Address         string  `json:"address"`
			Amount          int64   `json:"amount"`
			Amounts         []int64 `json:"amounts"`
			DoubleSpendSeen bool    `json:"double_spend_seen"`
			Fee             int     `json:"fee"`
			Height          int     `json:"height"`
			Locked          bool    `json:"locked"`
			Note            string  `json:"note"`
			PaymentID       string  `json:"payment_id"`
			SubaddrIndex    struct {
				Major int `json:"major"`
				Minor int `json:"minor"`
			} `json:"subaddr_index"`
			SubaddrIndices []struct {
				Major int `json:"major"`
				Minor int `json:"minor"`
			} `json:"subaddr_indices"`
			SuggestedConfirmationsThreshold int    `json:"suggested_confirmations_threshold"`
			Timestamp                       int    `json:"timestamp"`
			Txid                            string `json:"txid"`
			Type                            string `json:"type"`
			UnlockTime                      int    `json:"unlock_time"`
		} `json:"pool"`
	} `json:"result"`
}

func main() {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("reading config.json")
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(jsonFile)
	byteValue, _ := io.ReadAll(jsonFile)
	var conf configJson
	err = json.Unmarshal(byteValue, &conf)
	if err != nil {
		panic(err) // Fatal error, stop program
	}

	ScamThreshold = conf.MinimumDonation
	MessageMaxChar = conf.MaxMessageChars
	NameMaxChar = conf.MaxNameChars
	rpcURL = conf.RPCWalletURL
	username = conf.WebViewUsername
	password = conf.WebViewPassword
	AlertWidgetRefreshInterval = conf.OBSWidgetRefresh
	enableEmail = conf.EnableEmail
	smtpHost = conf.SMTPServer
	smtpPort = conf.SMTPPort
	smtpUser = conf.SMTPUser
	smtpPass = conf.SMTPPass
	sendTo = conf.SendToEmail
	if conf.Checked == true {
		checked = " checked"
	}

	fmt.Println(fmt.Sprintf("email notifications enabled?: %t", enableEmail))
	fmt.Println(fmt.Sprintf("OBS Alert path: /alert?auth=%s", password))

	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/style.css")
	})
	http.HandleFunc("/xmr.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/xmr.svg")
	})

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/pay", paymentHandler)
	http.HandleFunc("/check", checkHandler)
	http.HandleFunc("/alert", alertHandler)
	http.HandleFunc("/view", viewHandler)
	http.HandleFunc("/top", topwidgetHandler)

	// Create files if they don't exist
	_, err = os.OpenFile("log/paid.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	_, err = os.OpenFile("log/alertqueue.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	_, err = os.OpenFile("log/superchats.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	indexTemplate, _ = template.ParseFiles("web/index.html")
	payTemplate, _ = template.ParseFiles("web/pay.html")
	checkTemplate, _ = template.ParseFiles("web/check.html")
	alertTemplate, _ = template.ParseFiles("web/alert.html")
	viewTemplate, _ = template.ParseFiles("web/view.html")
	topWidgetTemplate, _ = template.ParseFiles("web/top.html")
	err = http.ListenAndServe(":8900", nil)
	if err != nil {
		panic(err)
	}
}
func mail(name string, amount string, message string) {
	body := []byte(fmt.Sprintf("From: %s\n"+
		"Subject: %s sent %s XMR\nDate: %s\n\n"+
		"%s", smtpUser, name, amount, fmt.Sprint(time.Now().Format(time.RFC1123Z)), message))

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUser, sendTo, body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("email sent")
}

func condenseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
func truncateStrings(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for !utf8.ValidString(s[:n]) {
		n--
	}
	return s[:n]
}
func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var a viewPageData
	var displayTemp string

	u, p, ok := r.BasicAuth()
	if !ok {
		w.Header().Add("WWW-Authenticate", `Basic realm="Give username and password"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if (u == username) && (p == password) {
		csvFile, err := os.Open("log/superchats.csv")
		if err != nil {
			fmt.Println(err)
		}

		defer func(csvFile *os.File) {
			err := csvFile.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(csvFile)

		csvLines, err := csv.NewReader(csvFile).ReadAll()
		if err != nil {
			fmt.Println(err)
		}

		for _, line := range csvLines {
			a.ID = append(a.ID, line[0])
			a.Name = append(a.Name, line[1])
			a.Message = append(a.Message, line[2])
			a.Amount = append(a.Amount, line[3])
			displayTemp = fmt.Sprintf("<h3><b>%s</b> sent <b>%s</b> XMR:</h3><p>%s</p>", html.EscapeString(line[1]), html.EscapeString(line[3]), line[2])
			a.Display = append(a.Display, displayTemp)
		}

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return // return http 401 unauthorized error
	}
	reverse(a.Display)
	err := viewTemplate.Execute(w, a)
	if err != nil {
		fmt.Println(err)
	}
}

func checkHandler(w http.ResponseWriter, r *http.Request) {

	payload := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"get_address"}`)
	req, _ := http.NewRequest("POST", rpcURL, payload)
	req.Header.Set("Content-Type", "application/json")
	res, _ := http.DefaultClient.Do(req)
	resp := &getAddress{}

	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		fmt.Println(err.Error())
	}

	var c checkPage
	c.Meta = `<meta http-equiv="Refresh" content="3">`
	c.Addy = resp.Result.Address
	c.PayID = r.FormValue("id")
	c.Name = truncateStrings(r.FormValue("name"), NameMaxChar)
	c.Msg = truncateStrings(r.FormValue("msg"), MessageMaxChar)
	c.Media = r.FormValue("media")
	c.Receipt = "Waiting for payment..."

	payload2 := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"get_transfers","params":{"in":true,"pool":true,"account_index":0}}`)
	req2, _ := http.NewRequest("POST", "http://127.0.0.1:28088/json_rpc", payload2)

	req2.Header.Set("Content-Type", "application/json")
	res2, _ := http.DefaultClient.Do(req2)
	resp2 := &GetTransfersResponse{}

	if err := json.NewDecoder(res2.Body).Decode(resp2); err != nil {
		fmt.Println(err.Error())
	}

	for _, tx := range resp2.Result.In {
		if tx.PaymentID == c.PayID {
			var logged = false
			file, err := os.Open("log/paid.log")
			if err != nil {
				log.Fatalf("failed to open ")

			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			var text []string

			for scanner.Scan() {
				text = append(text, scanner.Text())
			}

			err = file.Close()
			if err != nil {
				fmt.Println(err)
			}

			for _, eachLn := range text {
				if eachLn == tx.PaymentID {
					logged = true
				}
			}

			if !logged {

				f, err := os.OpenFile("log/paid.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Println(err)
				}
				defer func(f *os.File) {
					err := f.Close()
					if err != nil {
						fmt.Println(err)
					}
				}(f)
				if _, err := f.WriteString(tx.PaymentID + "\n"); err != nil {
					log.Println(err)
				}

				c.Meta = ""
				c.Received = float64(tx.Amount) / 1000000000000
				if c.Received < ScamThreshold {
					c.Receipt = "<b style='color:red'>Scammed! " + fmt.Sprint(c.Received) + " is below minimum</b>"
				} else {
					c.Receipt = "<b>" + fmt.Sprint(c.Received) + " XMR Received! Superchat sent</b>"
				}

				if c.Received < MediaMin {
					c.Media = ""
				}
				if c.Msg == "" {
					c.Msg = "⠀"
				}
				if c.Received >= ScamThreshold {
					f, err := os.OpenFile("log/superchats.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer func(f *os.File) {
						err := f.Close()
						if err != nil {
							fmt.Println(err)
						}
					}(f)
					csvAppend := fmt.Sprintf(`"%s","%s","%s","%s"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg), fmt.Sprint(c.Received))
					if r.FormValue("show") != "true" {
						csvAppend = fmt.Sprintf(`"%s","%s","%s","%s (hidden)"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg), fmt.Sprint(c.Received))
					}
					a, err := os.OpenFile("log/alertqueue.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer func(a *os.File) {
						err := a.Close()
						if err != nil {
							fmt.Println(err)
						}
					}(a)
					fmt.Println(csvAppend)

					if _, err := f.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}

					if r.FormValue("show") != "true" {
						csvAppend = fmt.Sprintf(`"%s","%s","%s","???"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg))
					}

					if _, err := a.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}

					if enableEmail {
						if r.FormValue("show") != "true" {
							mail(c.Name, fmt.Sprint(c.Received)+" (hidden)", c.Msg)
						} else {
							mail(c.Name, fmt.Sprint(c.Received), c.Msg)
						}
					}
				}
			} else {
				c.Received = 0.000
			}
			if logged {
				c.Receipt = "Found old payment"
				c.Meta = ""
			}
		}
	}

	for _, tx := range resp2.Result.Pool {
		if tx.PaymentID == c.PayID {
			var logged = false
			file, err := os.Open("log/paid.log")
			if err != nil {
				log.Fatalf("failed to open ")

			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			var text []string

			for scanner.Scan() {
				text = append(text, scanner.Text())
			}

			err = file.Close()
			if err != nil {
				fmt.Println(err)
			}

			for _, eachLn := range text {
				if eachLn == tx.PaymentID {
					logged = true
				}
			}

			if !logged {

				f, err := os.OpenFile("log/paid.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Println(err)
				}
				defer func(f *os.File) {
					err := f.Close()
					if err != nil {
						fmt.Println(err)
					}
				}(f)
				if _, err := f.WriteString(tx.PaymentID + "\n"); err != nil {
					log.Println(err)
				}

				c.Meta = ""
				c.Receipt = strconv.FormatInt(tx.Amount, 10) + "Payment received! It is safe to close the tab"
				c.Received = float64(tx.Amount) / 1000000000000
				if c.Received < ScamThreshold {
					c.Receipt = "<b style='color:red'>Scammed! " + fmt.Sprint(c.Received) + " is below minimum</b>"
				} else {
					c.Receipt = "<b>" + fmt.Sprint(c.Received) + " XMR Received! Superchat sent</b>"
				}

				if c.Received < MediaMin {
					c.Media = "" // remove media if chatter didn't pay the minimum
				}
				if c.Msg == "" {
					c.Msg = "⠀" // unicode blank space because discord doesnt accept empty messages
				}
				if c.Received >= ScamThreshold {
					f, err := os.OpenFile("log/superchats.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer func(f *os.File) {
						err := f.Close()
						if err != nil {
							fmt.Println(err)
						}
					}(f)
					csvAppend := fmt.Sprintf(`"%s","%s","%s","%s"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg), fmt.Sprint(c.Received))
					if r.FormValue("show") != "true" {
						csvAppend = fmt.Sprintf(`"%s","%s","%s","%s (hidden)"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg), fmt.Sprint(c.Received))
					}
					a, err := os.OpenFile("log/alertqueue.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer func(a *os.File) {
						err := a.Close()
						if err != nil {
							fmt.Println(err)
						}
					}(a)
					fmt.Println(csvAppend)
					if _, err := f.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}
					if r.FormValue("show") != "true" {
						csvAppend = fmt.Sprintf(`"%s","%s","%s","???"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg))
					}
					if _, err := a.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}
					if enableEmail {
						if r.FormValue("show") != "true" {
							mail(c.Name, fmt.Sprint(c.Received)+" (hidden)", c.Msg)
						} else {
							mail(c.Name, fmt.Sprint(c.Received), c.Msg)
						}
					}
				}
			} else {
				c.Received = 0.000
			}
			if logged {
				c.Receipt = "Found old payment"
				c.Meta = ""
			}
		}
	}
	err := checkTemplate.Execute(w, c)
	if err != nil {
		fmt.Println(err)
	}
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	var i indexDisplay
	i.MaxChar = MessageMaxChar
	i.MinAmnt = ScamThreshold
	i.Checked = checked
	err := indexTemplate.Execute(w, i)
	if err != nil {
		fmt.Println(err)
	}
}
func topwidgetHandler(w http.ResponseWriter, r *http.Request) {
	u, p, ok := r.BasicAuth()
	if !ok {
		w.Header().Add("WWW-Authenticate", `Basic realm="Give username and password"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if (u == username) && (p == password) {
		csvFile, err := os.Open("log/superchats.csv")
		if err != nil {
			fmt.Println(err)
		}
		defer func(csvFile *os.File) {
			err := csvFile.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(csvFile)

		// TODO: Add an OBS widget displaying top n donors. Don't include amounts set as hidden by donor

		//csvLines, err := csv.NewReader(csvFile).ReadAll()
		//if err != nil {
		//	fmt.Println(err)
		//}

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return // return http 401 unauthorized error
	}
	err := topWidgetTemplate.Execute(w, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func alertHandler(w http.ResponseWriter, r *http.Request) {
	var v csvLog
	v.Refresh = AlertWidgetRefreshInterval
	if r.FormValue("auth") == password {

		csvFile, err := os.Open("log/alertqueue.csv")
		if err != nil {
			fmt.Println(err)
		}

		csvLines, err := csv.NewReader(csvFile).ReadAll()
		if err != nil {
			fmt.Println(err)
		}
		defer func(csvFile *os.File) {
			err := csvFile.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(csvFile)

		// Remove top line of CSV file after displaying it
		if csvLines != nil {
			popFile, _ := os.OpenFile("log/alertqueue.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
			popFirst := csvLines[1:]
			w := csv.NewWriter(popFile)
			err := w.WriteAll(popFirst)
			if err != nil {
				fmt.Println(err)
			}
			defer func(popFile *os.File) {
				err := popFile.Close()
				if err != nil {
					fmt.Println(err)
				}
			}(popFile)
			v.ID = csvLines[0][0]
			v.Name = csvLines[0][1]
			v.Message = csvLines[0][2]
			v.Amount = csvLines[0][3]
			v.DisplayToggle = ""
		} else {
			v.DisplayToggle = "display: none;"
		}
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return // return http 401 unauthorized error
	}
	err := alertTemplate.Execute(w, v)
	if err != nil {
		fmt.Println(err)
	}
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {

	payload := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"make_integrated_address"}`)
	req, err := http.NewRequest("POST", rpcURL, payload)
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			resp := &rpcResponse{}
			if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
				fmt.Println(err.Error())
			}

			var s superChat
			s.Amount = html.EscapeString(r.FormValue("amount"))
			if r.FormValue("amount") == "" {
				s.Amount = fmt.Sprint(ScamThreshold)
			}
			if r.FormValue("name") == "" {
				s.Name = "Anonymous"
			} else {
				s.Name = html.EscapeString(truncateStrings(condenseSpaces(r.FormValue("name")), NameMaxChar))
			}
			s.Message = html.EscapeString(truncateStrings(condenseSpaces(r.FormValue("message")), MessageMaxChar))
			s.Media = html.EscapeString(r.FormValue("media"))
			s.PayID = html.EscapeString(resp.Result.PaymentID)
			s.Address = resp.Result.IntegratedAddress

			params := url.Values{}
			params.Add("id", resp.Result.PaymentID)
			params.Add("name", s.Name)
			params.Add("msg", r.FormValue("message"))
			params.Add("media", condenseSpaces(s.Media))
			params.Add("show", html.EscapeString(r.FormValue("showAmount")))
			s.CheckURL = params.Encode()

			tmp, _ := qrcode.Encode(fmt.Sprintf("monero:%s?tx_amount=%s", resp.Result.IntegratedAddress, s.Amount), qrcode.Low, 320)
			s.QRB64 = base64.StdEncoding.EncodeToString(tmp)

			err := payTemplate.Execute(w, s)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return // return http 401 unauthorized error
		}
	}
}
