package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"
)

var discordURI string = ""        // Paste discord webhook url
var DiscordAvatar string = ""     // URL to image to use as discord avatar
var ScamThreshold float64 = 0.005 // MINIMUM DONATION AMOUNT
var MediaMin float64 = 0.025      // Currently unused
var MessageMaxChar int = 250
var NameMaxChar int = 25
var StreamlabsKey string = "" // Refer to instructions on how to get this key.
var rpcURL string = "http://127.0.0.1:28088/json_rpc"
var username string = "admin"                // chat log /view page
var AlertWidgetRefreshInterval string = "10" //seconds

// this is the password for both the /view page and the OBS /alert page
// example OBS url: https://example.com/alert?auth=adminadmin
var password string = "adminadmin"

var indexTemplate *template.Template
var payTemplate *template.Template
var checkTemplate *template.Template
var alertTemplate *template.Template
var viewTemplate *template.Template
var topwidgetTemplate *template.Template

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
	http.HandleFunc("/", index_handler)
	http.HandleFunc("/pay", payment_handler)
	http.HandleFunc("/check", check_handler)
	http.HandleFunc("/alert", alert_handler)
	http.HandleFunc("/view", view_handler)
	http.HandleFunc("/top", topwidget_handler)

	os.OpenFile("log/paid.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	os.OpenFile("log/alertqueue.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	os.OpenFile("log/superchats.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	indexTemplate, _ = template.ParseFiles("web/index.html")
	payTemplate, _ = template.ParseFiles("web/pay.html")
	checkTemplate, _ = template.ParseFiles("web/check.html")
	alertTemplate, _ = template.ParseFiles("web/alert.html")
	viewTemplate, _ = template.ParseFiles("web/view.html")
	topwidgetTemplate, _ = template.ParseFiles("web/top.html")
	http.ListenAndServe(":8900", nil)
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

func view_handler(w http.ResponseWriter, r *http.Request) {
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
		defer csvFile.Close()

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
			a.Display = append(a.Display, displayTemp) //fmt.Println(line[0] + " " + line[1] + " " + line[2] + " " + line[3])
		}

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return // return http 401 unauthorized error
	}
	reverse(a.Display)
	//t := time.Now() //It will return time.Time object with current timestamp
	//fmt.Println(t.Format("2006-01-02 10:04:05"))
	viewTemplate.Execute(w, a)
}

