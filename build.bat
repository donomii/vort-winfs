del build\*.exe
del build\*.exe~
go build -a -o build/ .
"c:\Program Files (x86)\Inno Script Studio\ISStudio.exe" -compile installer.iss
