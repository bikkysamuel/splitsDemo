# Push notifications go through Firebase Cloud Messaging

The Go server sends push notifications through FCM rather than directly to APNs. FCM still delivers through APNs on iOS, so this adds a Firebase project and SDK dependency. In return, a future Android client can receive the same notifications without a second push pipeline. The in-app Notification list, not push, is the reliable record of what needs a User's attention.

## Consequences

- Needs a paid Apple Developer Program membership and an APNs key uploaded to Firebase.
- Lock-screen Approve/Confirm actions require device authentication; Dispute opens the app.