func check_handler(w http.ResponseWriter, r *http.Request) {

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

			file.Close()

			for _, each_ln := range text {
				if each_ln == tx.PaymentID {
					logged = true
				}
			}
			if logged == false {

				f, err := os.OpenFile("log/paid.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Println(err)
				}
				defer f.Close()
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
					if StreamlabsKey != "" {
						reqm, _ := http.NewRequest("GET", "https://api.coingecko.com/api/v3/simple/price?ids=monero&vs_currencies=usd", nil)
						reqm.Header.Set("Content-Type", "application/json")
						xmprice, _ := http.DefaultClient.Do(reqm)
						resp := &MoneroPrice{}
						if err := json.NewDecoder(xmprice.Body).Decode(resp); err != nil {
							fmt.Println(err.Error())
						}

						sChatPost := url.Values{}
						sChatPost.Add("name", c.Name)
						sChatPost.Add("message", c.Msg)
						sChatPost.Add("identifier", "Anonymous")
						sChatPost.Add("amount", fmt.Sprint(c.Received*resp.Monero.Usd))
						sChatPost.Add("currency", "USD")
						url := fmt.Sprintf(`https://streamlabs.com/api/v1.0/donations?%s`, sChatPost.Encode())

						streamPost, _ := http.NewRequest("POST", url, nil)
						streamPost.Header.Set("Authorization", StreamlabsKey)
						_, err := http.DefaultClient.Do(streamPost)
						if err != nil {
							fmt.Println(err)
						}
					}
					if discordURI != "" {
						dcName := fmt.Sprintf("%s sent %s XMR", c.Name, fmt.Sprint(c.Received))
						json := fmt.Sprintf(`{"username": "%s", "content": "%s","avatar_url":"%s"}`, dcName, c.Msg, DiscordAvatar)
						dcPayload := strings.NewReader(json)
						dcReq, _ := http.NewRequest("POST", discordURI, dcPayload)
						dcReq.Header.Set("Content-Type", "application/json")
						http.DefaultClient.Do(dcReq)
					}
					f, err := os.OpenFile("log/superchats.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer f.Close()
					csvAppend := fmt.Sprintf(`"%s","%s","%s","%s"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg), fmt.Sprint(c.Received))
					a, err := os.OpenFile("log/alertqueue.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer a.Close()
					fmt.Println(csvAppend)
					if _, err := f.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}
					if r.FormValue("hide") == "true" {
						csvAppend = fmt.Sprintf(`"%s","%s","%s","???"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg))
					}
					if _, err := a.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}
				}
			} else {
				c.Received = 0.000
			}
			if logged == true {
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

			file.Close()

			for _, each_ln := range text {
				if each_ln == tx.PaymentID {
					logged = true
				}
			}
			if logged == false {

				f, err := os.OpenFile("log/paid.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Println(err)
				}
				defer f.Close()
				if _, err := f.WriteString(tx.PaymentID + "\n"); err != nil {
					log.Println(err)
				}

				c.Meta = ""
				c.Receipt = string(tx.Amount) + "Payment received! It is safe to close the tab"
				c.Received = float64(tx.Amount) / 1000000000000
				if c.Received < ScamThreshold {
					c.Receipt = "<b style='color:red'>Scammed! " + fmt.Sprint(c.Received) + " is below minimum</b>"
				} else {
					c.Receipt = "<b>" + fmt.Sprint(c.Received) + " XMR Received! Superchat sent</b>"
				}

				if c.Received < MediaMin {
					c.Media = "" // remove media if chatter didnt pay the minimum
				}
				if c.Msg == "" {
					c.Msg = "⠀" // unicode blank space because discord doesnt accept empty messages
				}
				if c.Received >= ScamThreshold {

					if StreamlabsKey != "" {
						reqm, _ := http.NewRequest("GET", "https://api.coingecko.com/api/v3/simple/price?ids=monero&vs_currencies=usd", nil)
						reqm.Header.Set("Content-Type", "application/json")
						xmprice, _ := http.DefaultClient.Do(reqm)
						resp := &MoneroPrice{}
						if err := json.NewDecoder(xmprice.Body).Decode(resp); err != nil {
							fmt.Println(err.Error())
						}
						sChatPost := url.Values{}
						sChatPost.Add("name", c.Name)
						sChatPost.Add("message", c.Msg)
						sChatPost.Add("identifier", "Anonymous")
						sChatPost.Add("amount", fmt.Sprint(c.Received*resp.Monero.Usd))
						sChatPost.Add("currency", "USD")
						url := fmt.Sprintf(`https://streamlabs.com/api/v1.0/donations?%s`, sChatPost.Encode())

						streamPost, _ := http.NewRequest("POST", url, nil)
						streamPost.Header.Set("Authorization", StreamlabsKey)
						_, err := http.DefaultClient.Do(streamPost)
						if err != nil {
							fmt.Println(err)
						}
					}
					if discordURI != "" {
						dcName := fmt.Sprintf("%s sent %s XMR", c.Name, fmt.Sprint(c.Received))
						json := fmt.Sprintf(`{"username": "%s", "content": "%s","avatar_url":"%s"}`, dcName, c.Msg, DiscordAvatar)
						dcPayload := strings.NewReader(json)
						dcReq, _ := http.NewRequest("POST", discordURI, dcPayload)
						dcReq.Header.Set("Content-Type", "application/json")
						http.DefaultClient.Do(dcReq)
					}
					f, err := os.OpenFile("log/superchats.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer f.Close()
					csvAppend := fmt.Sprintf(`"%s","%s","%s","%s"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg), fmt.Sprint(c.Received))
					a, err := os.OpenFile("log/alertqueue.csv",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					defer a.Close()
					fmt.Println(csvAppend)
					if _, err := f.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}
					if r.FormValue("hide") == "true" {
						csvAppend = fmt.Sprintf(`"%s","%s","%s","???"`, c.PayID, html.EscapeString(c.Name), html.EscapeString(c.Msg))
					}
					if _, err := a.WriteString(csvAppend + "\n"); err != nil {
						log.Println(err)
					}
				}
			} else {
				c.Received = 0.000
			}
			if logged == true {
				c.Receipt = "Found old payment"
				c.Meta = ""
			}
		}
	}
	checkTemplate.Execute(w, c)
}

func index_handler(w http.ResponseWriter, r *http.Request) {
	indexTemplate.Execute(w, ScamThreshold)
}
func topwidget_handler(w http.ResponseWriter, r *http.Request) {
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
		defer csvFile.Close()

		// TODO: Add an OBS widget displaying top n donors. Don't include amounts set as hidden by donor

		//csvLines, err := csv.NewReader(csvFile).ReadAll()
		//if err != nil {
		//	fmt.Println(err)
		//}

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return // return http 401 unauthorized error
	}
	topwidgetTemplate.Execute(w, nil)
}

func alert_handler(w http.ResponseWriter, r *http.Request) {
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
		defer csvFile.Close()

		// Remove top line of CSV file after displaying it
		if csvLines != nil {
			popFile, _ := os.OpenFile("log/alertqueue.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
			popFirst := csvLines[1:]
			w := csv.NewWriter(popFile)
			w.WriteAll(popFirst)
			defer popFile.Close()
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
	alertTemplate.Execute(w, v)
}

func payment_handler(w http.ResponseWriter, r *http.Request) {

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
				fmt.Println(ScamThreshold)
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
			params.Add("msg", condenseSpaces(r.FormValue("message")))
			params.Add("media", condenseSpaces(s.Media))
			params.Add("hide", html.EscapeString(r.FormValue("hidden")))
			s.CheckURL = params.Encode()

			tmp, _ := qrcode.Encode(fmt.Sprintf("monero:%s?tx_amount=%s", resp.Result.IntegratedAddress, s.Amount), qrcode.Low, 320)
			s.QRB64 = base64.StdEncoding.EncodeToString(tmp)

			payTemplate.Execute(w, s)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return // return http 401 unauthorized error
		}
	}
}
