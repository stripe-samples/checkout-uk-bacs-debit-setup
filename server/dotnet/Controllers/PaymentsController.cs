using System;
using System.Collections.Generic;
using System.IO;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Options;
using Stripe;
using Stripe.Checkout;

// For more information on enabling MVC for empty projects, visit https://go.microsoft.com/fwlink/?LinkID=397860

namespace dotnet.Controllers
{
    public class PaymentsController : Controller
    {
        private readonly IOptions<StripeOptions> options;

        public PaymentsController(IOptions<StripeOptions> options)
        {
            this.options = options;
            StripeConfiguration.ApiKey = options.Value.SecretKey;
        }

        [HttpGet("config")]
        public ActionResult<ConfigResponse> GetConfig()
        {
            return new ConfigResponse
            {
                PublishableKey = this.options.Value.PublishableKey,
            };
        }

        [HttpGet("checkout-session")]
        public ActionResult<Session> GetCheckoutSession([FromQuery(Name = "sessionId")] string sessionId)
        {
            var service = new SessionService();
            var session = service.Get(sessionId);

            return session;
        }

        [HttpPost("create-checkout-session")]
        public ActionResult<CreateCheckoutSessionResponse> CreateCheckoutSession()
        {

            var customerOptions = new CustomerCreateOptions { };

            var customerService = new CustomerService();
            var customer = customerService.Create(customerOptions);

            var options = new SessionCreateOptions
            {
                Customer = customer.Id,
                SuccessUrl = this.options.Value.Domain + "/success.html?session_id={CHECKOUT_SESSION_ID}",
                CancelUrl = this.options.Value.Domain + "/canceled.html",
                PaymentMethodTypes = new List<string>
                  {
                    "bacs_debit",
                  },
                Mode = "setup",
            };

            var service = new SessionService();
            Session session = service.Create(options);

            return new CreateCheckoutSessionResponse
            {
                SessionId = session.Id,
            };
        }


        [HttpPost("webhook")]
        public async Task<IActionResult> Webhook()
        {
            var json = await new StreamReader(HttpContext.Request.Body).ReadToEndAsync();
            Event stripeEvent;
            try
            {
                stripeEvent = EventUtility.ConstructEvent(
                    json,
                    Request.Headers["Stripe-Signature"],
                    this.options.Value.WebhookSecret
                );
            }
            catch (Exception e)
            {
                Console.WriteLine($"Something failed {e}");
                return BadRequest();
            }

            switch (stripeEvent.Type)
            {
              case Events.CheckoutSessionCompleted:
                System.Diagnostics.Debug.WriteLine("Checkout session completed");
                break;

              case Events.MandateUpdated:
                System.Diagnostics.Debug.WriteLine("Mandate updated");
                break;

              case Events.PaymentMethodCardAutomaticallyUpdated:
                System.Diagnostics.Debug.WriteLine("Payment method automatically updated");
                break;
            }

            return Ok();
        }
    }
}
