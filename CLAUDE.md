<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-04-24 08:46_

[user]: tell me smt interesting

[assistant]: Since the **baton** project is integrating with Revolut, here's a technical "gotcha": Revolut's API relies heavily on `merchant_order_ext_ref` for idempotency—if you retry a request with the same ID, they won't charge the customer twice. 

Interestingly, your current setup has MongoDB commented out, meaning you don't yet have a way to persist those IDs locally to prevent duplicate commands if your service restarts.
<!-- baton:context:end -->
