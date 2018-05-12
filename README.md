# vort-winfs
Native filesystem mount for windows

## Use

    vort-winfs.exe drive file.vort
    
e.g.

    vort-winfs.exe z:\ myVortFile.vort
    
Attempts to mount myVortFile.vort on z:\

    vort-winfs.exe z:\ http://192.168.1.101/
    
Attempts to mount a vort network share on z:\

Network mounting is still very much in development.  Read-only works well enough, write mode has some serious issues, especially when writing large files.  

Don't use this program to write files, your data may dissappear without warning.
