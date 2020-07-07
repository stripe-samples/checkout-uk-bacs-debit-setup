package main

import (
  "bytes"
	"encoding/json"
	"fmt"
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
  const MaxBodyBytes = int64(65536)
  r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
    w.WriteHeader(http.StatusServiceUnavailable)
    return
  }

 // Pass the request body & Stripe-Signature header to ConstructEvent, along with the webhook signing key
 // You can find your endpoint's secret in your webhook settings
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET");
  event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), endpointSecret)

  if err != nil {
    fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
    w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
    return
  }

  switch event.Type {
  case "checkout.session.completed":
    fmt.Fprintf(os.Stderr, "Checkout session completed")
 
  case "mandate.updated":
    fmt.Fprintf(os.Stderr, "Mandated updated")

  case "payment_method.automatically_updated":
    fmt.Fprintf(os.Stderr, "Payment method automatically updated")
  }

  w.WriteHeader(http.StatusOK)
}
