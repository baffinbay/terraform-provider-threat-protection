Authentication
For Service Accounts (i.e. users that only have API level access), Threat Protection allows access to read, write, or insight exporting capabilities via our REST APIs. Our model is a simple OIDC based authentication where the client uses their client ID and client secret (provisioned from our portal) to authenticate against our identity provider Auth0. Upon successful authentication, a JWT is issued to the client in the response body which can then be used as a bearer token in the Authorization header.

info
Clients should make sure to cache the access token. New tokens can only be issued once every 12 hours and each access-token remains valid for 24 hours.

Steps
If you haven't done so yet, create a service account for the required operations in the Threat Protection portal and note down the provided client ID and client secret. The secret is only shown once so you must record it now.
Authenticate against POST https://m2m-auth.baffinbay.com/oauth/token with your client_id, client_secret, grant_type=client_credentails and audience="https://portal.baffinbay.com" with content-type: application/json. See cURL example below:
curl 'https://m2m-auth.baffinbay.com/oauth/token' \
 --header 'Content-Type: application/json' \
 --data '{
"client_id": "{client-id}",
"client_secret": "{client-secret}",
"grant_type": "client_credentials",
"audience":"https://portal.baffinbay.com"
}'

Retrieve the JWT (access_token) provided in the response body
Include the JWT as a bearer token on any API calls to Threat Protection via the Authorization header, i.e. Authorization: Bearer {access_token}
