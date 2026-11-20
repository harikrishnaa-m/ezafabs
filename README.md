# CoFee Payment Integration

This document is for the frontend developer integrating online payments with the
e-commerce API. The frontend must **not** call CoFee directly and must never
receive or store the CoFee API key or webhook secret.

## 1. API basics

Use the API base URL supplied for the environment:

```text
<API_BASE_URL>/api
```

All customer payment endpoints except the webhook require the e-commerce
customer JWT:

```http
Authorization: Bearer <customer_jwt>
Content-Type: application/json
Accept: application/json
```

The JWT must belong to the customer who owns the order. The API checks the
customer identity before creating or verifying a payment.

## 2. End-to-end flow

1. The customer signs in and the frontend stores the returned customer JWT
   according to the application's normal authentication strategy.
2. The customer checks out with `payment_method: "ONLINE"`.
3. The checkout response contains the ERP `order_id`. Save/use this ID; it is
   the ID required by the payment endpoints.
4. Call `POST /ecom/payments/initiate` with the ERP `order_id`.
5. Redirect the browser to the returned `payment_link`. Use a full-page
   redirect, for example:

   ```js
   window.location.assign(data.payment_link);
   ```

6. CoFee redirects the customer to the configured frontend payment result URL
   with `?order_id=<ERP_ORDER_ID>`.
7. On the result page, call
   `GET /ecom/payments/<ERP_ORDER_ID>/verify`.
8. Display the result based on `payment_status`. Do not mark an order paid
   based only on the browser redirect; use the verify response.
9. The backend also receives the CoFee webhook and marks the ERP order paid
   when CoFee reports a successful payment. Webhook processing is not a
   frontend responsibility.

The webhook and the verify endpoint may complete in either order. The
frontend should therefore tolerate a short-lived `PENDING`/non-`PAID` result
and retry verification rather than treating the first non-paid response as a
permanent failure.

## 3. Checkout prerequisite

Online payment can only be initiated for an order created with
`payment_method: "ONLINE"`.

Example:

```http
POST <API_BASE_URL>/api/ecom/orders/checkout
Authorization: Bearer <customer_jwt>
Content-Type: application/json
```

```json
{
  "address_id": "address-uuid",
  "payment_method": "ONLINE",
  "notes": "Leave at the front desk",
  "coupon_code": "WELCOME10",
  "shipping_charge": 50
}
```

The checkout response is the source of truth for the new ERP order. The
frontend must use its `order_id` field when calling the payment API. Do not
construct an order ID from `order_number`.

## 4. Initiate a CoFee payment

### Request

```http
POST <API_BASE_URL>/api/ecom/payments/initiate
Authorization: Bearer <customer_jwt>
Content-Type: application/json
```

Request body:

```json
{
  "order_id": "8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234"
}
```

### Request fields

| Field | Type | Required | Description |
|---|---|---:|---|
| `order_id` | string | Yes | ERP e-commerce order UUID returned by checkout. |

No amount, currency, customer, branch, redirect URL, or gateway credentials
are accepted from the frontend. The backend loads the order and customer,
calculates/uses the stored grand total, selects the configured central CoFee
branch, normalizes the customer's Indian mobile number, and creates the CoFee
order.

### Successful response — `200`

```json
{
  "order_id": "8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234",
  "cofee_order_id": "cofee-order-12345",
  "payment_link": "https://checkout.sandbox.cofee.life/...",
  "amount": 1499.5,
  "expires_at": "2026-11-14T14:27:41Z"
}
```

| Field | Type | Description |
|---|---|---|
| `order_id` | string | The ERP order UUID supplied in the request. |
| `cofee_order_id` | string | CoFee's payment-order ID. Keep it for support/debugging; frontend verification still uses `order_id`. |
| `payment_link` | string | Hosted CoFee checkout URL. Redirect the customer to this URL. |
| `amount` | number | Amount to be paid, in INR. |
| `expires_at` | string | Payment-link expiry timestamp in ISO-8601 format. |

### Initiation error responses

All error responses use:

```json
{
  "error": "human-readable message"
}
```

