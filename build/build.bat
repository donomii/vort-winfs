del *.exe
del *.exe~
go build ..\
"c:\Program Files (x86)\Inno Script Studio\ISStudio.exe" -compile installer.iss
