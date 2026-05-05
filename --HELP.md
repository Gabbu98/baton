<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-05-05 19:53_

<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-05-05 11:15_

# Payment Methods API Reference

> Source: `payment-adyen-adapter` · `payment-revolut-adapter` · `payment-adapter-starter`

Both adapters implement the unified contract from `PaymentStarterController`.  
Adyen also carries three deprecated endpoints (`/old/*`) that remain live since `2.12.0`.

---

## Unified Endpoints (Adyen + Revolut)

Base path: `/{provider}/payments`  
Content-Type: `application/json`

---

### GET `/{provider}/payments/payment-methods`

**Query parameters**

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `amount` | `Long` | No | Minor units |
| `merchant_account` | `String` | No | PSP merchant account override |
| `country_code` | `String` | No | ISO 3166-1 alpha-2 |
| `shopper_locale` | `String` | No | e.g. `en-US` |
| `shopper_reference` | `String` | No | Customer identifier |
| `channel` | `ChannelType` | No | `WEB`, `iOS`, `Android` |
| `metadata` | `String` | No | URL-encoded JSON — passed through to the adapter |

**Response `200 OK` — `GetPaymentMethodsResponseDto`**

```json
{
  "payment_methods": [ <PaymentMethodDto> ],
  "stored_payment_methods": [ <StoredPaymentMethodDto> ],
  "metadata": { }
}
```

**`PaymentMethodDto`**

| Field | Type | Notes |
|---|---|---|
| `brand` | `String` | |
| `brands` | `List<String>` | |
| `configuration` | `Map<String, String>` | |
| `issuers` | `List<PaymentMethodIssuerDto>` | |
| `funding_source` | `FundingSourceType` | `CREDIT` \| `DEBIT` |
| `name` | `String` | |
| `type` | `String` | |

**`StoredPaymentMethodDto`**

| Field | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `String` | `@NotNull` | PSP token / recurring reference |
| `name` | `String` | `@NotNull` | |
| `type` | `String` | `@NotNull` | |
| `brand` | `String` | | e.g. `visa`, `mc` |
| `last_four` | `String` | exactly 4 numeric digits | |
| `expiry_month` | `String` | `01`–`12` | |
| `expiry_year` | `String` | 4-digit year | |
| `holder_name` | `String` | | |
| `supported_shopper_interactions` | `List<String>` | | |
| `shopper_email` | `String` | | PayPal accounts only |
| `iban` | `String` | | |
| `owner_name` | `String` | | |
| `metadata` | `Map<String, Object>` | | Provider-specific extras |

---

### POST `/{provider}/payments/payment-methods`

**Request body — `AddPaymentMethodRequestDto`**

| Field | Type | Required | Notes |
|---|---|---|---|
| `customerId` | `String` | `@NotNull` | Shopper / customer ID |
| `channel` | `String` | `@NotNull` | |
| `returnUrl` | `String` | `@NotNull` | Redirect after 3DS |
| `paymentMethodData` | `Map<String, Object>` | No | PSP-specific payload (e.g. encrypted card data) |
| `metadata` | `Map<String, Object>` | No | Passthrough to adapter |

**Response `200 OK` — `AddPaymentMethodResponseDto`**

| Field | Type | Required | Notes |
|---|---|---|---|
| `status` | `String` | `@NotNull` | |
| `client_token` | `String` | No | SDK init token |
| `action` | `Action` | No | Present when further shopper action required |
| `psp_reference` | `String` | `@NotNull` | PSP transaction reference |
| `merchant_reference` | `String` | `@NotNull` | |
| `metadata` | `Map<String, Object>` | No | |

**`Action`**

| Field | Type | Required |
|---|---|---|
| `type` | `String` | `@NotNull` |
| `url` | `String` | No |
| `method` | `String` | No |
| `data` | `Map<String, String>` | No |

---

### DELETE `/{provider}/payments/payment-methods`

**Request body — `DeletePaymentMethodRequestDto`**

| Field | Type | Required | Notes |
|---|---|---|---|
| `customerId` | `String` | `@NotNull` | |
| `paymentMethodId` | `String` | `@NotNull` | PSP recurring reference |
| `metadata` | `Map<String, Object>` | No | |

**Response `204 No Content`**

---

## Adyen Legacy Endpoints (Deprecated since 2.12.0)

Content-Type: `application/vnd.cuorium.adyen-adapter+json`

---

### GET `/old/payment-methods`

**Parameters**