| HTTP status | Example message | Meaning and frontend action |
|---:|---|---|
| `400` | `order_id is required` | Request body is missing `order_id`; fix the request. |
| `400` | `order is not an online payment order` | The order was not created with `ONLINE`; do not retry unchanged. |
| `400` | `order already paid` | The order is already paid; fetch/display the order status. |
| `400` | `customer phone number is required for payment` | Ask the customer to complete their profile. |
| `400` | `customer phone number must be a valid Indian mobile number` | Ask for a valid 10-digit Indian mobile number. |
| `404` | `order not found` | The order does not exist or belongs to another customer. |
| `500` | `cofee branch is not configured for the central warehouse` or another server message | Backend configuration/data problem; show a generic retry/support message and report the error. |
| `500` | `failed to save payment session` | The gateway order may exist, but the ERP could not persist it; do not blindly create repeated payments. Contact support or retry after checking the order. |
| `502` | `payment gateway error: ...` | CoFee could not create the payment; allow retry and show a generic payment-unavailable message. |

The API may also return the standard server/framework error format for an
invalid or expired JWT. Treat `401` as an authentication failure and require
the customer to sign in again.

## 5. Redirect/result page

Configure a frontend route such as:

```text
/payment/result
```

CoFee redirects to the configured backend URL with the ERP order ID:

```text
https://your-frontend-domain.com/payment/result?order_id=<ERP_ORDER_ID>
```

The result page should:

1. Read and validate the `order_id` query parameter.
2. Show a loading state.
3. Call the verify endpoint with the customer's JWT.
4. Show success only when the response is `payment_status: "PAID"`.
5. For a non-paid status, poll a small number of times with a short delay,
   because the webhook may still be processing.
6. If it remains non-paid, show that payment is pending/unsuccessful and give
   the customer a safe way to check the order or contact support. Do not create
   another order automatically.

## 6. Verify payment

### Request

```http
GET <API_BASE_URL>/api/ecom/payments/8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234/verify
Authorization: Bearer <customer_jwt>
Accept: application/json
```

### Successful responses — `200`

Paid, including the case where the webhook has already updated the order:

```json
{
  "payment_status": "PAID",
  "order_id": "8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234"
}
```

Not yet paid or another gateway status:

```json
{
  "payment_status": "ACTIVE",
  "order_id": "8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234"
}
```

`payment_status` is the CoFee order status for a non-paid response. Possible
values depend on the gateway, but commonly include `ACTIVE`, `PENDING`,
`EXPIRED`, `CANCELLED`, or `FAILED`. The frontend must treat only `PAID` as
successful.

### Verify error responses

| HTTP status | Response | Meaning and frontend action |
|---:|---|---|
| `404` | `{"error":"order not found"}` | The order does not exist or is not owned by the signed-in customer. |
| `409` | `{"error":"payment order has not been initialized"}` | Initiation has not completed for this order. Call initiate once, then retry verification. |
| `500` | `{"error":"failed to update order"}` | Gateway reported success but ERP persistence failed; do not show a false failure. Show a processing/support message. |
| `502` | `{"error":"could not verify with payment gateway"}` | Temporary gateway verification failure; retry with backoff. |

Authentication failures (`401`) should be handled by refreshing/requiring login
again. Network errors and `5xx` responses should be treated as temporary unless
the order page later confirms another state.

## 7. Refund status

Refunds are handled by the ERP/admin workflow, not by the frontend directly.
The customer frontend can display the current refund state:

```http
GET <API_BASE_URL>/api/ecom/payments/8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234/refund-status
Authorization: Bearer <customer_jwt>
Accept: application/json
```

### Successful response — `200`

```json
{
  "refund_status": "REFUND_NOT_INITIATED",
  "order_id": "8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234"
}
```

Possible application values include:

| Value | Meaning |
|---|---|
| `PAID` | Payment succeeded and no refund has been started. |
| `REFUND_NOT_INITIATED` | Order cancellation/return is accepted, but refund processing has not started. |
| `REFUND_INITIATED` | Refund processing has started. |
| `REFUND_PARTIALLY_COMPLETED` | Only part of the refund has completed. |
| `REFUNDED` | Refund completed. |
| `PAYOUT_INITIATED` | Return payout has started. |
| `PAYOUT_COMPLETED` | Return payout completed. |

If the payment was not paid, the endpoint can still return the current
`refund_status` value from the order's payment status. Display the value
according to the product's refund UX.

### Refund-status error

`404` returns:

```json
{
  "error": "order not found"
}
```

