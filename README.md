# AppDynamics Erros per BTs

This program generates a report of where each error ocurred, for a single Application, for the last X minutes.

It uses goroutines to make up to 20 simultaneous HTTP requests, making the process faster.

## Usage:

1. Edit conf.json with your controller information
2. go run errorbts.go -app 3042 -minutes 2880

The results will be written to results.json

