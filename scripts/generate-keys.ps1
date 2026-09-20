$encKey = [Convert]::ToBase64String((1..32 | ForEach-Object { [byte](Get-Random -Max 256) }))
$jwtKey = [Convert]::ToBase64String((1..32 | ForEach-Object { [byte](Get-Random -Max 256) }))

Write-Host "ENCRYPTION_KEY=$encKey"
Write-Host "JWT_SECRET=$jwtKey"
Write-Host ""
Write-Host "Copy these into your backend/.env file."