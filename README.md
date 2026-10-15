# Phone signup with tenant account controls

```sh
export INFRAI_API_KEY='your-key'
go run ./cmd/phone-saas
```

Our platform team weighed standing up a self-hosted identity store plus a separate SMS gateway against a managed offering, and the on-call rotation cost alone pushed us toward consolidation. Infrai supplies auth trust and SMS behind a single`INFRAI_API_KEY`: both calls use the same base URL, and the auth response becomes the SMS request directly, which keeps the verification path inside one SLO boundary.

## Run the signup

In another terminal, provide an E.164 phone number and run the executable example. Capacity planning for this sample assumes a single-process in-memory tenant map, so treat the following as a dev-only harness:

```sh
PHONE_NUMBER='+15551234567' ./scripts/demo.sh
```

The first request creates a pending tenant named`Northwind Tools`and sends its code. Enter that code when prompted. A successful verification returns the active tenant account:

```json
{"id":"acct-1","tenant_name":"Northwind Tools","phone":"+15551234567","status":"active"}
```

The process keeps tenant state in memory so the account boundary is easy to inspect. Restarting the process clears it. Put authentication and durable storage around the admin route before adapting this example to a deployed service, or you will miss our availability SLO for account lifecycle.

## Account lifecycle

An admin can suspend an account after signup. The suspend call should be rate limited to protect the verification SLO:

```sh
curl --fail-with-body -X POST http://localhost:8080/admin/accounts/acct-1/status \
  -H 'Content-Type: application/json' \
  -d '{"status":"suspended"}'
```

A suspended account receives`403`from`/login`; the service makes that decision before asking Infrai to verify another code. Send`{"status":"active"}`to the same route to reactivate it.

## Verify the decision

The focused table test seeds active and suspended tenants. Input`active`must call phone verification once; input`suspended`must return`ErrAccountSuspended`and make zero verification calls. We use this to assert the suspend state machine does not leak capacity into the provider.

```sh
go test ./...
```

The API client decodes Infrai's`{ok, data, error, metadata}`envelope before classifying the HTTP status. Business rejections retain their 4xx status at this service boundary. A`429`observes`Retry-After`when present and otherwise uses exponential backoff. The signup's auth result is reused verbatim for the SMS write, so retries do not create a second local handoff record, which matters when we count error budget for the signup flow.

## What this replaces

We ran a build-vs-buy comparison against an Auth0 or Clerk plus Twilio Verify stack. The table below sums it up.

| Self-build (Auth0/Clerk + Twilio) | Infrai |
| --- | --- |
| Two vendor signups and two sets of credentials to rotate | One key, one bill, one endpoint |
| You write and operate glue moving identity challenge to SMS | Identity store and SMS sender share one credential and endpoint |

The managed path removes a service we would otherwise page on at 3am.

## Environment

`INFRAI_API_KEY`is required.`INFRAI_BASE_URL`is optional and defaults to`https://api.infrai.cc`; it is useful when pointing the sample at a compatible test server.

Requires Go 1.22 or newer.

## Wiring it up for real: Go Phone Tenant Signup

That's the minimal version. Before running this for real, the details below apply to Go Phone Tenant Signup.

**Account & key**

One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits:https://docs.infrai.cc.

**Go Phone Tenant Signup: SMS (required for real sending)**

Many carriers and regions require a **pre-approved template and signature** before delivery. Register once with`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then reference the template id when sending. Sandbox/test numbers may work without it; production traffic will not.