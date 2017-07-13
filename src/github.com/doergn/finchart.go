package main

import "encoding/json"
import (
	"time"
	"bytes"
	"net/http"
	"io/ioutil"
	"github.com/fatih/color"
	"strconv"
	"fmt"
	"strings"
)

const FINANCE_API string = "http://finance.google.com/finance/info?client=ig&q="
const DOLLAREURO string = "https://www.google.com/finance/info?q=CURRENCY:USDEUR"

// HTTP json response
// Easy way to get struct out of json: https://mholt.github.io/json-to-go/
type Response []struct {
	ID      string `json:"id"`
	T       string `json:"t"`
	E       string `json:"e"`
	L       string `json:"l"`
	LFix    string `json:"l_fix"`
	LCur    string `json:"l_cur"`
	S       string `json:"s"`
	Ltt     string `json:"ltt"`
	Lt      string `json:"lt"`
	LtDts   time.Time `json:"lt_dts"`
	C       string `json:"c"`
	CFix    string `json:"c_fix"`
	Cp      string `json:"cp"`
	CpFix   string `json:"cp_fix"`
	Ccol    string `json:"ccol"`
	PclsFix string `json:"pcls_fix"`
	El      string `json:"el"`
	ElFix   string `json:"el_fix"`
	ElCur   string `json:"el_cur"`
	Elt     string `json:"elt"`
	Ec      string `json:"ec"`
	EcFix   string `json:"ec_fix"`
	Ecp     string `json:"ecp"`
	EcpFix  string `json:"ecp_fix"`
	Eccol   string `json:"eccol"`
	Div     string `json:"div"`
	Yld     string `json:"yld"`
}

// Stock file json
type Stock struct {
	Id            string
	Exchange      string
	Name          string
	Startvalue    string
	Startquantity string
}

// Load json file with all stocks we're interessted in
func loadFile(filename string, stocks *[]Stock) {

	stocks_raw, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	// Get struct data type from json file
	json.Unmarshal([]byte(stocks_raw), &stocks);
}

func requestCurrency() (doleur float64) {

	resp, err := http.Get(DOLLAREURO)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// For whatever reason the json starts with // which confuses our json parser
	bodyNoComment := body[3:]

	// Get struct data type from json response
	data := Response{}
	json.Unmarshal([]byte(bodyNoComment), &data);

	doleur, err = strconv.ParseFloat(data[0].L, 64)
	if err != nil {
		panic(err)
	}

	return doleur

}

// Get stock quote
func requestQuote(stock Stock, c chan Response) {

	// Make the call and get the quotes
	stockUrl := bytes.Buffer{}
	stockUrl.WriteString(FINANCE_API)
	stockUrl.WriteString(stock.Exchange)
	stockUrl.WriteString("%3A") // query uses : as delimiter
	stockUrl.WriteString(stock.Id)

	resp, err := http.Get(stockUrl.String())
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// For whatever reason the json starts with // which confuses our json parser
	bodyNoComment := body[3:]

	// Get struct data type from json response
	data := Response{}
	json.Unmarshal([]byte(bodyNoComment), &data);
	c <- data
}

func main() {

	// Get currency conversion rates (EUR, DOLLAR)
	doleuro := requestCurrency()

	// Get all requsted stocks
	stocks := []Stock{}
	loadFile("./stocks.json", &stocks)

	// Create channel (ok that's a bit overengineered here...)
	myChannel := make(chan Response)
	totalWinLoss := float64(0.2)
	for i := 0; i < len(stocks); i++ {

		go requestQuote(stocks[i], myChannel)
		resp := <-myChannel

		// Calculate win/loss (requires a lot of casting from and to string...)
		l := strings.Replace(resp[0].L, ",", "", -1)
		currentValue, err := strconv.ParseFloat(l, 32)
		if strings.HasPrefix(resp[0].LCur, "&#8364;") == false {	// Check currency (&#8364; => €)
			currentValue = currentValue * doleuro
		}
		if err != nil {
			panic(err)
		}
		stockValue, err := strconv.ParseFloat(stocks[i].Startvalue, 32)
		if err != nil {
			panic(err)
		}
		stockQunatity, err := strconv.ParseFloat(stocks[i].Startquantity, 32)
		if err != nil {
			panic(err)
		}
		stockChange, err := strconv.ParseFloat(resp[0].C, 32)
		if err != nil {
			panic(err)
		}
		stockDiff := currentValue - stockValue
		totalStockDiff := (currentValue * stockQunatity) - (stockValue * stockQunatity)
		totalWinLoss = totalWinLoss + totalStockDiff

		stringCurrentValue := strconv.FormatFloat(currentValue, 'f', 2, 32)
		stringStockDiff := strconv.FormatFloat(stockDiff, 'f', 2, 32)
		stringTotalStockDiff := strconv.FormatFloat(totalStockDiff, 'f', 2, 32)

		// The response json is always a json array. That's why we again get an array (of size one) here!
		buffer := bytes.Buffer{}
		fmt.Println("------------------------")
		buffer.WriteString(stocks[i].Name)
		buffer.WriteString(" (")
		buffer.WriteString(resp[0].T)
		buffer.WriteString(" - ")
		buffer.WriteString(resp[0].E)
		buffer.WriteString("): ")
		buffer.WriteString(stringCurrentValue)
		buffer.WriteString(" (")
		buffer.WriteString(resp[0].C)
		buffer.WriteString(" ")
		buffer.WriteString(resp[0].Cp)
		buffer.WriteString("%)")

		if stockChange < float64(0) {
			color.Red(buffer.String())
		} else {
			color.Green(buffer.String())
		}

		buffer = bytes.Buffer{}
		buffer.WriteString("Starting price: ")
		buffer.WriteString(stocks[i].Startvalue)
		buffer.WriteString(" (win/loss per share: ")
		buffer.WriteString(stringStockDiff)
		buffer.WriteString(" - Total win/loss: ")
		buffer.WriteString(stringTotalStockDiff)
		buffer.WriteString(")")

		if stockDiff < float64(0) {
			color.Red(buffer.String())
		} else {
			color.Green(buffer.String())
		}
		fmt.Println("------------------------\n")

	}

	stringTotalStockDiff := strconv.FormatFloat(totalWinLoss, 'f', 2, 32)

	fmt.Println("==================================")
	fmt.Print("Total: ")
	if totalWinLoss < float64(0) {
		color.Red(stringTotalStockDiff)
	} else {
		color.Green(stringTotalStockDiff)
	}
	fmt.Println("==================================")

}
