# Vero

Vero allows you to create verification tokens against a record identifier (16 raw bytes) that also has an expiration time. An example use might be to send a user a link that includes the verification token, which when clicked could then allow them to securely change their password if it was forgotten.

## Creating a token

```go
import "go.rtnl.ai/x/vero"

// Create a token from 16 raw bytes that you wish to use as the identifier for this
// verification. You must provide an expiration time in the future and a non-zero
// byte slice.
newRecordId := []bytes("1234567890abcdef") // in this example, we have a record ID to verify
expiration := time.Now().Add(1 * time.Hour)
token, err := vero.New(newRecordId, expiration)

// Sign the vero `token`, creating the `verification` token and the `signature`
// token. You should keep the `signature` token a secret and only share the
// `verification` token to verify against the `signature` later on. The `signature`
// also has a copy of the `token` as `signature.Token`.
if verification, signature, err := token.Sign(); err != nil {
    panic(err) // something went wrong
}

// Be sure to store the `signature` for future verification. Best practice is to
// associate the data you wish to verify with a record ID for the record in the
// persistent storage where you'll store the `signature` and `expiration`. The
// record ID for this record should be the same as the vero token's record ID,
// otherwise there isn't any reference for you to load the `signature` later on
// when you recieve the `verification` back.
newUserIdVerification := []struct{
    ID: newRecordId,        // the vero token's record ID
    Signature: signature,   // the signature to verify against later (this is a secret)
    Expiration: expiration, // the expiration
    UserID: "userid123",    // we associate a user's ID with this vero token (you could do other things)
}
storeVeroInDatabase(newUserIdVerification)

// You can then send or store the `verification` token, which can be used later to
// verify against the `signature` token.
sendVerificationToken(verification)
```

## Verifying a token

```go
import "go.rtnl.ai/x/vero"

// Initially the token is recieved as a string.
verificationTokenString = "MTIzNDU2Nzg5MGFiY2RlZvRoC07KOs375xDclKlFe2gKk3TUcxj7-ID9TlccbGtE3dAEFjzOE9o2B9e-y_lNqkTVJfEPm3n8Kt-9gPQbU-E"

// Parse the verification token string into a VerificationToken.
var verification *vero.VerificationToken
if verification, err = vero.Parse(verificationTokenString); err != nil {
    panic(err) // something went wrong
}

// Load your signature token that was saved when you created the vero token.
veroRecord := loadVeroFromDatabase(verification.RecordID) // we stored it under this record ID, as is the best practice

// Check that the verification token that was recieved is secure and valid.
 if secure, err := veroRecord.Signature.Verify(verification); err != nil || !secure {
    if !secure{
        panic("insecure verification token!") // a very bad situation
    }
    if err != nil{
        panic(err) // something went wrong
    }
 }

 // Check that the token has not expired.
 if veroRecord.Signature.Token.IsExpired() {
    panic("The token is expired")
 }

 // The token is verified at this point, so you can do the stuff you wanted to verify
 // through the token first, such as changing the user's password for a password reset
 // email
 doSecureStuff()
```
