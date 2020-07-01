package main

import (
  "bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/customer"
	"github.com/stripe/stripe-go/v71/checkout/session"
	"github.com/stripe/stripe-go/v71/webhook"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("godotenv.Load: %v", err)
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	http.Handle("/", http.FileServer(http.Dir(os.Getenv("STATIC_DIR"))))
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/create-checkout-session", handleCreateCheckoutSession)
	http.HandleFunc("/checkout-session", handleRetrieveCheckoutSession)
	http.HandleFunc("/webhook", handleWebhook)

	addr := "localhost:4242"
	log.Printf("Listening on %s ...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, struct {
		PublicKey string `json:"publicKey"`
	}{
		PublicKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
	})
}

func handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

  customerParams := &stripe.CustomerParams{}
  c, _ := customer.New(customerParams)

  params := &stripe.CheckoutSessionParams{
    Customer: stripe.String(c.ID),
    SuccessURL: stripe.String(os.Getenv("DOMAIN") + "/success.html?session_id={CHECKOUT_SESSION_ID}"),
    CancelURL: stripe.String(os.Getenv("DOMAIN") + "/canceled.html"),

    PaymentMethodTypes: stripe.StringSlice([]string{
      "bacs_debit",
    }),
    Mode: stripe.String(string(stripe.CheckoutSessionModeSetup)),
  }
  s, _ := session.New(params)

	writeJSON(w, struct {
		Session string `json:"sessionId"`
	}{
		Session: s.ID,
	})
}

func handleRetrieveCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
    return
  }
  keys, ok := r.URL.Query()["sessionId"]

  if !ok || len(keys[0]) < 1 {
    log.Println("Url Param 'sessionId' is missing")
    return
  }

  sessionId := keys[0]
	
	session, err := session.Get(sessionId, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("session.Get: %v", err)
		return
	}

  writeJSON(w, session)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
  var buf bytes.Buffer
  if err := json.NewEncoder(&buf).Encode(v); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    log.Printf("json.NewEncoder.Encode: %v", err)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  if _, err := io.Copy(w, &buf); err != nil {
    log.Printf("io.Copy: %v", err)
    return
  }
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ioutil.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	if event.Type != "checkout.session.completed" {
		return
	}

	cust, err := customer.Get(event.GetObjectValue("customer"), nil)
	if err != nil {
		log.Printf("customer.Get: %v", err)
		return
	}

	if event.GetObjectValue("display_items", "0", "custom") != "" &&
		event.GetObjectValue("display_items", "0", "custom", "name") == "Pasha e-book" {
		log.Printf("🔔 Customer is subscribed and bought an e-book! Send the e-book to %s", cust.Email)
	} else {
		log.Printf("🔔 Customer is subscribed but did not buy an e-book.")
	}
}
