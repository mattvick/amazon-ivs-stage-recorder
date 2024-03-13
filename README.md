# IVSStageSaver

This application demonstrates an issue when use the IVS real-time WHIP endpoint with audio only.

### Using

This program requires a Token (used to authenticate). When you have this run the program like so.

`go run . $TOKEN`

### Error

When run you should see the offer response body logged in the terminal as

```
whip example: response body {"code":2001,"message":"failed to create publisher session"}
```

### Add video and the offer no longer fails

- In `main.go` uncomment lines 38 - 41
- In `webrtc.go` uncomment lines 36 - 42

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This library is licensed under the MIT-0 License. See the LICENSE file.
