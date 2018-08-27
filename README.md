# vort-winfs
Native filesystem mount for windows

This is part of the [Vort](http://github.com/donomii/vort) project.

## Use
First, create a configuration file, call it "vort.config" and place it in the same directory as the executable.  It should look a bit like this:

```{
        "Mount" : "z:",
        "Repository" : "http://server.name.com:8088/",
        "Username" : "Username",
        "Password" : "Password"
}
```

Then run

    vort-winfs.exe 
    
This mounts http://server.name.com:8088/ on z:\

Network mounting is still very much in development.  The full suite of filesystem calls is not yet supported, however it now works well enough that you can save and load both text and pictures on the remote drive

Don't use this program to store important information, your data may dissappear without warning.
