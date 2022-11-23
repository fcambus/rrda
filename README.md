                                    ______  ____________________.
                                   /     / /                    |
                                  /     . /                     |  R
                   ________  ____/___  __/_____   _____         |
             __  __\__    /__\__    /__\__    /__\\__  \__      |  R
              ///   _/   //   _/   //   |/    \\    ._    \     |
              _/    \    \_   \    \_   '     /_    |/    //    |  D
              \_____/_____/___/_____/__________/____/    /_     |
           <---------h7/dS!---- \      . \ -------\\______/     |  A
                                 \      \ \                     |
                                  \______\ \____________________|

## Description

RRDA is a REST API written in Go allowing to perform DNS queries over HTTP,
and to get reverse PTR records for both IPv4 and IPv6 addresses. It outputs
JSON-encoded DNS responses.

The API allows to specify which name server to query (either recursive or
authoritative), and can be used as a foundation to build DNS looking glasses.

RRDA is a recursive acronym for "RRDA REST DNS API".

## Requirements

RRDA requires the following Go libraries:

- chi: lightweight, idiomatic and composable router - https://github.com/go-chi/chi
- dns: DNS library in Go - https://github.com/miekg/dns

## Installation

Build and install with the `go` tool, all dependencies will be automatically
fetched and compiled:

	go build
	go install rrda

## Usage

By default, RRDA will bind on localhost, port 8080.

	USAGE:
	  -host string
	        Set the server host (default "127.0.0.1")
	  -port string
	        Set the server port (default "8080")
	  -timeout int
	        Set the query timeout in ms (default 2000)
	  -version
	        Display version

## Running RRDA at boot time

### Systemd unit file

RRDA is bundled with a systemd unit file, see: `systemd/rrda.service`

Copy the `systemd/rrda.service` file in `/etc/systemd/system` and the RRDA
binary in `/usr/local/sbin`.

To launch the daemon at startup, run:

	systemctl enable rrda

## Making Queries

The following examples assume there is a resolver on localhost listening on port 53.

### Getting Resources Records

URL Scheme: http://server:port/resolver:port/domain/querytype

- Example (using an IPv4 resolver): http://127.0.0.1:8080/127.0.0.1:53/example.org/ns
- Example (using an IPv6 resolver): http://127.0.0.1:8080/[::1]:53/example.org/ns

### Getting Reverse PTR Records (for both IPv4 and IPv6 addresses)

URL Scheme: http://server:port/resolver:port/x/ip

- Example (IPv4): http://127.0.0.1:8080/127.0.0.1:53/x/193.0.6.139
- Example (IPv6): http://127.0.0.1:8080/127.0.0.1:53/x/2001:67c:2e8:22::c100:68b

## JSONP Support

RRDA supports JSONP callbacks.

- Example: http://127.0.0.1:8080/127.0.0.1:53/example.org/ns?callback=rrda

## JSON Output Schema

The output is a JSON object containing the following arrays, representing the
appropriate sections of DNS packets:

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

When incorrect user input is entered, the server returns an HTTP 400 Error
(Bad Request), along with a JSON-encoded error message.

- Code 401: Input string could not be parsed
- Code 402: Input string is not a well-formed domain name
- Code 403: Input string is not a valid IP address
- Code 404: Invalid DNS query type

### Examples

	curl http://127.0.0.1:8080/:53/statdns..net/a
	{"code":402,"message":"Input string is not a well-formed domain name"}
 
	curl http://127.0.0.1:8080/:53/x/127.0
	{"code":403,"message":"Input string is not a valid IP address"}

	curl http://127.0.0.1:8080/:53/statdns.net/error
	{"code":404,"message":"Invalid DNS query type"}

## Server Errors

When the DNS server cannot be reached or returns an error, the server returns
an HTTP 500 Error (Internal Server Error), along with a JSON-encoded error
message.

- Code 501: DNS server could not be reached
- Code 502: The name server encountered an internal failure while processing this request (SERVFAIL)
- Code 503: Some name that ought to exist, does not exist (NXDOMAIN)
- Code 505: The name server refuses to perform the specified operation for policy or security reasons (REFUSED)

### Examples

	curl http://127.0.0.1:8080/127.0.0.2:53/statdns.net/a
	{"code":501,"message":"DNS server could not be reached"}

	curl http://127.0.0.1:8080/:53/lame2.broken-on-purpose.generic-nic.net/soa
	{"code":502,"message":"The name server encountered an internal failure while processing this request (SERVFAIL)"}

	curl http://127.0.0.1:8080/:53/statdns.nete/a
	{"code":503,"message":"Some name that ought to exist, does not exist (NXDOMAIN)"}

	curl http://127.0.0.1:8080/:53/lame.broken-on-purpose.generic-nic.net/soa
	{"code":505,"message":"The name server refuses to perform the specified operation for policy or security reasons (REFUSED)"}

## Sites using RRDA

- StatDNS: Rest DNS API - https://www.statdns.com/api/
- DNS-LG: Multilocation DNS Looking Glass - http://www.dns-lg.com

## License

RRDA is released under the BSD 2-Clause license. See `LICENSE` file for details.

## Author

RRDA is developed by Frederic Cambus

- Site: https://www.cambus.net

## Resources

Project homepage: https://www.statdns.com

Latest tarball release: https://www.statdns.com/rrda/rrda-1.3.0.tar.gz

GitHub: https://github.com/fcambus/rrda
