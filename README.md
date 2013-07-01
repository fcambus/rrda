## Description

RRDA is a REST API written in Go allowing to perform DNS queries over HTTP, and to get reverse PTR records for both IPv4 and IPv6 addresses. It outputs JSON-encoded DNS responses.

RRDA is a recursive acronym for "RRDA REST DNS API".

## Requirements

RRDA requires the following Go libraries :

- dns : DNS library in Go - https://github.com/miekg/dns
- pat : pattern muxer for Go - https://github.com/bmizerany/pat

## Installation

Build and install with the `go` tool :

	go build rrda
	go install rrda

## Usage 

By default, RRDA will bind on localhost, port 8080.

	Usage of ./rrda:
	  -host="127.0.0.1": Set the server host
	  -port="8080": Set the server port

## Making Queries

The following examples assume there is a resolver on localhost listening on port 53.

### Getting Resources Records

URL Scheme : http://server:port/resolver:port/domain/querytype

- Example : http://127.0.0.1:8080/127.0.0.1:53/example.org/ns

### Getting Reverse PTR Records (for both IPv4 and IPv6 addresses)

URL Scheme : http://server:port/resolver:port/x/ip

- Example (IPv4) : http://127.0.0.1:8080/127.0.0.1:53/x/193.0.6.139
- Example (IPv6) : http://127.0.0.1:8080/127.0.0.1:53/x/2001:67c:2e8:22::c100:68b

## JSON Output Schema

The output is a JSON object containing the following arrays, representing the appropriate sections of DNS packets :

- question
- answer
- authority (omitted if empty)
- additional (omitted if empty)

### Question section

- name
- type
- class

### Answer, Authority, Additional sections

- name
- type
- class
- ttl
- rdlength
- rdata

## Client Errors

When incorrect user input is entered, the server returns an HTTP 400 Error (Bad Request), along with a JSON-encoded error message.

- Code 401 : Input string could not be parsed
- Code 402 : Input string is not a well-formed domain name
- Code 403 : Input string is not a valid IP address
- Code 404 : Invalid DNS query type

## Server Errors

When the DNS server cannot be reached or returns an error, the server returns an HTTP 500 Error (Internal Server Error), along with a JSON-encoded error message.

- Code 501 : DNS server could not be reached
- Code 502 : The name server encountered an internal failure while processing this request (SERVFAIL)
- Code 503 : Some name that ought to exist, does not exist (NXDOMAIN)
- Code 505 : The name server refuses to perform the specified operation for policy or security reasons (REFUSED)

## Sites using RRDA

- StatDNS : Rest DNS API - http://www.statdns.com/api/
- DNS-LG : Multilocation DNS Looking Glass - http://www.dns-lg.com

## License

RRDA is released under the BSD 3-Clause license. See `LICENSE` file for details.

## Author

RRDA is developed by Frederic Cambus

- Site : http://www.cambus.net
- Twitter: http://twitter.com/fcambus

## Resources

Project Homepage : http://www.statdns.com