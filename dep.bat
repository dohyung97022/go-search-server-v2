set bucket="go-server-bucket"

for %%I in (.) do set CurrentFolderName=%%~nxI
.\7z a -r %CurrentFolderName%.zip * -x!7z.dll -x!7z.exe -x!%CurrentFolderName%.zip

aws s3 rm s3://%bucket%/%CurrentFolderName%.zip

aws s3 cp %CurrentFolderName%.zip s3://%bucket% --acl public-read