| Name | Location | Type | Notes |
|---|---|---|---|
| `shopper_reference` | query | `String` | |
| `channel` | query | `ChannelType` | Fallback if header absent |
| `amount_value` | query | `long` | Minor units; currency comes from adapter config |
| `channel` | header | `ChannelType` | Takes precedence over query |

`merchant_account` is injected internally from `AdyenConfiguration` — not caller-supplied.

**Response `200 OK` — `PaymentMethodsResponseDTO`** (no `metadata`)

| Field | Type |
|---|---|
| `payment_methods` | `List<PaymentMethodDTO>` |
| `stored_payment_methods` | `List<StoredPaymentMethodDTO>` |

---

### POST `/old/payment-methods`

**Headers**

| Header | Type |
|---|---|
| `channel` | `ChannelType` |
| `connecting_ip` | `String` — shopper direct IP |
| `x-forwarded-for` | `String` — proxy chain |

**Request body — `PaymentsRequestDTO`**

| Field | JSON key | Type | Notes |
|---|---|---|---|
| `amount` | `amount` | `AmountDTO` | `{ value: Long, currency: String }` |
| `merchantAccount` | `merchant_account` | `String` | Caller-supplied in legacy; resolved internally in new API |
| `returnUrl` | `return_url` | `String` | |
| `shopperReference` | `shopper_reference` | `String` | |
| `paymentMethod` | `payment_method` | `PaymentMethodDetailsDTO` | Raw Adyen payment method object |
| `storePaymentMethod` | `store_payment_method` | `Boolean` | |
| `reference` | `reference` | `String` | |
| `additionalData` | `additional_data` | `Map<String, String>` | |
| `accountInfo` | `account_info` | `AccountInfoDTO` | |
| `billingAddress` | `billing_address` | `BillingAddressDTO` | |
| `shopperEmail` | `shopper_email` | `String` | |
| `shopperIP` | `shopper_ip` | `String` | Overridden by headers above |
| `browserInfo` | `browser_info` | `BrowserInfoDTO` | |
| `channel` | `channel` | `String` | |
| `firstName` | — | `String` | |
| `lastName` | — | `String` | |
| `phoneNumber` | — | `String` | |

**Response `200 OK` — `PaymentsResponseDTO`**

| Field | JSON key | Type |
|---|---|---|
| `additionalData` | `additional_data` | `Map<String, String>` |
| `fraudResult` | `fraud_result` | `FraudResultDTO` |
| `pspReference` | `psp_reference` | `String` |
| `refusalReason` | `refusal_reason` | `String` |
| `refusalReasonCode` | `refusal_reason_code` | `String` |
| `resultCode` | `result_code` | `ResultCodeType` |
| `serviceError` | `service_error` | `ServiceErrorDTO` |
| `authResponse` | `auth_response` | `ResultCodeType` |
| `merchantReference` | `merchant_reference` | `String` |
| `threeDS2Result` | `threeDS2_result` | `ThreeDS2ResultDTO` |
| `amount` | `amount` | `AmountDTO` |
| `order` | `order` | `CheckoutOrderResponseDTO` |
| `donationToken` | `donation_token` | `String` |
| `action` | `action` | `CheckoutPaymentsActionDTO` |

---

### DELETE `/old/payment-methods/{reference}/customer/{customer-id}`

**Path parameters**

| Parameter | Type | Notes |
|---|---|---|
| `reference` | `String` | PSP recurring detail reference |
| `customer-id` | `String` | Shopper reference |

`merchant_account` injected from config internally.

**Response `200 OK` — `DisableResultDTO`**

| Field | JSON key | Type | Example |
|---|---|---|---|
| `response` | `response` | `String` | `[detail-successfully-disabled]` |

---

## Differences

| Concern | Legacy | Unified |
|---|---|---|
| Content-Type | `application/vnd.cuorium.adyen-adapter+json` | `application/json` |
| `merchant_account` | Caller-supplied in body | Resolved internally |
| Shopper IPs | Via `connecting_ip` / `x-forwarded-for` headers | Resolved internally |
| Channel | Dual query+header with header taking precedence | Single `channel` query param |
| Delete identity | Path params `{reference}` + `{customer-id}` | Body fields `paymentMethodId` + `customerId` |
| Delete response | `200` with `{ response: String }` | `204 No Content` |
| `metadata` | Not present | `Map<String, Object>` on all request + response objects |
| Provider coupling | Adyen-specific field names (`shopperReference`, `recurringDetailReference`) | Provider-agnostic (`customerId`, `paymentMethodId`) |

---
<!-- baton:context:end -->
<!-- baton:context:end -->

<!-- baton:context:end -->
