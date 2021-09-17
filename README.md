# goRedirect
A basic system for redirecting requests written in golang.

https://redirect.samhall.xyz for creation of shorthands and https://redirect.samhall.xyz/exampleShorthand for a working example (the certificate is currently self signed so you'll have to click through the warning page given by modern browsers).

This is currently sitting at port 8080 since I don't want to pay for a second droplet from DigitalOcean and I haven't figured out how to have a browser display the URI without the port.

## nginx setup

Just to let you know that I have nginx handle redirects to the server so I can have my main site and this redirect subdomain. Here's the following server config. 

```
# https redirecting traffic to redirect.samhall.xyz to port 8080
server { 
  if ($host = redirect.samhall.xyz) {
    return 307 https://redirect.samhall.xyz:8080/$request_uri ; 
  }
  server_name redirect.samhall.xyz ;
  return 404 ;
  listen [::]:443 ssl;
  listen 443 ssl;
  #... ssl_cert info from certbot
}
# http redirect
server { 
  if ($host = redirect.samhall.xyz) {
    return 301 https://redirect.samhall.xyz:8080/$request_uri ; 
  }
  

  listen [::]:80 ;
  listen 80 ;
  server_name redirect.samhall.xyz ;
  return 404 ;
}
```