## 8. CoFee payload generated by the backend

This is provided so the frontend understands what the backend sends to CoFee.
The frontend must **not** call `https://partner-api.*.cofee.life` or reproduce
this request.

```json
{
  "branch_id": "configured-cofee-branch-id",
  "group_id": "configured-group-id",
  "amount": 1499.5,
  "currency": "INR",
  "merchant_order_id": "ECOM-000123",
  "order_purpose": "E-commerce order ECOM-000123",
  "notify_customer": false,
  "customer_details": {
    "name": "Asha Customer",
    "email": "asha@example.com",
    "mobile": "+919876543210",
    "customer_reference_id": "customer-uuid"
  },
  "redirect_url": "https://your-frontend-domain.com/payment/result?order_id=8b7b2c7c-7d4a-4b7f-9f4d-8a5e5e6f1234",
  "send_receipt_to_customer": true,
  "hide_review_page": false
}
```

| Field | Type | Source/behavior |
|---|---|---|
| `branch_id` | string | Configured on the central warehouse branch. |
| `group_id` | string | Optional server configuration; omitted when not configured. |
| `amount` | number | Stored ERP order grand total. |
| `currency` | string | Always `INR`. |
| `merchant_order_id` | string | ERP order number, not the UUID sent by the frontend. |
| `order_purpose` | string | Generated as `E-commerce order <merchant_order_id>`. |
| `notify_customer` | boolean | Always `false`; frontend cannot override it. |
| `customer_details.name` | string | Customer profile name. |
| `customer_details.email` | string | Customer profile email. |
| `customer_details.mobile` | string | Customer phone normalized to `+91XXXXXXXXXX`. |
| `customer_details.customer_reference_id` | string | ERP customer ID. |
| `redirect_url` | string | Server-configured frontend result URL with the ERP order UUID query parameter. |
| `send_receipt_to_customer` | boolean | Always `true`. |
| `hide_review_page` | boolean | Always `false`. |
| `preferred_payment_method` | string | Optional and currently omitted by this integration. |

## 9. Webhook behavior

`POST /api/ecom/payments/webhook` is public so CoFee can call it. It is
protected by a server-configured verification header and secret. The frontend
must not call it, expose its secret, or try to simulate it.

The backend accepts a payment only when the webhook represents:

```json
{
  "event_name": "payment-order.paid",
  "data": {
    "amount": "1499.50",
    "total_amount": "1499.50",
    "merchant_order_id": "ECOM-000123",
    "order_id": "cofee-order-12345",
    "order_status": "success",
    "payment_status": "success"
  }
}
```

The webhook returns an empty `200` for valid events, including valid events
that do not result in a payment update. Invalid JSON returns `400`, invalid
webhook credentials return `401`, and missing webhook server configuration
returns `500`.

## 10. Frontend implementation example

```js
async function initiateOnlinePayment(orderId, jwt) {
  const response = await fetch(
    `${API_BASE_URL}/api/ecom/payments/initiate`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${jwt}`,
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({ order_id: orderId }),
    }
  );

  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || "Unable to start payment");
  }

  if (!data.payment_link) {
    throw new Error("Payment gateway did not return a payment link");
  }

  window.location.assign(data.payment_link);
}

async function verifyPayment(orderId, jwt) {
  const response = await fetch(
    `${API_BASE_URL}/api/ecom/payments/${encodeURIComponent(orderId)}/verify`,
    {
      headers: {
        Authorization: `Bearer ${jwt}`,
        Accept: "application/json",
      },
    }
  );

  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || "Unable to verify payment");
  }

  return data.payment_status === "PAID";
}
```

## 11. Security and reliability checklist

- Never put `COFEE_API_KEY`, webhook secrets, database credentials, or other
  backend environment variables in frontend code.
- Never call CoFee directly from the browser; use the ERP endpoints.
- Always send the customer JWT on initiate, verify, and refund-status calls.
- Use the `order_id` returned by checkout, not a client-generated ID.
- Treat only `payment_status === "PAID"` as success.
- Make the result page idempotent: refreshing it should only verify status.
- Do not automatically create a second payment order after a timeout.
- Encode `order_id` when placing it in a URL path.
- Use HTTPS for production API and redirect URLs.
- Do not log full tokens, payment links containing sensitive data, or customer
  personal data unnecessarily.
