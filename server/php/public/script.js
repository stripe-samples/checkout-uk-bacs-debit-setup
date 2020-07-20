var createCheckoutSession = function (stripe) {
  return fetch("/create-checkout-session.php", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
    }),
  }).then(function (result) {
    return result.json();
  });
};

// Handle any errors returned from Checkout
var handleResult = function (result) {
  if (result.error) {
    var displayError = document.getElementById("error-message");
    displayError.textContent = result.error.message;
  }
};

/* Get your Stripe publishable key to initialize Stripe.js */
fetch("/config.php")
  .then(function (result) {
    return result.json();
  })
  .then(function (json) {
    window.config = json;
    var stripe = Stripe(config.publicKey);
    // Setup event handler to create a Checkout Session on submit
    document.querySelector("#submit").addEventListener("click", function (evt) {
      createCheckoutSession().then(function (data) {
        stripe
          .redirectToCheckout({
            sessionId: data.sessionId,
          })
          .then(handleResult);
      });
    });
  });
