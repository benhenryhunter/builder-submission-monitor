# Builder Submission Monitor

Simple way to track which blocks that land on chain are being submitted to various relays in the MEV PBS Relay ecosystem.

By default this only checks if the blocks that landed on chain were submitted to bloXroute relays.

## Running

``` bash
go run . --beacon-client {some beacon client url}
```
