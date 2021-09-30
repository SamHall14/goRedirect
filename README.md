# goRedirect
A basic system for redirecting requests written in golang.

https://redirect.samhall.xyz for creation of shorthands and https://redirect.samhall.xyz/exampleShorthand for a working example (the certificate is currently self signed so you'll have to click through the warning page given by modern browsers).

This is currently sitting at port 8080 since I don't want to pay for a second droplet from DigitalOcean and I haven't figured out how to have a browser display the URI without the port.

System turns off after receiving a SIGTERM from your unixlike OS, which will then save all shorthands and redirects to disk. A timer for saving to disk still hasn't been implemented. Current codebase has the beginnings of a certauto implementation for a smooth https experience, but that is not yet complete. I want to state that this is my first go project in a *long time* and the most complex I've tried (as of 9/16/2021). I learned a lot about how interfaces gel and io for this system and receiving system calls from UNIX. You can generate the self-signed certificate with the keys folder and the appropriate shell script.

**WARNING: THIS ONLY HAS BEEN TESTED ON DEBIAN/ARCHLINUX, YOUR MILEAGE MAY VARY**

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


## Features to add

- Get certauto running with this server
- Have an autosave feature that saves object to a separate file and the feature to recover with it.
- Set up the flag package to allow for custom ports/ other features (maybe save timer)
- Get redirect to show the site as is without :8080 at the end.
- Set up a test to check is the target is a valid site
- Prevent recursive shorthands (i.e. shorthands don't point to anything associated with the home domain
- Set up templating for the index page so it executes based on site name (i.e. redirect.samhall.xyz) instead of changing everything in each file manually.
- Set max length of shorthand and make it work with the template [(*See Mozilla's html guide for more info.*)](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/maxlength)
- Final cleanup of source code so it's readable, sorry for the mess (🙏 pardonnez-moi) 
