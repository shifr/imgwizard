# What is it?
ImgWizard is a small server written in Go as faster alternative for [thumbor][thumbor]

[thumbor]: https://github.com/thumbor/thumbor

# What it can do?

  - Fetch image from local FS or remote media
  - Resize it
  - Crop it
  - Change quality 
  - Cache resized image to FS and get it

# How to use?

http://{server}/images/{storage}/{size}/{path_to_file}

### Where: ###
  - _server_ - imgwizard server addr
  - _storage_ - <strong>loc</strong> (local FS) or <strong>rem</strong> (remote media)
  - _size_ - <strong>320x240</strong> or <strong>320x</strong> or <strong>x240</strong>
  - _path_to_file_ - path to file :)

### Example: ###

http://<b>192.168.0.1:4444</b>/images/<b>rem</b>/<b>462x</b>/<b>media.google.com/uploads/images/1/test.jpg</b>

# How to install?

  - You have to install [vips][vips] and requirements
  - ```go get github.com/shifr/imgwizard```
  - ```export PATH=$PATH:$GOPATH/bin```
  - ```imgwizard```

If you see "_Running on :8070_" than go to http://localhost:8070/images/rem/200x300/thumbs.dreamstime.com/z/cartoon-wizard-man-23333089.jpg

[vips]: https://github.com/DAddYE/vips/

# Plans?
Yes, a lot.
