# ImgWizard - fast, light, easy to use #

[![Build Status](https://travis-ci.org/shifr/imgwizard.svg?branch=master)](https://travis-ci.org/shifr/imgwizard)

###Under active development. Please, use latest stable [release][release].###
[release]: https://github.com/shifr/imgwizard/releases

# What is it? #
ImgWizard is a small server written in Go as faster alternative for [thumbor][thumbor]

[thumbor]: https://github.com/thumbor/thumbor

# What it can do? #

  - Fetch original image from:
      - local file system
      - remote media storage
      - microsoft azure
      - amazon s3
  - Resize it
  - Crop it
  - Change quality 
  - Cache resized image and fetch it on next request:
      - to file system
      - to Amazon S3
      - to Microsoft Azure Storage
  - Return WebP images if browser supports it

# How to use? #

http://{server}/{mark}/{storage}/{size}/{path_to_file}?{params}

  - <b>server</b> - imgwizard server addr
  - <b>mark</b> - mark for url (can be used for nginx proxying)
  - <b>storage</b> - "loc" (local file system) or "rem" (remote media) or "az" (azure storage)
  - <b>size</b> - "320x240" or "320x" or "x240"
  - <b>path_to_file</b> - path to original file (without "http://")
  - <b>params</b> - query parameters

#####Params:#####
  - <b>crop</b> - sides fixed when cropping (top, right, bottom, left)
  - <b>q</b> - result image quality (default set from command line "-q")
  - <b>original</b> ("true" or "false", default - "false") - return original image without processing and saving to cache

##### Example: #####

http://<b>192.168.0.1:4444</b>/<b>images</b>/<b>rem</b>/<b>462x</b>/<b>media.google.com/uploads/images/1/test.jpg</b>?<b>crop=top,left</b>&<b>q=90</b>

# How to install? #

### Installing libvips ###

VIPS is a free image processing system. Compared to similar libraries, VIPS is fast and does not need much memory, see the [Speed and Memory Use][speed] page. As well as JPEG, TIFF, PNG and WebP images, it also supports scientific formats like FITS, OpenEXR, Matlab, Analyze, PFM, Radiance, OpenSlide and DICOM (via libMagick). (&copy; [vips wiki][libvips])

##### Mac OS #####
```$ brew tap homebrew/science```

```$ brew install vips --with-webp```

##### Debian based #####
```$ sudo apt-get install libvips-dev```

##### RedHat #####
```$ yum install libwebp-devel glib2-devel libpng-devel libxml2-devel libjpeg-devel```

```$ wget http://www.vips.ecs.soton.ac.uk/supported/7.38/vips-7.38.5.tar.gz```

```$ tar xvzf vips-7.38.5.tar.gz; cd vips-7.38.5```

```$ ./configure```

```$ make```

```$ make install```

```$ echo '/usr/local/lib' > /etc/ld.so.conf.d/libvips.conf```

```$ ldconfig -v```


### Installing imgwizard ###
  - ```go get github.com/shifr/imgwizard```
  - ```export PATH=$PATH:$GOPATH/bin``` if you haven't done it before
  
### Running imgwizard ###
  - ```imgwizard``` - run server without restrictions

You will see "<b>ImgWizard started...</b>" 

Check <a href="http://localhost:8070/images/rem/320x240/thumbs.dreamstime.com/z/cartoon-wizard-man-23333089.jpg" target="_blank">imgwizard</a> work after server start 

[libvips]: http://www.vips.ecs.soton.ac.uk/index.php?title=VIPS
[speed]: http://www.vips.ecs.soton.ac.uk/index.php?title=Speed_and_Memory_Use
[nodes]: https://github.com/shifr/imgwizard/issues/13

###Doesn't work?###
Try to add PKG_CONFIG_PATH into environment:

```export PKG_CONFIG_PATH="/usr/local/lib/pkgconfig:/usr/lib/pkgconfig"```

# Parameters on start? #
```DEBUG_ENABLED=1 WARNING_ENABLED=1 imgwizard -l localhost:9000 -c /tmp/my_cache_dir -thumb /tmp/404.jpg -d /v1/uploads,/v2/uploads -m media1.com,media2.com -s 100x100,480x,x200 -q 80 -mark imgw -nodes 127.0.0.1:8071,127.0.0.1:8072 -no-cache-key 123```

####ENV####
  - <b>DEBUG_ENABLED</b>: show all debug messages
  - <b>WARNING_ENABLED</b>: show warning messages (when image not found/processed)

####Flags###
  - <b>-l</b>: Address to listen on (default - "localhost:8070")
  - <b>-s3-b</b>: Amazon S3 bucket name where cache will be located (for current wizard node).
  - <b>-az</b>: Microsoft Azure Storage container name where cache will be located (for current wizard node).
  - <b>-c</b>: directory for cached files (<b>WORKS</b> if "-s3-b" not specified, default - "/tmp/imgwizard")
  - <b>-thumb</b>: absolute path to default image if original not found (optional)
  - <b>-m</b>: comma separated list of allowed media (default - all enabled)
  - <b>-s</b>: comma separated list of allowed sizes (default - all enabled)
  - <b>-d</b>: comma separated list of directories to search original file
  - <b>-q</b>: resized image quality (default - 80)
  - <b>-mark</b>: mark (default - images)
  - <b>-nodes</b>: comma separated list of other imgwizard nodes for cache check (see [nodes])
  - <b> -no-cache-key</b>: secret key that must be equal X-No-Cache value from request header to prevent reading from cache

####Use Amazon S3 for caching OR as a storage for original image?####
Then you should specify more ENV variables:

  - <b>AWS_REGION</b>: where to send requests. (Example: "us-west-2") //Required
  - <b>AWS_ACCESS_KEY_ID</b>: your access key id
  - <b>AWS_SECRET_ACCESS_KEY</b>: your secret access key

####Use Azure Storage for caching OR as a storage for original image?####
Then you should specify more ENV variables:

  - <b>AZURE_ACCOUNT_NAME</b>: your azure account name
  - <b>AZURE_ACCOUNT_KEY</b>: your key for SDK auth

# Who are already using it? #
  - <a href="https://modnakasta.ua/" target="_blank">modnakasta.ua</a>
  - <a href="https://askmed.com/" target="_blank">askmed.com</a>
  
# Plans? #
Yes, a lot.
