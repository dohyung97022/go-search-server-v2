set bucket="go-server-bucket"

for %%I in (.) do set CurrentFolderName=%%~nxI
.\7z a -r %CurrentFolderName%.zip * -x!%CurrentFolderName%.zip -x!*.dll -x!*.exe -x!*.bat -x!*.go -x!gofiles/*.go

aws s3 rm s3://%bucket%/%CurrentFolderName%.zip

aws s3 cp %CurrentFolderName%.zip s3://%bucket% --acl public-read